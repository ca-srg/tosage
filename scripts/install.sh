#!/bin/bash
# tosage Automated Installation Script
# This script downloads the latest tosage DMG, installs the application,
# and creates a configuration file through interactive prompts.

set -e # Exit on error
set -u # Exit on undefined variable
set -o pipefail # Exit on pipe failure

# Color codes for output
readonly COLOR_RESET='\033[0m'
readonly COLOR_RED='\033[0;31m'
readonly COLOR_GREEN='\033[0;32m'
readonly COLOR_YELLOW='\033[0;33m'
readonly COLOR_BLUE='\033[0;34m'
readonly COLOR_MAGENTA='\033[0;35m'
readonly COLOR_CYAN='\033[0;36m'

# Global variables
readonly GITHUB_REPO="ca-srg/tosage"
readonly APP_NAME="tosage"
readonly CONFIG_DIR="$HOME/.config/tosage"
readonly CONFIG_FILE="$CONFIG_DIR/config.json"
readonly TEMP_DIR="$(mktemp -d)"
readonly REQUIRED_COMMANDS=("curl" "jq" "hdiutil" "diskutil")

# Cleanup function
cleanup() {
    local exit_code=$?
    
    # Unmount DMG if mounted
    if [[ -n "${DMG_MOUNT_POINT:-}" ]] && [[ -d "$DMG_MOUNT_POINT" ]]; then
        log_info "Unmounting DMG..."
        hdiutil detach "$DMG_MOUNT_POINT" -quiet 2>/dev/null || true
    fi
    
    # Remove temporary directory
    if [[ -d "$TEMP_DIR" ]]; then
        log_info "Cleaning up temporary files..."
        rm -rf "$TEMP_DIR"
    fi
    
    if [[ $exit_code -ne 0 ]]; then
        log_error "Installation failed. Please check the error messages above."
    fi
    
    exit $exit_code
}

# Set trap for cleanup
trap cleanup EXIT INT TERM

# Helper function for reading input (works with pipes)
safe_read() {
    local var_name="$1"
    local read_args="${@:2}"
    
    if [[ -t 0 ]]; then
        # Standard input is a terminal
        read $read_args "$var_name"
    else
        # Standard input is not a terminal (e.g., piped)
        # Use /dev/tty for user interaction
        read $read_args "$var_name" < /dev/tty
    fi
}

# Logging functions
log_info() {
    echo -e "${COLOR_BLUE}[INFO]${COLOR_RESET} $1"
}

log_success() {
    echo -e "${COLOR_GREEN}[SUCCESS]${COLOR_RESET} $1"
}

log_warning() {
    echo -e "${COLOR_YELLOW}[WARNING]${COLOR_RESET} $1"
}

log_error() {
    echo -e "${COLOR_RED}[ERROR]${COLOR_RESET} $1" >&2
}

log_prompt() {
    echo -e "${COLOR_CYAN}[PROMPT]${COLOR_RESET} $1"
}

# Print banner
print_banner() {
    echo -e "${COLOR_MAGENTA}"
    echo "╔════════════════════════════════════════╗"
    echo "║      tosage Automated Installer        ║"
    echo "║       Token Usage Tracker              ║"
    echo "╚════════════════════════════════════════╝"
    echo -e "${COLOR_RESET}"
}

# Check system requirements
check_requirements() {
    log_info "Checking system requirements..."
    
    # Check if running on macOS
    if [[ "$(uname)" != "Darwin" ]]; then
        log_error "This installer only supports macOS."
        exit 1
    fi
    
    # Check macOS version (require 10.15 or later)
    local macos_version=$(sw_vers -productVersion)
    local major_version=$(echo "$macos_version" | cut -d. -f1)
    local minor_version=$(echo "$macos_version" | cut -d. -f2)
    
    if [[ "$major_version" -lt 10 ]] || ([[ "$major_version" -eq 10 ]] && [[ "$minor_version" -lt 15 ]]); then
        log_error "macOS 10.15 (Catalina) or later is required. Current version: $macos_version"
        exit 1
    fi
    
    # Check required commands
    local missing_commands=()
    for cmd in "${REQUIRED_COMMANDS[@]}"; do
        if ! command -v "$cmd" &> /dev/null; then
            missing_commands+=("$cmd")
        fi
    done
    
    if [[ ${#missing_commands[@]} -gt 0 ]]; then
        log_error "Missing required commands: ${missing_commands[*]}"
        if [[ " ${missing_commands[*]} " =~ " jq " ]]; then
            log_info "To install jq: brew install jq"
        fi
        exit 1
    fi
    
    # Check disk space (require at least 100MB)
    local available_space=$(df -k /Applications 2>/dev/null | tail -1 | awk '{print $4}')
    if [[ -z "$available_space" ]] || [[ "$available_space" -lt 102400 ]]; then
        log_error "Insufficient disk space. At least 100MB required in /Applications."
        exit 1
    fi
    
    # Check internet connectivity with better error handling
    log_info "Checking internet connectivity..."
    local test_urls=("https://api.github.com" "https://github.com")
    local connectivity_ok=false
    
    for url in "${test_urls[@]}"; do
        if curl -s -I --connect-timeout 5 "$url" > /dev/null 2>&1; then
            connectivity_ok=true
            break
        fi
    done
    
    if [[ "$connectivity_ok" != "true" ]]; then
        log_error "No internet connection or GitHub is unreachable."
        log_info "Please check your internet connection and any proxy settings."
        
        # Check for proxy environment variables
        if [[ -n "${HTTP_PROXY:-}" ]] || [[ -n "${HTTPS_PROXY:-}" ]]; then
            log_info "Proxy detected. Ensure proxy settings are correct."
        fi
        
        exit 1
    fi
    
    # Check if running with sudo (warn if yes)
    if [[ "$EUID" -eq 0 ]]; then
        log_warning "Running as root/sudo is not recommended for the initial setup."
        log_warning "The installer will prompt for sudo only when needed."
    fi
    
    log_success "All system requirements met."
}

# Detect system architecture
detect_architecture() {
    local arch=$(uname -m)
    case "$arch" in
        arm64|aarch64)
            echo "arm64"
            ;;
        x86_64)
            echo "x86_64"
            ;;
        *)
            log_error "Unsupported architecture: $arch"
            exit 1
            ;;
    esac
}

# Fetch latest release information from GitHub
fetch_latest_release() {
    log_info "Fetching latest release information from GitHub..."
    
    local api_url="https://api.github.com/repos/$GITHUB_REPO/releases/latest"
    local response
    local http_code
    
    # Make API request with timeout and capture response
    local curl_opts=(-s -w "\n%{http_code}" --connect-timeout 10 --max-time 30)
    curl_opts+=(-H "Accept: application/vnd.github.v3+json")
    
    # Add proxy support if environment variables are set
    if [[ -n "${HTTPS_PROXY:-}" ]]; then
        curl_opts+=(--proxy "$HTTPS_PROXY")
    elif [[ -n "${HTTP_PROXY:-}" ]]; then
        curl_opts+=(--proxy "$HTTP_PROXY")
    fi
    
    response=$(curl "${curl_opts[@]}" "$api_url" 2>&1) || {
        log_error "Failed to connect to GitHub API"
        if [[ -n "${HTTPS_PROXY:-}" ]] || [[ -n "${HTTP_PROXY:-}" ]]; then
            log_info "Proxy is configured. Check proxy settings and connectivity."
        fi
        exit 1
    }
    
    # Extract HTTP code (last line)
    http_code=$(echo "$response" | tail -1)
    # Remove last line to get response body
    response=$(echo "$response" | sed '$d')
    
    # Check for rate limiting
    if [[ "$http_code" == "403" ]]; then
        log_error "GitHub API rate limit exceeded. Please try again later."
        exit 1
    fi
    
    # Check for success
    if [[ "$http_code" != "200" ]]; then
        log_error "Failed to fetch release information. HTTP code: $http_code"
        exit 1
    fi
    
    # Parse release information
    RELEASE_VERSION=$(echo "$response" | jq -r '.tag_name // empty') || {
        log_error "Failed to parse release version"
        exit 1
    }
    
    if [[ -z "$RELEASE_VERSION" ]]; then
        log_error "No releases found for $GITHUB_REPO"
        exit 1
    fi
    
    # Get architecture-specific DMG URL
    local arch=$(detect_architecture)
    # Keep the 'v' in the version for the DMG filename
    local dmg_name="tosage-${RELEASE_VERSION}-darwin-${arch}.dmg"
    
    DMG_DOWNLOAD_URL=$(echo "$response" | jq -r ".assets[] | select(.name == \"$dmg_name\") | .browser_download_url // empty") || {
        log_error "Failed to parse DMG download URL"
        exit 1
    }
    
    if [[ -z "$DMG_DOWNLOAD_URL" ]]; then
        log_error "No DMG found for architecture: $arch"
        log_error "Looking for: $dmg_name"
        exit 1
    fi
    
    DMG_SIZE=$(echo "$response" | jq -r ".assets[] | select(.name == \"$dmg_name\") | .size // 0") || DMG_SIZE=0
    
    log_success "Found latest release: $RELEASE_VERSION"
    log_info "DMG URL: $DMG_DOWNLOAD_URL"
    log_info "DMG Size: $(( DMG_SIZE / 1024 / 1024 )) MB"
    
    # Export variables for use in other functions
    export RELEASE_VERSION
    export DMG_DOWNLOAD_URL
    export DMG_SIZE
    export DMG_NAME="$dmg_name"
}

download_dmg() {
    log_info "Downloading DMG file..."
    
    if [[ -z "${DMG_DOWNLOAD_URL:-}" ]]; then
        log_error "DMG download URL not set. Please run fetch_latest_release first."
        exit 1
    fi
    
    local dmg_path="$TEMP_DIR/$DMG_NAME"
    local max_retries=3
    local retry_count=0
    local download_success=false
    
    while [[ $retry_count -lt $max_retries ]] && [[ "$download_success" == "false" ]]; do
        if [[ $retry_count -gt 0 ]]; then
            log_warning "Retrying download... (Attempt $((retry_count + 1))/$max_retries)"
            sleep 2
        fi
        
        # Download with progress bar
        log_info "Downloading from: $DMG_DOWNLOAD_URL"
        
        # Build curl options with proxy support
        local download_opts=(-L --fail --progress-bar)
        download_opts+=(--connect-timeout 30 --max-time 600)
        download_opts+=(-o "$dmg_path")
        
        # Add proxy support if environment variables are set
        if [[ -n "${HTTPS_PROXY:-}" ]]; then
            download_opts+=(--proxy "$HTTPS_PROXY")
        elif [[ -n "${HTTP_PROXY:-}" ]]; then
            download_opts+=(--proxy "$HTTP_PROXY")
        fi
        
        if curl "${download_opts[@]}" "$DMG_DOWNLOAD_URL"; then
            download_success=true
        else
            local curl_exit_code=$?
            retry_count=$((retry_count + 1))
            if [[ $retry_count -lt $max_retries ]]; then
                log_warning "Download failed (exit code: $curl_exit_code). Will retry..."
                # Remove partial download
                rm -f "$dmg_path" 2>/dev/null || true
            fi
        fi
    done
    
    if [[ "$download_success" != "true" ]]; then
        log_error "Failed to download DMG after $max_retries attempts"
        exit 1
    fi
    
    # Verify download
    if [[ ! -f "$dmg_path" ]]; then
        log_error "Downloaded file not found at: $dmg_path"
        exit 1
    fi
    
    # Check file size
    local actual_size=$(stat -f%z "$dmg_path" 2>/dev/null || stat -c%s "$dmg_path" 2>/dev/null)
    if [[ -n "$DMG_SIZE" ]] && [[ "$DMG_SIZE" -gt 0 ]]; then
        if [[ "$actual_size" != "$DMG_SIZE" ]]; then
            log_warning "Downloaded file size ($actual_size) does not match expected size ($DMG_SIZE)"
        fi
    fi
    
    # Verify DMG format
    log_info "Verifying DMG file integrity..."
    if ! hdiutil verify "$dmg_path" -quiet 2>/dev/null; then
        log_error "DMG verification failed. The file may be corrupted."
        exit 1
    fi
    
    log_success "DMG downloaded and verified successfully"
    
    # Export path for use in installation
    export DMG_PATH="$dmg_path"
}

install_application() {
    log_info "Installing application..."
    
    if [[ -z "${DMG_PATH:-}" ]] || [[ ! -f "$DMG_PATH" ]]; then
        log_error "DMG file not found. Please run download_dmg first."
        exit 1
    fi
    
    # Mount DMG
    log_info "Mounting DMG..."
    local mount_output
    mount_output=$(hdiutil attach "$DMG_PATH" -nobrowse 2>&1) || {
        log_error "Failed to mount DMG: $mount_output"
        exit 1
    }
    
    # Extract mount point - look for line containing /Volumes/
    DMG_MOUNT_POINT=$(echo "$mount_output" | grep "/Volumes/" | tail -1 | awk '{for(i=1;i<=NF;i++) if ($i ~ /^\/Volumes\//) print $i}' | tail -1)
    
    # If that didn't work, try a simpler approach
    if [[ -z "$DMG_MOUNT_POINT" ]]; then
        DMG_MOUNT_POINT=$(echo "$mount_output" | grep -o '/Volumes/[^[:space:]]*' | tail -1)
    fi
    
    if [[ -z "$DMG_MOUNT_POINT" ]] || [[ ! -d "$DMG_MOUNT_POINT" ]]; then
        log_error "Failed to find mount point in output:"
        echo "$mount_output" >&2
        exit 1
    fi
    
    log_info "DMG mounted at: $DMG_MOUNT_POINT"
    
    # Check if app exists in DMG
    local app_in_dmg="$DMG_MOUNT_POINT/tosage.app"
    if [[ ! -d "$app_in_dmg" ]]; then
        log_error "tosage.app not found in DMG"
        hdiutil detach "$DMG_MOUNT_POINT" -quiet 2>/dev/null || true
        exit 1
    fi
    
    # Check for existing installation
    local target_app="/Applications/tosage.app"
    if [[ -d "$target_app" ]]; then
        log_warning "Existing installation found at $target_app"
        
        # Ask user for confirmation
        echo ""
        log_prompt "Do you want to replace the existing installation? (y/N)"
        safe_read replace_answer -r -n 1
        echo ""
        
        if [[ ! "$replace_answer" =~ ^[Yy]$ ]]; then
            log_info "Installation cancelled by user."
            hdiutil detach "$DMG_MOUNT_POINT" -quiet 2>/dev/null || true
            exit 0
        fi
        
        # Backup existing installation
        local backup_name="/Applications/tosage.app.backup.$(date +%Y%m%d%H%M%S)"
        log_info "Backing up existing installation to: $backup_name"
        
        if ! sudo mv "$target_app" "$backup_name"; then
            log_error "Failed to backup existing installation"
            hdiutil detach "$DMG_MOUNT_POINT" -quiet 2>/dev/null || true
            exit 1
        fi
    fi
    
    # Copy app to Applications
    log_info "Copying tosage.app to Applications folder..."
    if ! sudo cp -R "$app_in_dmg" "/Applications/"; then
        log_error "Failed to copy application to /Applications"
        hdiutil detach "$DMG_MOUNT_POINT" -quiet 2>/dev/null || true
        exit 1
    fi
    
    # Remove quarantine attributes
    log_info "Removing macOS security restrictions..."
    if ! sudo xattr -cr "$target_app" 2>/dev/null; then
        log_warning "Failed to remove quarantine attributes (this is normal if not quarantined)"
    fi
    
    # Apply ad-hoc signature
    log_info "Applying ad-hoc signature..."
    if ! sudo codesign --force --sign - "$target_app" 2>/dev/null; then
        log_warning "Failed to apply ad-hoc signature (application may still work)"
    fi
    
    # Unmount DMG
    log_info "Unmounting DMG..."
    if ! hdiutil detach "$DMG_MOUNT_POINT" -quiet 2>/dev/null; then
        log_warning "Failed to unmount DMG cleanly (you may need to manually eject)"
    fi
    
    # Clear mount point variable
    unset DMG_MOUNT_POINT
    
    # Verify installation
    if [[ -d "$target_app" ]]; then
        log_success "Application installed successfully!"
        log_info "Location: $target_app"
    else
        log_error "Installation verification failed"
        exit 1
    fi
}

collect_configuration() {
    log_info "Collecting configuration settings..."
    
    echo ""
    log_info "Please provide your configuration values."
    log_info "Press Enter to skip optional fields."
    echo ""
    
    # Initialize configuration variables
    local prometheus_url=""
    local prometheus_username=""
    local prometheus_password=""
    local prometheus_host_label=""
    local prometheus_interval="600"
    local prometheus_timeout="30"
    local promtail_url=""
    local promtail_username=""
    local promtail_password=""
    
    # Prometheus configuration
    log_prompt "=== Prometheus Configuration ==="
    
    # Remote Write URL (required)
    while true; do
        log_prompt "Prometheus Remote Write URL (required): "
        safe_read prometheus_url -r
        
        if [[ -z "$prometheus_url" ]]; then
            log_error "Prometheus URL is required"
            continue
        fi
        
        # Basic URL validation
        if [[ ! "$prometheus_url" =~ ^https?:// ]]; then
            log_error "Invalid URL format. Please include http:// or https://"
            continue
        fi
        
        break
    done
    
    # Username (required)
    while true; do
        log_prompt "Prometheus Username (required): "
        safe_read prometheus_username -r
        
        if [[ -z "$prometheus_username" ]]; then
            log_error "Prometheus username is required"
            continue
        fi
        
        break
    done
    
    # Password (required, hidden input)
    while true; do
        log_prompt "Prometheus Password (required, input hidden): "
        safe_read prometheus_password -r -s
        echo "" # New line after hidden input
        
        if [[ -z "$prometheus_password" ]]; then
            log_error "Prometheus password is required"
            continue
        fi
        
        break
    done
    
    # Host label (optional)
    log_prompt "Host Label (optional, press Enter to skip): "
    safe_read prometheus_host_label -r
    
    # Interval (optional, with default)
    log_prompt "Metrics Interval in seconds (optional, default: 600): "
    safe_read input_interval -r
    if [[ -n "$input_interval" ]] && [[ "$input_interval" =~ ^[0-9]+$ ]]; then
        prometheus_interval="$input_interval"
    fi
    
    # Timeout (optional, with default)
    log_prompt "Request Timeout in seconds (optional, default: 30): "
    safe_read input_timeout -r
    if [[ -n "$input_timeout" ]] && [[ "$input_timeout" =~ ^[0-9]+$ ]]; then
        prometheus_timeout="$input_timeout"
    fi
    
    echo ""
    log_prompt "=== Promtail Configuration (Optional) ==="
    log_info "Leave blank to skip Promtail logging configuration"
    
    # Promtail URL (optional)
    log_prompt "Promtail Logs URL (optional): "
    safe_read promtail_url -r
    
    # Only ask for credentials if URL is provided
    if [[ -n "$promtail_url" ]]; then
        # Basic URL validation
        if [[ ! "$promtail_url" =~ ^https?:// ]]; then
            log_warning "Invalid Promtail URL format. Skipping Promtail configuration."
            promtail_url=""
        else
            # Username
            log_prompt "Promtail Username (optional): "
            safe_read promtail_username -r
            
            # Password (hidden input)
            if [[ -n "$promtail_username" ]]; then
                log_prompt "Promtail Password (optional, input hidden): "
                safe_read promtail_password -r -s
                echo "" # New line after hidden input
            fi
        fi
    fi
    
    # Export configuration for use in create_config_file
    export CONFIG_PROMETHEUS_URL="$prometheus_url"
    export CONFIG_PROMETHEUS_USERNAME="$prometheus_username"
    export CONFIG_PROMETHEUS_PASSWORD="$prometheus_password"
    export CONFIG_PROMETHEUS_HOST_LABEL="$prometheus_host_label"
    export CONFIG_PROMETHEUS_INTERVAL="$prometheus_interval"
    export CONFIG_PROMETHEUS_TIMEOUT="$prometheus_timeout"
    export CONFIG_PROMTAIL_URL="$promtail_url"
    export CONFIG_PROMTAIL_USERNAME="$promtail_username"
    export CONFIG_PROMTAIL_PASSWORD="$promtail_password"
    
    echo ""
    log_success "Configuration values collected successfully"
}

create_config_file() {
    log_info "Creating configuration file..."
    
    # Check if configuration values are set
    if [[ -z "${CONFIG_PROMETHEUS_URL:-}" ]]; then
        log_error "Configuration values not set. Please run collect_configuration first."
        exit 1
    fi
    
    # Create config directory if it doesn't exist
    if [[ ! -d "$CONFIG_DIR" ]]; then
        log_info "Creating configuration directory: $CONFIG_DIR"
        if ! mkdir -p "$CONFIG_DIR"; then
            log_error "Failed to create configuration directory"
            exit 1
        fi
    fi
    
    # Check for existing config file
    if [[ -f "$CONFIG_FILE" ]]; then
        log_warning "Existing configuration file found: $CONFIG_FILE"
        
        # Ask user for confirmation
        echo ""
        log_prompt "Do you want to overwrite the existing configuration? (y/N)"
        safe_read overwrite_answer -r -n 1
        echo ""
        
        if [[ ! "$overwrite_answer" =~ ^[Yy]$ ]]; then
            log_info "Configuration file creation cancelled by user."
            log_info "Existing configuration preserved."
            return 0
        fi
        
        # Backup existing configuration
        local backup_file="$CONFIG_FILE.backup.$(date +%Y%m%d%H%M%S)"
        log_info "Backing up existing configuration to: $backup_file"
        
        if ! cp "$CONFIG_FILE" "$backup_file"; then
            log_error "Failed to backup existing configuration"
            exit 1
        fi
    fi
    
    # Generate JSON configuration
    local config_json
    config_json=$(cat <<EOF
{
  "prometheus": {
    "remote_write_url": "${CONFIG_PROMETHEUS_URL}",
    "username": "${CONFIG_PROMETHEUS_USERNAME}",
    "password": "${CONFIG_PROMETHEUS_PASSWORD}",
    "host_label": "${CONFIG_PROMETHEUS_HOST_LABEL}",
    "interval_seconds": ${CONFIG_PROMETHEUS_INTERVAL},
    "timeout_seconds": ${CONFIG_PROMETHEUS_TIMEOUT}
  },
  "logging": {
    "promtail": {
      "url": "${CONFIG_PROMTAIL_URL}",
      "username": "${CONFIG_PROMTAIL_USERNAME}",
      "password": "${CONFIG_PROMTAIL_PASSWORD}"
    }
  }
}
EOF
)
    
    # Validate JSON before writing
    if ! echo "$config_json" | jq '.' > /dev/null 2>&1; then
        log_error "Generated configuration is not valid JSON"
        log_error "This is likely an internal error. Please report this issue."
        exit 1
    fi
    
    # Write configuration file with atomic operation
    local temp_config="${CONFIG_FILE}.tmp.$$"
    if ! echo "$config_json" | jq '.' > "$temp_config" 2>/dev/null; then
        log_error "Failed to write configuration file"
        rm -f "$temp_config" 2>/dev/null || true
        exit 1
    fi
    
    # Move temp file to final location (atomic)
    if ! mv "$temp_config" "$CONFIG_FILE" 2>/dev/null; then
        log_error "Failed to move configuration file to final location"
        rm -f "$temp_config" 2>/dev/null || true
        exit 1
    fi
    
    # Set secure permissions (readable only by owner)
    if ! chmod 600 "$CONFIG_FILE"; then
        log_error "Failed to set secure permissions on configuration file"
        exit 1
    fi
    
    log_success "Configuration file created successfully!"
    log_info "Location: $CONFIG_FILE"
    
    # Display configuration summary (without sensitive data)
    echo ""
    log_info "Configuration Summary:"
    log_info "  Prometheus URL: ${CONFIG_PROMETHEUS_URL}"
    log_info "  Prometheus Username: ${CONFIG_PROMETHEUS_USERNAME}"
    log_info "  Host Label: ${CONFIG_PROMETHEUS_HOST_LABEL:-<not set>}"
    log_info "  Metrics Interval: ${CONFIG_PROMETHEUS_INTERVAL}s"
    
    if [[ -n "$CONFIG_PROMTAIL_URL" ]]; then
        log_info "  Promtail URL: ${CONFIG_PROMTAIL_URL}"
        log_info "  Promtail Username: ${CONFIG_PROMTAIL_USERNAME:-<not set>}"
    else
        log_info "  Promtail: <not configured>"
    fi
}

# Signal handler for graceful interruption
handle_interrupt() {
    echo ""
    log_warning "Installation interrupted by user"
    cleanup
    exit 130
}

# Main installation function
main() {
    # Set up interrupt handler
    trap handle_interrupt INT TERM
    
    print_banner
    
    log_info "Starting tosage installation process..."
    log_info "Press Ctrl+C at any time to cancel"
    echo ""
    
    # Check system requirements
    check_requirements
    
    # Detect architecture
    local arch=$(detect_architecture)
    log_info "Detected architecture: $arch"
    
    # Fetch latest release
    fetch_latest_release
    
    # Download DMG
    download_dmg
    
    # Install application
    install_application
    
    # Ask if user wants to configure now
    echo ""
    log_prompt "Would you like to configure tosage now? (Y/n)"
    safe_read configure_answer -r -n 1
    echo ""
    
    if [[ "$configure_answer" =~ ^[Nn]$ ]]; then
        log_info "Skipping configuration. You can configure tosage later by:"
        log_info "  1. Running this installer again"
        log_info "  2. Manually creating ~/.config/tosage/config.json"
        echo ""
        log_success "Application installed successfully!"
        exit 0
    fi
    
    # Collect configuration
    collect_configuration
    
    # Create config file
    create_config_file
    
    echo ""
    log_success "Installation completed successfully!"
    echo ""
    log_info "tosage has been installed and configured!"
    echo ""
    log_info "Next steps:"
    log_info "  1. Open tosage from Applications folder or Launchpad"
    log_info "  2. Or run from Terminal: open /Applications/tosage.app"
    log_info "  3. The app will start monitoring token usage automatically"
    echo ""
    log_info "Configuration file location: $CONFIG_FILE"
    log_info "To modify settings, edit the config file or re-run this installer"
    echo ""
}

# Run main function
main "$@"
# DMG Code Signing and Notarization Guide

This guide explains how to set up code signing for tosage DMG distribution.

## Prerequisites

- macOS development environment
- Apple Developer account (for notarization)
- Developer ID Application certificate

## Local Development

### 1. Certificate Setup

Ensure you have a valid Developer ID Application certificate in your keychain:

```bash
security find-identity -v -p codesigning
```

### 2. Build and Sign

```bash
# Build DMG with signing
make dmg-arm64

# Build without signing (if no certificate)
SIGNING_IDENTITY="" make dmg-arm64
```

### 3. Environment Variables

- `SIGNING_IDENTITY`: Certificate name (default: "Developer ID Application")
- `TEAM_ID`: Apple Team ID (auto-detected if not set)
- `APPLE_ID`: Apple ID for notarization
- `APP_PASSWORD`: App-specific password for notarization

## GitHub Actions Setup

### Required Secrets

Configure these secrets in your GitHub repository:

1. **MACOS_CERTIFICATE**: Base64-encoded .p12 certificate
   ```bash
   base64 -i certificate.p12 | pbcopy
   ```

2. **MACOS_CERTIFICATE_PWD**: Certificate password

3. **SIGNING_IDENTITY**: Certificate common name (optional)
   - Example: "Developer ID Application: Your Name (TEAM_ID)"

4. **APPLE_TEAM_ID**: Your Apple Developer Team ID

5. **APPLE_ID**: Apple ID for notarization (optional)

6. **APP_PASSWORD**: App-specific password (optional)

### Certificate Export

1. Open Keychain Access
2. Find your Developer ID Application certificate
3. Export as .p12 file with password
4. Convert to base64 for GitHub secret

## Verification

### Local Verification

```bash
# Verify signature
codesign --verify --deep --strict tosage.app
spctl -a -t exec -vv tosage.app

# Verify DMG
spctl -a -t open --context context:primary-signature -vv tosage.dmg
```

### Gatekeeper Test

```bash
# Test Gatekeeper acceptance
sudo spctl --master-disable
sudo spctl --master-enable
spctl -a -t open tosage.dmg
```

## Troubleshooting

### Common Issues

1. **Certificate not found**
   - Check certificate name matches exactly
   - Ensure certificate is valid and not expired

2. **Notarization fails**
   - Verify Apple ID and app-specific password
   - Check entitlements are correct

3. **Gatekeeper rejection**
   - Ensure proper code signing
   - Check notarization status

### Debug Commands

```bash
# List certificates
security find-identity -v -p codesigning

# Check signature details
codesign -dv --verbose=4 tosage.app

# View entitlements
codesign -d --entitlements - tosage.app
```
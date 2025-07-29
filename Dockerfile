# Use debian-slim as base image for gcloud CLI support
FROM debian:12-slim

# Build argument for architecture
ARG TARGETARCH

# Install dependencies and gcloud CLI
RUN apt-get update && apt-get install -y \
    curl \
    python3 \
    python3-crcmod \
    ca-certificates \
    gnupg \
    lsb-release \
    && echo "deb [signed-by=/usr/share/keyrings/cloud.google.gpg] https://packages.cloud.google.com/apt cloud-sdk main" | tee -a /etc/apt/sources.list.d/google-cloud-sdk.list \
    && curl https://packages.cloud.google.com/apt/doc/apt-key.gpg | apt-key --keyring /usr/share/keyrings/cloud.google.gpg add - \
    && apt-get update && apt-get install -y google-cloud-cli \
    && apt-get clean \
    && rm -rf /var/lib/apt/lists/*

# Create non-root user
RUN useradd -r -u 65532 -g nogroup nonroot

# Copy pre-built binary based on architecture
COPY tosage-linux-${TARGETARCH} /tosage
RUN chmod +x /tosage

# Set PATH to include gcloud CLI
ENV PATH="/usr/lib/google-cloud-sdk/bin:${PATH}"

# Use nonroot user
USER nonroot:nogroup

# Set labels for OCI compliance
LABEL org.opencontainers.image.source="https://github.com/ca-srg/tosage" \
      org.opencontainers.image.description="Token usage tracking for Claude Code and Cursor" \
      org.opencontainers.image.licenses="MIT" \
      org.opencontainers.image.authors="CA-SRG"

# Set entrypoint
ENTRYPOINT ["/tosage"]
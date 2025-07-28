# Use distroless as base image
FROM gcr.io/distroless/static:nonroot

# Build argument for architecture
ARG TARGETARCH

# Copy pre-built binary based on architecture
COPY tosage-linux-${TARGETARCH} /tosage

# Use nonroot user
USER nonroot:nonroot

# Set labels for OCI compliance
LABEL org.opencontainers.image.source="https://github.com/ca-srg/tosage" \
      org.opencontainers.image.description="Token usage tracking for Claude Code and Cursor" \
      org.opencontainers.image.licenses="MIT" \
      org.opencontainers.image.authors="CA-SRG"

# Set entrypoint
ENTRYPOINT ["/tosage"]

# Default command (CLI mode)
CMD ["--mode", "cli"]
# DMG Build Notes

## CGO Requirement

The tosage application uses the `systray` package which requires CGO (C bindings) on macOS. This means:

### Local Building

1. **Native Architecture Only**: You can only build for your current architecture
   - On Apple Silicon (M1/M2): Only ARM64 builds are possible
   - On Intel Macs: Only AMD64 builds are possible

2. **Cross-compilation**: Not supported with CGO enabled
   - Cannot build AMD64 on ARM64 Mac
   - Cannot build ARM64 on Intel Mac

### Building DMG

```bash
# On Apple Silicon (ARM64)
make dmg-arm64

# On Intel Mac (AMD64)
make dmg-amd64
```

### GitHub Actions

The release workflow handles both architectures by:
- Using macOS runners that support the target architecture
- Building with CGO enabled for proper systray support

### Workarounds for Cross-Architecture Builds

If you need to create DMGs for both architectures:

1. **Use GitHub Actions**: Push a tag to trigger the release workflow
2. **Use Two Macs**: Build on both Intel and Apple Silicon machines
3. **Universal Binary**: Not currently implemented but could be added

### Testing Without CGO

For testing purposes only, you can disable systray functionality:
```bash
CGO_ENABLED=0 go build -tags "nocgo" .
```
Note: This will disable the system tray functionality.
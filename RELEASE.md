# BEM Server - Release Process

This document describes how to create and publish releases of the BigFix Enterprise Mobile (BEM) server.

## Branching Strategy

We follow **GitHub Flow** for version control:

- **main branch**: Production-ready code
- **Feature branches**: `feature/feature-name` or `bugfix/issue-description`
- **Releases**: Tagged commits on main branch

### Creating Feature/Bug Fix Branches

```bash
# Create feature branch
git checkout -b feature/add-new-endpoint

# Create bugfix branch
git checkout -b bugfix/fix-cache-leak

# Make changes, commit, and push
git add .
git commit -m "Add new /status endpoint"
git push origin feature/add-new-endpoint

# Create PR to main, get review, merge
```

## Versioning

We use **Semantic Versioning** (SemVer): `MAJOR.MINOR.PATCH`

- **MAJOR**: Breaking changes (e.g., API changes requiring client updates)
- **MINOR**: New features (backward compatible)
- **PATCH**: Bug fixes only

### Examples:
- `1.0.0` → `1.0.1` - Bug fix release
- `1.0.1` → `1.1.0` - New feature added (backward compatible)
- `1.1.0` → `2.0.0` - Breaking change (API redesign)

## Release Checklist

### 1. Pre-Release

- [ ] All tests pass: `make test`
- [ ] Code is merged to `main` branch
- [ ] README.md is up to date
- [ ] CHANGELOG has been updated with release notes
- [ ] No critical bugs outstanding

### 2. Version Bump

Edit the `VERSION` file to reflect the new version:

```bash
# For a minor release (1.0.0 → 1.1.0)
echo "1.1.0" > VERSION
git add VERSION
git commit -m "Bump version to 1.1.0"
git push origin main
```

### 3. Build Release Binaries

Build for all platforms:

```bash
# Build all platform binaries
make release

# Create distribution packages
make packages
```

This creates:
- **Binary archives** in `dist/`:
  - `bigfix-mobile-enterprise-1.1.0-linux-amd64.tar.gz`
  - `bigfix-mobile-enterprise-1.1.0-linux-arm64.tar.gz`
  - `bigfix-mobile-enterprise-1.1.0-darwin-amd64.tar.gz`
  - `bigfix-mobile-enterprise-1.1.0-darwin-arm64.tar.gz`
  - `bigfix-mobile-enterprise-1.1.0-windows-amd64.zip`
- **Source archives**:
  - `bigfix-mobile-enterprise-1.1.0-source.tar.gz`
  - `bigfix-mobile-enterprise-1.1.0-source.zip`

### 4. Test the Release Build

```bash
# Test the binary
./build/bem-linux-amd64 --version

# Expected output:
# BEM Server 1.1.0 (built 2025-10-24T14:30:15Z, commit a1b2c3d)
```

### 5. Create Git Tag

```bash
# Create annotated tag
git tag -a v1.1.0 -m "Release version 1.1.0

New features:
- Added cache pagination to CLI
- Enhanced summary command with hit/miss statistics

Bug fixes:
- Fixed TLS connection logging

See CHANGELOG.md for full details."

# Push tag to remote
git push origin v1.1.0
```

### 6. Create GitHub Release

1. Go to https://github.com/tubalcaine/bigfix-mobile-enterprise/releases
2. Click "Draft a new release"
3. Select tag: `v1.1.0`
4. Release title: `BEM Server v1.1.0`
5. Description: Copy from CHANGELOG.md or git tag message
6. Attach release artifacts from `dist/` directory:
   - All `.tar.gz` files
   - All `.zip` files
7. Click "Publish release"

## Hotfix Process (Emergency Bug Fixes)

For critical bugs in production:

```bash
# Work on main branch or create hotfix branch
git checkout main
git checkout -b hotfix/critical-security-fix

# Make the fix
git add .
git commit -m "Fix critical security vulnerability in auth"

# Merge to main
git checkout main
git merge hotfix/critical-security-fix
git push origin main

# Bump PATCH version
echo "1.1.1" > VERSION
git add VERSION
git commit -m "Bump version to 1.1.1 (hotfix)"
git push origin main

# Build and release (steps 3-6 above)
make release
make packages
git tag -a v1.1.1 -m "Hotfix: Security vulnerability in auth"
git push origin v1.1.1

# Create GitHub release with hotfix assets
```

## Build System Reference

### Makefile Targets

```bash
make build          # Build for current platform only
make release        # Build for all platforms (Linux, macOS, Windows)
make packages       # Create release archives (.tar.gz, .zip)
make test           # Run tests
make clean          # Remove build artifacts
make version        # Display current version info
make install        # Install to /usr/local/bin (requires sudo)
make uninstall      # Remove from /usr/local/bin
make help           # Show all available targets
```

### Version Information

Version information is injected at build time:
- **VERSION file**: Source of truth for version number
- **BuildDate**: Automatically set to build timestamp (UTC)
- **GitCommit**: Automatically set to short commit hash

Display in binary:
```bash
./bem --version
# Output: BEM Server 1.1.0 (built 2025-10-24T14:30:15Z, commit a1b2c3d)
```

## Post-Release

### Update Documentation

- [ ] Announce release in README.md if significant
- [ ] Update deployment documentation if config changes
- [ ] Notify users via appropriate channels

### Verify Release

- [ ] Download release assets from GitHub
- [ ] Extract and test binaries on each platform
- [ ] Verify version matches: `./bem --version`

## Troubleshooting

### Build Fails

```bash
# Clean and rebuild
make clean
make build

# Check Go version (requires 1.18+)
go version

# Update dependencies
go mod tidy
go mod vendor
```

### Missing Git Commit Hash

If `git rev-parse` fails (e.g., not a git repository), commit hash shows as "unknown". Ensure you're in a git repository:

```bash
git status
```

### Version Mismatch

If `--version` shows "dev" instead of expected version:
- Ensure VERSION file exists and contains valid version
- Rebuild with `make clean && make build`
- Check LDFLAGS are being applied: look at Makefile

## Example Release Timeline

### Minor Release (1.0.0 → 1.1.0)

```
Week 1-2: Development
  - Feature branches merged to main
  - Tests passing

Week 3: Release Preparation
  Monday:    Feature freeze, final testing
  Tuesday:   Update docs, CHANGELOG, VERSION
  Wednesday: make packages, create tag, GitHub release
  Thursday:  Post-release verification
```

### Patch Release (1.1.0 → 1.1.1)

```
Same day as fix:
  - Fix bug on main
  - Update VERSION to 1.1.1
  - make packages
  - Create tag and GitHub release
```

## Version History

See CHANGELOG.md for detailed version history and release notes.

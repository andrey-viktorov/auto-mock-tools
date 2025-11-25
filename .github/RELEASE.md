# Release Process

## Automated Release Methods

### Method 1: Using VERSION file (Automatic)

The simplest way - just update the `VERSION` file and push to main:

```bash
echo "1.2.3" > VERSION
git add VERSION
git commit -m "Bump version to 1.2.3"
git push origin main
```

This will automatically:
1. Create a git tag `v1.2.3`
2. Trigger the release workflow
3. Build binaries for all platforms
4. Create a GitHub release with all artifacts

### Method 2: Manual trigger via GitHub UI

1. Go to **Actions** tab in GitHub
2. Select **Tag and Release** workflow
3. Click **Run workflow**
4. Enter version (e.g., `v1.2.3`)
5. Optionally mark as pre-release
6. Click **Run workflow**

This will create the tag and release in one step.

### Method 3: Push a tag manually

If you prefer manual tagging:

```bash
git tag -a v1.2.3 -m "Release v1.2.3"
git push origin v1.2.3
```

This will automatically trigger the release build.

## Release Artifacts

Each release includes:
- `auto-tools-vX.Y.Z-linux-amd64.tar.gz` - Linux binaries
- `auto-tools-vX.Y.Z-darwin-amd64.tar.gz` - macOS Intel binaries
- `auto-tools-vX.Y.Z-darwin-arm64.tar.gz` - macOS Apple Silicon binaries
- `auto-tools-vX.Y.Z-windows-amd64.zip` - Windows binaries
- `checksums.txt` - SHA256 checksums for all archives

## Pre-releases

To create a pre-release:
- Use version format like `v1.2.3-beta.1` or `v1.2.3-rc.1`
- Or use the GitHub UI method and check "Mark as pre-release"

## Version Format

Follow semantic versioning: `vMAJOR.MINOR.PATCH`
- MAJOR: Breaking changes
- MINOR: New features (backward compatible)
- PATCH: Bug fixes

Examples:
- `v1.0.0` - First stable release
- `v1.1.0` - New feature added
- `v1.1.1` - Bug fix
- `v2.0.0` - Breaking change
- `v1.2.0-beta.1` - Pre-release

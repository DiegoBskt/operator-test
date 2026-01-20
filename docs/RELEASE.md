# Release Process

This document describes the release process for the Cluster Assessment Operator.

## Overview

The release process is largely automated via GitHub Actions. A release is triggered by pushing a Git tag, which initiates the CI/CD pipeline to build and push all artifacts.

## Version Management

### Single Source of Truth

The project uses a `VERSION` file at the repository root as the single source of truth for versioning:

```bash
cat VERSION
# Output: 1.2.11
```

All other version references are derived from this file via Makefile variables.

### Version-Related Files

| File | Purpose | Updated By |
|------|---------|------------|
| `VERSION` | Single source of truth | Manual |
| `Makefile` | Reads VERSION, sets `IMG`, `BUNDLE_IMG`, etc. | Automatic |
| `config/manager/manager.yaml` | Operator deployment image | `make update-manifests` |
| `config/console-plugin/deployment.yaml` | Console plugin image | `make update-manifests` |
| `bundle/manifests/*.yaml` | OLM bundle CSV (name, version, images) | `make update-manifests` |
| `catalog-templates/*.yaml` | FBC catalog templates | `make update-catalogs` |
| `CHANGELOG.md` | Release notes | Manual |

> [!IMPORTANT]
> **Bundle CSV Version is Critical for OLM**
> 
> The bundle CSV file (`bundle/manifests/cluster-assessment-operator.clusterserviceversion.yaml`) must have:
> - `metadata.name: cluster-assessment-operator.v<VERSION>` 
> - `spec.version: <VERSION>`
> 
> If these don't match the release version, OLM catalogs will fail with errors like:
> `bundle "cluster-assessment-operator.vX.Y.Z" not found in any channel entries`
>
> The `make update-manifests` command now automatically updates these fields.

---

## Pre-Release Checklist

Before creating a release:

- [ ] All CI checks pass on `main` branch
- [ ] `CHANGELOG.md` is updated with new version section
- [ ] `VERSION` file is updated to new version
- [ ] All tests pass locally: `make test`
- [ ] Linting passes: `make lint`

---

## Release Procedure

### 1. Update Version

```bash
# Update VERSION file
echo "1.3.0" > VERSION

# Verify
make version
```

### 2. Update All Manifests

```bash
# This updates:
# - config/manager/manager.yaml (operator image)
# - config/console-plugin/deployment.yaml (console plugin image)
# - bundle/manifests/cluster-assessment-operator.clusterserviceversion.yaml:
#   - metadata.name (cluster-assessment-operator.v<VERSION>)
#   - spec.version (<VERSION>)
#   - containerImage annotation
#   - olm.skipRange annotation
make update-manifests
```

### 3. Update Catalog Templates

```bash
# This updates all catalog-templates/v4.*.yaml files
make update-catalogs
```

### 4. Regenerate and Validate Catalogs

```bash
# Generate catalogs from templates
make catalogs

# Validate all catalogs
make catalog-validate
```

### 5. Or Use the Combined Target

```bash
# Does all of the above in one command
make release-prep
```

### 6. Update CHANGELOG.md

Add a new section at the top of `CHANGELOG.md`:

```markdown
## [1.3.0] - YYYY-MM-DD

### Added
- New feature X

### Changed
- Updated Y

### Fixed
- Bug Z
```

### 7. Commit and Tag

```bash
# Commit all changes
git add -A
git commit -m "chore: prepare release v1.3.0"

# Create tag
git tag v1.3.0

# Push
git push origin main v1.3.0
```

---

## CI/CD Automation

When a tag is pushed, GitHub Actions automatically:

### 1. Build and Push Images (`release.yaml`)

| Image | Tags |
|-------|------|
| Operator | `v1.3.0`, `v1.3`, `sha-xxxxx` |
| Bundle | `v1.3.0` |
| Console Plugin | `v1.3.0`, `latest` |
| Catalog (per OCP version) | `v4.12`, `v4.13`, ..., `v4.20` |

### 2. Create GitHub Release

- Generates release notes from commits
- Attaches `dist/install.yaml` for direct deployment
- Marks pre-releases appropriately (for `-rc`, `-beta`, `-alpha` tags)

### 3. Build Catalog Images

Matrix build for all supported OCP versions (v4.12 - v4.20).

---

## Release Types

### Stable Release (e.g., `v1.3.0`)

- Published to `stable-v1` and `candidate-v1` channels
- Catalog images tagged as `v4.XX` (e.g., `v4.20`)
- GitHub Release marked as "Latest"

### Pre-Release (e.g., `v1.3.0-rc.1`, `v1.3.0-beta.1`)

- Published to `candidate-v1` channel only
- Catalog images tagged as `v4.XX-candidate`
- GitHub Release marked as "Pre-release"

---

## Manual Image Building

If you need to build images manually:

```bash
# Build operator image (multi-arch)
make podman-buildx

# Build bundle image (multi-arch)
make bundle-buildx

# Build catalog images (all OCP versions)
make catalog-build
make catalog-push

# Build single catalog
make catalog-build-single OCP_VERSION=v4.20
```

---

## Hotfix Process

For urgent fixes:

1. Create a hotfix branch from the release tag:
   ```bash
   git checkout -b hotfix/v1.3.1 v1.3.0
   ```

2. Apply fixes and commit

3. Update VERSION to `1.3.1`

4. Run `make release-prep`

5. Tag and push:
   ```bash
   git tag v1.3.1
   git push origin hotfix/v1.3.1 v1.3.1
   ```

6. Merge back to main

---

## Troubleshooting

### Catalog Validation Fails

```bash
# Check specific catalog
opm validate catalogs/v4.20

# Regenerate from templates
make catalogs
```

### Image Not Found

Ensure the bundle image exists in the registry before building catalogs:

```bash
# Check if bundle exists
podman pull ghcr.io/diegobskt/cluster-assessment-operator-bundle:v1.3.0
```

### Version Mismatch

Verify all files are in sync:

```bash
# Check VERSION file
cat VERSION

# Check Makefile derived version
make version

# Check manifest images
grep -r "image:.*cluster-assessment-operator" config/

# CRITICAL: Check bundle CSV version (must match release version for OLM)
grep "name: cluster-assessment-operator.v" bundle/manifests/cluster-assessment-operator.clusterserviceversion.yaml | head -1
grep "^  version:" bundle/manifests/cluster-assessment-operator.clusterserviceversion.yaml
```

### OLM Catalog Error: "bundle not found in any channel entries"

This error occurs when the bundle CSV version doesn't match the channel entries:

```bash
# Error example:
# package "cluster-assessment-operator", bundle "cluster-assessment-operator.v1.2.0" 
# not found in any channel entries

# Solution: Ensure bundle CSV version matches the release
echo "1.3.0" > VERSION
make update-manifests  # This now updates CSV name and version
make update-catalogs

# Verify:
grep "name: cluster-assessment-operator.v" bundle/manifests/cluster-assessment-operator.clusterserviceversion.yaml
# Should show: name: cluster-assessment-operator.v1.3.0
```

---

## See Also

- [CHANGELOG.md](../CHANGELOG.md) - Version history
- [CONTRIBUTING.md](../CONTRIBUTING.md) - Contribution guidelines
- [README.md](../README.md) - Quick start guide

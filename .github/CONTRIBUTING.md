# Contributing to Cluster Assessment Operator

Thank you for your interest in contributing! This document describes the development workflow and release process.

## Development Workflow

### 1. Fork and Clone

```bash
git clone https://github.com/<your-username>/cluster-assessment-operator
cd cluster-assessment-operator
```

### 2. Create a Feature Branch

```bash
git checkout -b feature/my-feature
```

### 3. Make Changes

- Write code following Go best practices
- Add tests for new functionality
- Update documentation as needed

### 4. Local Validation

```bash
# Run linting
go vet ./...
gofmt -l .

# Run tests
go test -v ./...

# Build
go build -o bin/manager ./main.go

# Validate bundle
operator-sdk bundle validate ./bundle

# Validate FBC catalogs
for v in v4.12 v4.13 v4.14 v4.15 v4.16 v4.17 v4.18 v4.19 v4.20; do
  opm validate catalogs/$v
done
```

### 5. Create Pull Request

Push your branch and create a PR against `main`. CI will automatically run:
- Lint checks
- Unit tests
- Build verification
- Bundle validation
- FBC catalog validation

## Release Process

### Release Channels

| Channel | Purpose | Tag Pattern | Example |
|---------|---------|-------------|---------|
| `candidate-v1` | Pre-release testing | `v*-rc.*`, `v*-beta.*` | v1.2.0-rc.1 |
| `stable-v1` | Production | `v*.*.*` | v1.2.0 |

### Creating a Pre-release (for testing)

1. Update version files:
   - `bundle/manifests/*.clusterserviceversion.yaml`
   - `CHANGELOG.md` (add "Unreleased" section)
   - `catalogs/*/cluster-assessment-operator/catalog.yaml`

2. Commit and tag:
   ```bash
   git add -A
   git commit -m "chore: prepare v1.2.0-rc.1"
   git tag v1.2.0-rc.1
   git push origin main --tags
   ```

3. Result: Published to **candidate-v1** channel only

### Creating a Stable Release

1. Update version files (same as above, but with stable version)

2. Update upgrade path in CSV:
   ```yaml
   olm.skipRange: ">=1.0.0 <1.2.0"
   replaces: cluster-assessment-operator.v1.1.1
   ```

3. Commit and tag:
   ```bash
   git add -A
   git commit -m "chore: release v1.2.0"
   git tag v1.2.0
   git push origin main --tags
   ```

4. Result: Published to **both** stable-v1 and candidate-v1 channels

### Testing in Cluster

```bash
# Subscribe to candidate channel for testing
cat <<EOF | oc apply -f -
apiVersion: operators.coreos.com/v1alpha1
kind: CatalogSource
metadata:
  name: cluster-assessment-candidate
  namespace: openshift-marketplace
spec:
  sourceType: grpc
  image: ghcr.io/diegobskt/cluster-assessment-operator-catalog:v4.20-candidate
  displayName: Cluster Assessment (Candidate)
EOF

# Subscribe to stable channel for production
cat <<EOF | oc apply -f -
apiVersion: operators.coreos.com/v1alpha1
kind: CatalogSource
metadata:
  name: cluster-assessment-stable
  namespace: openshift-marketplace
spec:
  sourceType: grpc
  image: ghcr.io/diegobskt/cluster-assessment-operator-catalog:v4.20
  displayName: Cluster Assessment (Stable)
EOF
```

## Code Style

- Follow [Effective Go](https://go.dev/doc/effective_go)
- Use `gofmt` for formatting
- Write meaningful commit messages following [Conventional Commits](https://www.conventionalcommits.org/)

# Contributing to Cluster Assessment Operator

First off, thank you for considering contributing to the Cluster Assessment Operator! 🎉

## Code of Conduct

This project adheres to a code of conduct. By participating, you are expected to uphold this code.

## How Can I Contribute?

### Reporting Bugs

- **Ensure the bug was not already reported** by searching [Issues](https://github.com/diegobskt/cluster-assessment-operator/issues)
- If you're unable to find an open issue, [open a new one](https://github.com/diegobskt/cluster-assessment-operator/issues/new)
- Include a **clear title and description**, as much relevant information as possible

### Suggesting Enhancements

- Open an issue with the `enhancement` label
- Describe the current behavior and explain the behavior you expected
- Explain why this enhancement would be useful

### Pull Requests

1. Fork the repo and create your branch from `main`
2. Make your changes
3. Ensure tests pass: `make test`
4. Ensure linting passes: `make lint`
5. Update documentation if needed
6. Submit a pull request

## Development Setup

### Prerequisites

- Go 1.25+
- Podman or Docker
- Access to an OpenShift 4.12+ cluster (for testing)
- operator-sdk v1.42.0+ (for OLM bundle validation)
- opm v1.36.0+ (for FBC catalog management)

### Getting Started

```bash
# Clone your fork
git clone https://github.com/YOUR_USERNAME/cluster-assessment-operator.git
cd cluster-assessment-operator

# Install dependencies
make deps

# Run tests
make test

# Run linter
make lint

# Build locally
make build

# Run locally (requires KUBECONFIG)
make run
```

### Making Changes

1. **Create a branch**
   ```bash
   git checkout -b feature/my-feature
   ```

2. **Make your changes** and add tests

3. **Run the test suite**
   ```bash
   make test
   make lint
   ```

4. **Commit your changes**
   ```bash
   git commit -m "feat: add new validator for XYZ"
   ```

### Commit Message Format

We follow [Conventional Commits](https://www.conventionalcommits.org/):

- `feat:` - New feature
- `fix:` - Bug fix
- `docs:` - Documentation only
- `test:` - Adding or updating tests
- `refactor:` - Code change that neither fixes a bug nor adds a feature
- `chore:` - Maintenance tasks

### Adding a New Validator

1. Create a new package under `pkg/validators/yourvalidator/`
2. Implement the `validator.Validator` interface
3. Register in `init()` function
4. Import in `main.go`
5. Add tests under `pkg/validators/yourvalidator/yourvalidator_test.go`
6. Update the README validators table

Example structure:
```go
package yourvalidator

import (
    "context"
    assessmentv1alpha1 "github.com/yourorg/cluster-assessment-operator/api/v1alpha1"
    "github.com/yourorg/cluster-assessment-operator/pkg/profiles"
    "github.com/yourorg/cluster-assessment-operator/pkg/validator"
    "sigs.k8s.io/controller-runtime/pkg/client"
)

func init() {
    _ = validator.Register(&YourValidator{})
}

type YourValidator struct{}

func (v *YourValidator) Name() string     { return "yourvalidator" }
func (v *YourValidator) Category() string { return "YourCategory" }

func (v *YourValidator) Validate(ctx context.Context, c client.Client, profile *profiles.Profile) ([]assessmentv1alpha1.Finding, error) {
    // Implementation
    return nil, nil
}
```

## Testing

### Running Tests

```bash
# All tests
make test

# With coverage
make test-coverage

# Specific package
go test -v ./pkg/validators/yourvalidator/...
```

### Integration Testing

```bash
# Deploy to cluster
make deploy

# Run an assessment
oc apply -f examples/quick-html-assessment.yaml
oc get clusterassessment -w
```

## Release Process

Releases are automated via GitHub Actions. To create a release:

1. Update `CHANGELOG.md`
2. Update version in CSV if needed
3. Create and push a version tag:
   ```bash
   git tag v1.x.x
   git push origin v1.x.x
   ```

This triggers:
- Multi-arch operator + bundle image builds
- FBC catalog images for OCP v4.12-v4.20
- GitHub Release with install.yaml
- Auto-generated PR to update FBC catalogs

### FBC Catalog Validation

Before release, validate catalogs locally:
```bash
make catalog-validate
operator-sdk bundle validate ./bundle --select-optional suite=operatorframework
```

## Questions?

Feel free to open an issue with the `question` label.

# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Added
- GitHub Actions CI/CD workflows
- Dependabot configuration for dependency updates
- CONTRIBUTING.md guidelines
- **File Based Catalog (FBC)** support for OCP v4.12-v4.20
- Three OLM channels: `stable-v1`, `candidate-v1`, `fast-v1`
- FBC validation in CI workflow
- Catalog image auto-build on release
- Auto-generated PR for FBC catalog updates

### Changed
- ConfigMap report names now include timestamp to prevent overwriting
- Updated channel naming per [OLM best practices](https://olm.operatorframework.io/docs/best-practices/channel-naming/)
- operator-sdk version updated to v1.42.0

### Fixed
- Race condition in status updates with RetryOnConflict
- Stuck assessment recovery with timeout mechanism

## [1.0.0] - 2026-01-14

### Added
- Initial release of Cluster Assessment Operator
- **12 Validators**:
  - `version` - OpenShift version, upgrade channel, update availability
  - `nodes` - Node count, conditions, roles, OS consistency
  - `machineconfig` - MachineConfigPool health, custom MachineConfigs
  - `apiserver` - API server status, etcd health, encryption, audit logging
  - `operators` - ClusterServiceVersion states, ClusterOperator health
  - `certificates` - TLS certificate expiration, custom certs
  - `etcdbackup` - OADP, Velero, backup CronJob configuration
  - `security` - Cluster-admin bindings, privileged pods, RBAC
  - `networking` - CNI type, NetworkPolicies, ingress configuration
  - `storage` - StorageClasses, default SC, CSI drivers
  - `monitoring` - Cluster monitoring, user workload monitoring
  - `deprecation` - Deprecated patterns, missing probes

- **Report Formats**: JSON, HTML, PDF
- **Report Storage**: ConfigMap with automatic timestamp naming
- **Baseline Profiles**: Production (strict) and Development (relaxed)
- **Scheduled Assessments**: Cron-based scheduling support
- **Severity Filtering**: Filter findings by minimum severity (INFO, PASS, WARN, FAIL)
- **Prometheus Metrics**: Assessment score, findings count, duration
- **OLM Bundle**: Full OLM support with scorecard tests passing
- **Multi-arch Support**: amd64 and arm64 container images
- **Red Hat Certification Ready**:
  - UBI9 base image
  - Required container labels
  - License directory
  - Non-root execution

### Security
- Read-only RBAC (only get, list, watch on cluster resources)
- Non-root container execution (USER 65532)
- Seccomp RuntimeDefault profile
- No privilege escalation

---

## Version History

| Version | Date | Description |
|---------|------|-------------|
| 1.0.0 | 2026-01-14 | Initial release |

[Unreleased]: https://github.com/diegobskt/cluster-assessment-operator/compare/v1.0.0...HEAD
[1.0.0]: https://github.com/diegobskt/cluster-assessment-operator/releases/tag/v1.0.0

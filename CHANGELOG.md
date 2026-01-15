# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

## [1.1.1] - 2026-01-15

### Fixed
- FBC catalog image references now use lowercase (fixes OLM visibility)
- ConfigMap names always include timestamp to prevent overwriting previous reports

## [1.1.0] - 2026-01-15

### Added
- **6 New Validators** (total now 18):
  - `imageregistry` - Registry configuration, storage backend, pruning, replicas
  - `compliance` - Pod Security Admission, OAuth providers, kubeadmin user
  - `resourcequotas` - ResourceQuota coverage, utilization, LimitRanges
  - `logging` - ClusterLogging operator, log forwarding, collector health
  - `costoptimization` - Orphan PVCs, idle deployments, resource specifications
  - `networkpolicyaudit` - Policy coverage, allow-all detection, default deny
- New **Governance** category for resource management validators

### Changed
- Validators are now organized alphabetically in main.go imports

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
| 1.1.1 | 2026-01-15 | FBC fix, ConfigMap timestamp enhancement |
| 1.1.0 | 2026-01-15 | 6 new validators (18 total) |
| 1.0.0 | 2026-01-14 | Initial release |

[Unreleased]: https://github.com/diegobskt/cluster-assessment-operator/compare/v1.1.1...HEAD
[1.1.1]: https://github.com/diegobskt/cluster-assessment-operator/compare/v1.1.0...v1.1.1
[1.1.0]: https://github.com/diegobskt/cluster-assessment-operator/compare/v1.0.0...v1.1.0
[1.0.0]: https://github.com/diegobskt/cluster-assessment-operator/releases/tag/v1.0.0


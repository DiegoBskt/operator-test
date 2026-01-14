# OpenShift Cluster Assessment Operator

[![Go Version](https://img.shields.io/badge/Go-1.25-blue.svg)](https://golang.org)
[![OpenShift](https://img.shields.io/badge/OpenShift-4.12+-red.svg)](https://www.redhat.com/en/technologies/cloud-computing/openshift)
[![License](https://img.shields.io/badge/License-Apache%202.0-blue.svg)](LICENSE)

A Kubernetes Operator for Red Hat OpenShift that performs **read-only** assessments of cluster configuration and generates human-readable reports highlighting configuration gaps, unsupported settings, and improvement opportunities.

## 🎯 Overview

The Cluster Assessment Operator is designed for consulting engagements where customers need visibility into their OpenShift cluster's configuration health. It provides:

- **Read-only assessments**: No automatic remediation or configuration changes
- **12 Validators**: Comprehensive checks across platform, security, networking, storage, and more  
- **Multiple report formats**: JSON, HTML, and PDF output
- **Baseline profiles**: Production (strict) and Development (relaxed) thresholds
- **Scheduled assessments**: On-demand or cron-based execution
- **Prometheus metrics**: Export assessment results as metrics for alerting
- **Severity filtering**: Focus on WARN/FAIL findings only

## ✨ Features

### Validators

| Validator | Category | Checks |
|-----------|----------|--------|
| `version` | Platform | OpenShift version, upgrade channel, update availability |
| `nodes` | Infrastructure | Node count, conditions, roles, OS consistency |
| `machineconfig` | Platform | MachineConfigPool health, custom MachineConfigs |
| `apiserver` | Platform | API server status, etcd health, encryption, audit logging |
| `operators` | Platform | ClusterServiceVersion states, ClusterOperator health |
| `certificates` | Security | TLS certificate expiration, custom certs |
| `etcdbackup` | Platform | OADP, Velero, backup CronJob configuration |
| `security` | Security | Cluster-admin bindings, privileged pods, RBAC |
| `networking` | Networking | CNI type, NetworkPolicies, ingress configuration |
| `storage` | Storage | StorageClasses, default SC, CSI drivers |
| `monitoring` | Observability | Cluster monitoring, user workload monitoring |
| `deprecation` | Compatibility | Deprecated patterns, missing probes |

### Prometheus Metrics

```
cluster_assessment_score{assessment_name, profile}
cluster_assessment_findings_total{assessment_name, status}
cluster_assessment_findings_by_category{assessment_name, category, status}
cluster_assessment_validator_findings{assessment_name, validator, status}
cluster_assessment_last_run_timestamp{assessment_name}
cluster_assessment_duration_seconds{assessment_name}
cluster_assessment_cluster_info{cluster_id, cluster_version, platform, channel}
```

## 📦 Installation

### Prerequisites

- OpenShift 4.12+
- `oc` CLI with cluster-admin access
- Podman (for building images)

### Quick Install

```bash
# Apply all manifests
oc apply -f config/crd/bases/
oc apply -f config/rbac/
oc apply -f config/manager/
```

### Build from Source

```bash
# Build single-arch image
make podman-build IMG=your-registry/cluster-assessment-operator:v1.0.0

# Build multi-arch image (amd64 + arm64)
make podman-buildx IMG=your-registry/cluster-assessment-operator:v1.0.0

# Push and deploy
make podman-push IMG=your-registry/cluster-assessment-operator:v1.0.0
make deploy IMG=your-registry/cluster-assessment-operator:v1.0.0
```

## 🚀 Usage

### Quick Assessment

```yaml
apiVersion: assessment.openshift.io/v1alpha1
kind: ClusterAssessment
metadata:
  name: production-assessment
spec:
  profile: production
  reportStorage:
    configMap:
      enabled: true
      format: "json,html,pdf"
```

```bash
oc apply -f examples/full-production-assessment.yaml
oc get clusterassessment -w
```

### Severity Filtering

Only include WARN and FAIL findings:

```yaml
spec:
  profile: production
  minSeverity: WARN
  reportStorage:
    configMap:
      enabled: true
      format: "html"
```

### Scheduled Assessment

```yaml
spec:
  schedule: "0 2 * * 0"  # Every Sunday at 2 AM
  profile: production
```

### Accessing Reports

```bash
# List report files
oc get configmap <name>-report -n openshift-cluster-assessment -o jsonpath='{.data}' | jq 'keys'

# Extract HTML report
oc get configmap <name>-report -n openshift-cluster-assessment \
  -o jsonpath='{.data.report\.html}' > report.html

# Extract PDF report
oc get configmap <name>-report -n openshift-cluster-assessment \
  -o jsonpath='{.binaryData.report\.pdf}' | base64 -d > report.pdf
```

## 📊 Baseline Profiles

### Production Profile
Strict enterprise requirements:
- Minimum 3 control plane nodes
- Minimum 3 worker nodes  
- Network policies required
- No privileged containers
- Updates within 90 days

### Development Profile
Relaxed for dev/test:
- Single node supported
- Privileged containers allowed
- Updates within 180 days

## 🔧 Development

```bash
# Run tests
make test

# Run tests with coverage
make test-coverage

# Lint code
make lint

# Build binary
make build

# Run locally
make run
```

## 📋 OLM / OperatorHub

The operator includes full OLM support:

```bash
# Generate bundle
make bundle

# Build bundle image
make bundle-build BUNDLE_IMG=your-registry/cluster-assessment-operator-bundle:v1.0.0

# Run scorecard tests
make scorecard

# Run Red Hat Preflight certification
make preflight
```

### Bundle Contents
- ClusterServiceVersion with spec/status descriptors
- Scorecard configuration for OLM validation
- Multi-architecture support (amd64, arm64)

## 📚 Documentation

| Document | Description |
|----------|-------------|
| [Troubleshooting](docs/troubleshooting.md) | Common issues and solutions |
| [Upgrade Guide](docs/upgrade.md) | Version upgrade procedures |
| [Examples](examples/) | Sample ClusterAssessment resources |

## 🏗️ Architecture

```
┌─────────────────────────────────────────────────────────────┐
│                    ClusterAssessment CR                      │
│  (spec: profile, schedule, validators, minSeverity)         │
└─────────────────────────────────────────────────────────────┘
                              │
                              ▼
┌─────────────────────────────────────────────────────────────┐
│                   Assessment Controller                      │
│  - Triggers assessments (on-demand or scheduled)            │
│  - Coordinates validators, filters by severity              │
│  - Records Prometheus metrics                               │
└─────────────────────────────────────────────────────────────┘
                              │
                              ▼
┌─────────────────────────────────────────────────────────────┐
│                 Validator Registry (12 validators)           │
│  version, nodes, operators, security, networking, ...       │
└─────────────────────────────────────────────────────────────┘
                              │
                              ▼
┌─────────────────────────────────────────────────────────────┐
│                    Report Generator                          │
│  - JSON / HTML / PDF output                                 │
│  - ConfigMap storage                                        │
└─────────────────────────────────────────────────────────────┘
```

## 🔒 Security

- **Read-only RBAC**: Only `get`, `list`, `watch` on cluster resources
- **Minimal container**: Distroless base image, non-root user
- **No privilege escalation**: `allowPrivilegeEscalation: false`
- **Seccomp**: `RuntimeDefault` profile enabled

## 🤝 Contributing

1. Fork the repository
2. Create a feature branch
3. Run tests: `make test`
4. Submit a pull request

## 📄 License

Apache License 2.0
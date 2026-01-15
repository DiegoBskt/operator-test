# Upgrade Guide

This document describes how to upgrade the Cluster Assessment Operator to newer versions.

## Version Compatibility

| Operator Version | OpenShift Versions | Channels | Breaking Changes |
|-----------------|-------------------|----------|------------------|
| 1.0.0 | 4.12 - 4.20 | stable-v1, candidate-v1, fast-v1 | Initial release |

---

## Upgrade Methods

### Method 1: OLM Automatic Upgrade (Recommended)

If installed via OLM, upgrades are automatic based on your subscription settings:

```bash
# Check current version
oc get csv -n openshift-operators | grep cluster-assessment

# Check subscription channel and approval
oc get subscription cluster-assessment-operator -n openshift-operators \
  -o jsonpath='channel: {.spec.channel}, approval: {.spec.installPlanApproval}'
```

**Channels:**
- `stable-v1` - Production-ready (recommended)
- `candidate-v1` - Pre-release/testing
- `fast-v1` - Early adopter access

**Approval:** `Automatic` (auto-upgrade) or `Manual` (approve each)

### Method 2: Manual Upgrade

If installed via manifests:

```bash
# 1. Update CRDs first (always do this before operator)
oc apply -f config/crd/bases/

# 2. Update RBAC if changed
oc apply -f config/rbac/

# 3. Update operator deployment
oc set image deploy/cluster-assessment-operator \
  manager=ghcr.io/diegobskt/cluster-assessment-operator:v1.1.0 \
  -n openshift-cluster-assessment

# 4. Verify rollout
oc rollout status deploy/cluster-assessment-operator -n openshift-cluster-assessment
```

---

## Pre-Upgrade Checklist

1. **Backup existing assessments:**
   ```bash
   oc get clusterassessments -o yaml > assessments-backup.yaml
   ```

2. **Check for in-progress assessments:**
   ```bash
   oc get clusterassessments -o jsonpath='{range .items[*]}{.metadata.name} {.status.phase}{"\n"}{end}'
   ```

3. **Review release notes** for breaking changes

4. **Test in non-production first**

---

## Post-Upgrade Verification

1. **Check operator is running:**
   ```bash
   oc get pods -n openshift-cluster-assessment
   ```

2. **Check new version:**
   ```bash
   oc logs -n openshift-cluster-assessment deploy/cluster-assessment-operator | head -5
   ```

3. **Run a test assessment:**
   ```bash
   oc apply -f examples/quick-html-assessment.yaml
   oc get clusterassessment quick-html-assessment -w
   ```

---

## Rollback Procedure

If issues occur after upgrade:

```bash
# Rollback operator
oc rollout undo deploy/cluster-assessment-operator -n openshift-cluster-assessment

# Or specify previous image
oc set image deploy/cluster-assessment-operator \
  manager=ghcr.io/diegobskt/cluster-assessment-operator:v1.0.0 \
  -n openshift-cluster-assessment
```

---

## CRD Versioning

The operator uses API versioning:

- **v1alpha1:** Current stable API
- **v1beta1:** (Future) Feature-complete API
- **v1:** (Future) Stable GA API

When API versions change, the operator includes conversion webhooks to automatically migrate resources.

---

## Breaking Changes Log

### v1.0.0 (Initial Release)
- No breaking changes (initial release)

### Future Versions
- Breaking changes will be documented here
- Migration scripts will be provided when necessary

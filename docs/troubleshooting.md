# Troubleshooting Guide

This guide helps diagnose and resolve common issues with the Cluster Assessment Operator.

## Common Issues

### 1. Assessment Stuck in "Pending" Phase

**Symptoms:** The ClusterAssessment CR shows `phase: Pending` and never progresses.

**Possible Causes:**
- Operator pod is not running
- RBAC permissions are insufficient

**Resolution:**
```bash
# Check operator pod status
oc get pods -n openshift-cluster-assessment

# Check operator logs
oc logs -n openshift-cluster-assessment deploy/cluster-assessment-operator

# Verify RBAC is applied
oc get clusterrole cluster-assessment-operator
```

---

### 2. Assessment Stuck in "Running" Phase

**Symptoms:** The assessment starts but never completes.

**Possible Causes:**
- Validator timeout (large cluster)
- Network connectivity issues
- API server throttling

**Resolution:**
```bash
# Check operator logs for errors
oc logs -n openshift-cluster-assessment deploy/cluster-assessment-operator --tail=100

# Check for API server throttling
oc get --raw /metrics | grep apiserver_request_total
```

---

### 3. Assessment Shows "Failed" Phase

**Symptoms:** The ClusterAssessment CR shows `phase: Failed`.

**Possible Causes:**
- Missing API resources (CRDs not installed)
- Operator crash during assessment

**Resolution:**
```bash
# Check the status message
oc get clusterassessment <name> -o jsonpath='{.status.message}'

# Check operator logs for stack traces
oc logs -n openshift-cluster-assessment deploy/cluster-assessment-operator | grep -A 10 "error"
```

---

### 4. No Findings in Report

**Symptoms:** Assessment completes but has 0 findings.

**Possible Causes:**
- All validators filtered by `validators` spec
- Severity filtering (`minSeverity`) too restrictive

**Resolution:**
```bash
# Check the spec
oc get clusterassessment <name> -o yaml | grep -A 20 spec

# Try without filtering
oc patch clusterassessment <name> --type=merge -p '{"spec":{"minSeverity":""}}'
```

---

### 5. ConfigMap Report Not Created

**Symptoms:** Assessment completes but no ConfigMap is created.

**Possible Causes:**
- `reportStorage.configMap.enabled` is false
- Namespace permissions issue

**Resolution:**
```bash
# Check if enabled
oc get clusterassessment <name> -o jsonpath='{.spec.reportStorage.configMap.enabled}'

# List ConfigMaps
oc get configmaps -n openshift-cluster-assessment | grep report
```

---

### 6. PDF Report Empty or Corrupted

**Symptoms:** The PDF file cannot be opened or is 0 bytes.

**Possible Causes:**
- Memory constraints in operator pod
- Base64 decoding issue

**Resolution:**
```bash
# Check if PDF data exists
oc get configmap <name>-report -n openshift-cluster-assessment -o jsonpath='{.binaryData}' | jq -r 'keys'

# Properly decode
oc get configmap <name>-report -n openshift-cluster-assessment -o jsonpath='{.binaryData.report\.pdf}' | base64 -d > report.pdf
```

---

### 7. Scheduled Assessment Not Running

**Symptoms:** Assessment has a schedule but never runs.

**Possible Causes:**
- Invalid cron expression
- `suspend: true` is set
- Operator restarted and lost schedule state

**Resolution:**
```bash
# Check schedule and suspend status
oc get clusterassessment <name> -o jsonpath='{.spec.schedule} {.spec.suspend}'

# Check nextRunTime
oc get clusterassessment <name> -o jsonpath='{.status.nextRunTime}'

# Delete and recreate to reset schedule
oc delete clusterassessment <name>
oc apply -f <name>.yaml
```

---

### 8. Prometheus Metrics Not Available

**Symptoms:** Metrics endpoint returns empty or 404.

**Possible Causes:**
- Metrics server not started
- ServiceMonitor not configured

**Resolution:**
```bash
# Check metrics endpoint
oc port-forward -n openshift-cluster-assessment deploy/cluster-assessment-operator 8080:8080
curl localhost:8080/metrics | grep cluster_assessment

# Verify operator is exposing metrics port
oc get deploy cluster-assessment-operator -n openshift-cluster-assessment -o yaml | grep -A 5 ports
```

---

## Collecting Debug Information

If you need to open a support ticket, collect the following:

```bash
# Operator version and status
oc get deploy cluster-assessment-operator -n openshift-cluster-assessment -o yaml > operator-deploy.yaml

# Operator logs
oc logs -n openshift-cluster-assessment deploy/cluster-assessment-operator --all-containers > operator-logs.txt

# All ClusterAssessments
oc get clusterassessments -o yaml > assessments.yaml

# Cluster version info
oc get clusterversion version -o yaml > cluster-version.yaml
```

---

## Getting Help

- **GitHub Issues:** Report bugs and feature requests
- **Documentation:** Check the README for latest information
- **Logs:** Always check operator logs first

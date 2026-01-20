---
description: Clean up all Cluster Assessment Operator resources from the cluster
---

# Cleanup Operator Resources

This workflow removes all cluster-assessment-operator resources from an OpenShift cluster.

## Prerequisites

- Logged into an OpenShift cluster with cluster-admin privileges
- `oc` CLI installed and configured

## Full Cleanup Procedure

// turbo-all

### 1. Delete any existing ClusterAssessments (custom resources)
```bash
echo "Deleting ClusterAssessment resources..."
oc delete clusterassessments --all -A 2>/dev/null || echo "No ClusterAssessments found"
```

### 2. Delete generated report ConfigMaps
```bash
echo "Deleting assessment report ConfigMaps..."
oc delete configmap -l app.kubernetes.io/managed-by=cluster-assessment-operator -n cluster-assessment-operator 2>/dev/null || true
```

### 3. Check how the operator was installed

#### Option A: If installed via OLM (Subscription/CSV)
```bash
echo "Checking for OLM installation..."
if oc get subscription cluster-assessment-operator -n cluster-assessment-operator &>/dev/null; then
    echo "Found OLM installation, cleaning up..."
    
    # Delete Subscription
    oc delete subscription cluster-assessment-operator -n cluster-assessment-operator 2>/dev/null || true
    
    # Delete CSV
    oc delete csv -l operators.coreos.com/cluster-assessment-operator.cluster-assessment-operator -n cluster-assessment-operator 2>/dev/null || true
    
    # Delete OperatorGroup
    oc delete operatorgroup cluster-assessment-operator -n cluster-assessment-operator 2>/dev/null || true
    
    # Delete CatalogSource
    oc delete catalogsource cluster-assessment-catalog -n openshift-marketplace 2>/dev/null || true
    
    echo "OLM resources cleaned up"
else
    echo "No OLM installation found"
fi
```

#### Option B: If installed via direct deployment (Makefile)
```bash
cd /Users/diego/Documents/cluster-assessment-operator
make undeploy-all
```

#### Option C: If installed via operator-sdk run bundle
```bash
cd /Users/diego/Documents/cluster-assessment-operator
make cleanup-olm
```

### 4. Disable the Console Plugin
```bash
echo "Disabling console plugin..."
# Get current plugins and remove cluster-assessment-plugin
CURRENT_PLUGINS=$(oc get consoles.operator.openshift.io cluster -o jsonpath='{.spec.plugins}' 2>/dev/null || echo "[]")
if echo "$CURRENT_PLUGINS" | grep -q "cluster-assessment-plugin"; then
    NEW_PLUGINS=$(echo "$CURRENT_PLUGINS" | jq 'map(select(. != "cluster-assessment-plugin"))')
    oc patch consoles.operator.openshift.io cluster \
      --type=merge \
      --patch="{\"spec\":{\"plugins\":$NEW_PLUGINS}}"
    echo "Console plugin disabled"
else
    echo "Console plugin was not enabled"
fi
```

### 5. Delete the namespace (if desired)
```bash
echo "Deleting namespace..."
oc delete namespace cluster-assessment-operator --wait=false 2>/dev/null || echo "Namespace already deleted"
```

### 6. Delete CRDs (optional - removes all data!)
```bash
echo "Deleting CRDs..."
oc delete crd clusterassessments.assessment.openshift.io 2>/dev/null || echo "CRD already deleted"
```

---

## Quick Cleanup Commands

### For OLM Installation (one-liner)
```bash
oc delete subscription,csv,operatorgroup -l operators.coreos.com/cluster-assessment-operator -n cluster-assessment-operator; \
oc delete catalogsource cluster-assessment-catalog -n openshift-marketplace; \
oc delete namespace cluster-assessment-operator
```

### For Direct Deployment (Makefile)
```bash
cd /Users/diego/Documents/cluster-assessment-operator && make undeploy-all
```

### For operator-sdk Bundle
```bash
cd /Users/diego/Documents/cluster-assessment-operator && make cleanup-olm
```

---

## Verification

After cleanup, verify all resources are removed:

```bash
# Check namespace is gone
oc get namespace cluster-assessment-operator 2>/dev/null && echo "WARN: Namespace still exists" || echo "OK: Namespace deleted"

# Check CRD is gone
oc get crd clusterassessments.assessment.openshift.io 2>/dev/null && echo "WARN: CRD still exists" || echo "OK: CRD deleted"

# Check CatalogSource is gone
oc get catalogsource cluster-assessment-catalog -n openshift-marketplace 2>/dev/null && echo "WARN: CatalogSource still exists" || echo "OK: CatalogSource deleted"

# Check console plugin is disabled
oc get consoles.operator.openshift.io cluster -o jsonpath='{.spec.plugins}' | grep -q cluster-assessment-plugin && echo "WARN: Console plugin still enabled" || echo "OK: Console plugin disabled"
```

---
description: Deploy the Cluster Assessment Operator from scratch with latest version
---

# Deploy Operator from Scratch

This workflow deploys the cluster-assessment-operator and console plugin on an OpenShift cluster using the latest version.

## Prerequisites

- Logged into an OpenShift cluster with cluster-admin privileges
- `oc` CLI installed and configured
- For OLM deployment: operator images must be published to the registry

## Option A: OLM Deployment (Production - Recommended)

This method uses OLM (Operator Lifecycle Manager) with the FBC catalog.

// turbo-all

### 1. Get the OpenShift version to determine the correct catalog
```bash
OCP_VERSION=$(oc version -o json | jq -r '.openshiftVersion' | cut -d. -f1,2 | sed 's/^/v/')
echo "Detected OpenShift version: $OCP_VERSION"
```

### 2. Deploy the CatalogSource, Namespace, OperatorGroup, and Subscription
```bash
oc apply -f - <<EOF
---
apiVersion: operators.coreos.com/v1alpha1
kind: CatalogSource
metadata:
  name: cluster-assessment-catalog
  namespace: openshift-marketplace
spec:
  sourceType: grpc
  image: ghcr.io/diegobskt/cluster-assessment-operator-catalog:${OCP_VERSION:-v4.20}
  displayName: Cluster Assessment Operator
  publisher: Community
  updateStrategy:
    registryPoll:
      interval: 10m
---
apiVersion: v1
kind: Namespace
metadata:
  name: cluster-assessment-operator
  labels:
    openshift.io/cluster-monitoring: "true"
---
apiVersion: operators.coreos.com/v1
kind: OperatorGroup
metadata:
  name: cluster-assessment-operator
  namespace: cluster-assessment-operator
spec: {}
---
apiVersion: operators.coreos.com/v1alpha1
kind: Subscription
metadata:
  name: cluster-assessment-operator
  namespace: cluster-assessment-operator
spec:
  channel: stable-v1
  name: cluster-assessment-operator
  source: cluster-assessment-catalog
  sourceNamespace: openshift-marketplace
  installPlanApproval: Automatic
EOF
```

### 3. Wait for the operator to be installed
```bash
echo "Waiting for CSV to be created..."
sleep 10
oc get csv -n cluster-assessment-operator -w
```

### 4. Enable the Console Plugin
```bash
oc patch consoles.operator.openshift.io cluster \
  --type=merge \
  --patch='{"spec":{"plugins":["cluster-assessment-plugin"]}}'
```

### 5. Verify deployment
```bash
oc get pods -n cluster-assessment-operator
oc get csv -n cluster-assessment-operator
```

---

## Option B: Direct Deployment (Development/Testing)

This method deploys directly from manifests without OLM.

// turbo-all

### 1. Navigate to the operator directory
```bash
cd /Users/diego/Documents/cluster-assessment-operator
```

### 2. Deploy using Makefile
```bash
make deploy-all
```

### 3. Enable the Console Plugin
```bash
oc patch consoles.operator.openshift.io cluster \
  --type=merge \
  --patch='{"spec":{"plugins":["cluster-assessment-plugin"]}}'
```

### 4. Verify deployment
```bash
oc get pods -n cluster-assessment-operator
```

---

## Option C: OLM Bundle Deployment (Quick Testing)

This method uses operator-sdk for quick testing with a bundle image.

// turbo-all

### 1. Navigate to the operator directory
```bash
cd /Users/diego/Documents/cluster-assessment-operator
```

### 2. Deploy via operator-sdk run bundle
```bash
make deploy-olm
```

### 3. Enable the Console Plugin
```bash
oc patch consoles.operator.openshift.io cluster \
  --type=merge \
  --patch='{"spec":{"plugins":["cluster-assessment-plugin"]}}'
```

---

## Verification Steps

After deployment, verify the operator is working:

### Check operator pod
```bash
oc get pods -n cluster-assessment-operator -l app.kubernetes.io/name=cluster-assessment-operator
```

### Check console plugin pod
```bash
oc get pods -n cluster-assessment-operator -l app=cluster-assessment-plugin
```

### Create a test assessment
```bash
oc apply -f examples/clusterassessment-basic.yaml
```

### Watch assessment progress
```bash
oc get clusterassessment -w
```

### Check Console UI
Navigate to **OpenShift Console** → **Observe** → **Cluster Assessment**

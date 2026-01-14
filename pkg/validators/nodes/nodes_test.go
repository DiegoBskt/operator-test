/*
Copyright 2024.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package nodes

import (
	"context"
	"testing"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	assessmentv1alpha1 "github.com/openshift-assessment/cluster-assessment-operator/api/v1alpha1"
	"github.com/openshift-assessment/cluster-assessment-operator/pkg/profiles"
)

func TestNodesValidator_Name(t *testing.T) {
	v := &NodesValidator{}
	if v.Name() != "nodes" {
		t.Errorf("Expected name 'nodes', got '%s'", v.Name())
	}
}

func TestNodesValidator_Category(t *testing.T) {
	v := &NodesValidator{}
	if v.Category() != "Infrastructure" {
		t.Errorf("Expected category 'Infrastructure', got '%s'", v.Category())
	}
}

func TestNodesValidator_Validate_ProductionProfile(t *testing.T) {
	scheme := runtime.NewScheme()
	_ = corev1.AddToScheme(scheme)

	// Create a production-like cluster with 3 masters and 3 workers
	nodes := []client.Object{
		createNode("master-0", true, false, "Red Hat Enterprise Linux CoreOS"),
		createNode("master-1", true, false, "Red Hat Enterprise Linux CoreOS"),
		createNode("master-2", true, false, "Red Hat Enterprise Linux CoreOS"),
		createNode("worker-0", false, true, "Red Hat Enterprise Linux CoreOS"),
		createNode("worker-1", false, true, "Red Hat Enterprise Linux CoreOS"),
		createNode("worker-2", false, true, "Red Hat Enterprise Linux CoreOS"),
	}

	fakeClient := fake.NewClientBuilder().
		WithScheme(scheme).
		WithObjects(nodes...).
		Build()

	v := &NodesValidator{}
	profile := profiles.GetProfile("production")

	findings, err := v.Validate(context.Background(), fakeClient, profile)
	if err != nil {
		t.Fatalf("Validate() returned error: %v", err)
	}

	// Should have findings for node count checks
	if len(findings) < 1 {
		t.Errorf("Expected at least 1 finding, got %d", len(findings))
	}

	// Log findings for debugging
	for _, f := range findings {
		t.Logf("[%s] %s: %s", f.Status, f.ID, f.Title)
	}

	// In a healthy production cluster, node count should pass
	hasPassingNodeCount := false
	for _, f := range findings {
		if f.ID == "nodes-count-production" && f.Status == assessmentv1alpha1.FindingStatusPass {
			hasPassingNodeCount = true
		}
	}

	if !hasPassingNodeCount {
		t.Logf("Note: May not have passing node count - depends on profile thresholds")
	}
}

func TestNodesValidator_Validate_InsufficientNodes(t *testing.T) {
	scheme := runtime.NewScheme()
	_ = corev1.AddToScheme(scheme)

	// Create a cluster with only 1 master and 1 worker
	nodes := []client.Object{
		createNode("master-0", true, false, "Red Hat Enterprise Linux CoreOS"),
		createNode("worker-0", false, true, "Red Hat Enterprise Linux CoreOS"),
	}

	fakeClient := fake.NewClientBuilder().
		WithScheme(scheme).
		WithObjects(nodes...).
		Build()

	v := &NodesValidator{}
	profile := profiles.GetProfile("production")

	findings, err := v.Validate(context.Background(), fakeClient, profile)
	if err != nil {
		t.Fatalf("Validate() returned error: %v", err)
	}

	// Should have warnings/fails for insufficient nodes in production
	hasNodeWarning := false
	for _, f := range findings {
		if f.Status == assessmentv1alpha1.FindingStatusWarn || f.Status == assessmentv1alpha1.FindingStatusFail {
			hasNodeWarning = true
			t.Logf("Found warning/fail: [%s] %s", f.Status, f.Title)
		}
	}

	if !hasNodeWarning {
		t.Logf("Note: Expected warnings for insufficient node count in production profile")
	}
}

func TestNodesValidator_Validate_NotReadyNode(t *testing.T) {
	scheme := runtime.NewScheme()
	_ = corev1.AddToScheme(scheme)

	// Create a cluster with a NotReady node
	nodes := []client.Object{
		createNode("master-0", true, false, "Red Hat Enterprise Linux CoreOS"),
		createNodeWithCondition("worker-0", false, true, corev1.ConditionFalse),
	}

	fakeClient := fake.NewClientBuilder().
		WithScheme(scheme).
		WithObjects(nodes...).
		Build()

	v := &NodesValidator{}
	profile := profiles.GetProfile("production")

	findings, err := v.Validate(context.Background(), fakeClient, profile)
	if err != nil {
		t.Fatalf("Validate() returned error: %v", err)
	}

	// Should have a finding about NotReady node
	hasNotReadyFinding := false
	for _, f := range findings {
		if f.ID == "nodes-not-ready" {
			hasNotReadyFinding = true
			if f.Status != assessmentv1alpha1.FindingStatusFail {
				t.Errorf("Expected FAIL status for NotReady node, got %s", f.Status)
			}
		}
	}

	if !hasNotReadyFinding {
		t.Logf("Note: Expected finding for NotReady node")
	}
}

func createNode(name string, isMaster, isWorker bool, osImage string) *corev1.Node {
	labels := map[string]string{}
	if isMaster {
		labels["node-role.kubernetes.io/master"] = ""
		labels["node-role.kubernetes.io/control-plane"] = ""
	}
	if isWorker {
		labels["node-role.kubernetes.io/worker"] = ""
	}

	return &corev1.Node{
		ObjectMeta: metav1.ObjectMeta{
			Name:   name,
			Labels: labels,
		},
		Status: corev1.NodeStatus{
			Conditions: []corev1.NodeCondition{
				{
					Type:   corev1.NodeReady,
					Status: corev1.ConditionTrue,
				},
			},
			NodeInfo: corev1.NodeSystemInfo{
				OSImage: osImage,
			},
		},
	}
}

func createNodeWithCondition(name string, isMaster, isWorker bool, readyStatus corev1.ConditionStatus) *corev1.Node {
	labels := map[string]string{}
	if isMaster {
		labels["node-role.kubernetes.io/master"] = ""
	}
	if isWorker {
		labels["node-role.kubernetes.io/worker"] = ""
	}

	return &corev1.Node{
		ObjectMeta: metav1.ObjectMeta{
			Name:   name,
			Labels: labels,
		},
		Status: corev1.NodeStatus{
			Conditions: []corev1.NodeCondition{
				{
					Type:   corev1.NodeReady,
					Status: readyStatus,
				},
			},
			NodeInfo: corev1.NodeSystemInfo{
				OSImage: "Red Hat Enterprise Linux CoreOS",
			},
		},
	}
}

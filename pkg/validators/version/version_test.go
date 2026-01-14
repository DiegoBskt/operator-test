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

package version

import (
	"context"
	"testing"

	configv1 "github.com/openshift/api/config/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	assessmentv1alpha1 "github.com/openshift-assessment/cluster-assessment-operator/api/v1alpha1"
	"github.com/openshift-assessment/cluster-assessment-operator/pkg/profiles"
)

func TestVersionValidator_Name(t *testing.T) {
	v := &VersionValidator{}
	if v.Name() != "version" {
		t.Errorf("Expected name 'version', got '%s'", v.Name())
	}
}

func TestVersionValidator_Category(t *testing.T) {
	v := &VersionValidator{}
	if v.Category() != "Platform" {
		t.Errorf("Expected category 'Platform', got '%s'", v.Category())
	}
}

func TestVersionValidator_Validate(t *testing.T) {
	scheme := runtime.NewScheme()
	_ = configv1.AddToScheme(scheme)

	tests := []struct {
		name           string
		clusterVersion *configv1.ClusterVersion
		wantFindings   int
		wantStatus     assessmentv1alpha1.FindingStatus
	}{
		{
			name: "healthy cluster version",
			clusterVersion: &configv1.ClusterVersion{
				ObjectMeta: metav1.ObjectMeta{Name: "version"},
				Spec: configv1.ClusterVersionSpec{
					ClusterID: "test-cluster-id",
					Channel:   "stable-4.14",
				},
				Status: configv1.ClusterVersionStatus{
					History: []configv1.UpdateHistory{
						{
							State:   configv1.CompletedUpdate,
							Version: "4.14.8",
						},
					},
					Conditions: []configv1.ClusterOperatorStatusCondition{
						{
							Type:   configv1.OperatorAvailable,
							Status: configv1.ConditionTrue,
						},
						{
							Type:   configv1.OperatorProgressing,
							Status: configv1.ConditionFalse,
						},
						{
							Type:   configv1.OperatorDegraded,
							Status: configv1.ConditionFalse,
						},
					},
				},
			},
			wantFindings: 5, // version check, channel check, conditions, updates, lifecycle
			wantStatus:   assessmentv1alpha1.FindingStatusPass,
		},
		{
			name: "degraded cluster",
			clusterVersion: &configv1.ClusterVersion{
				ObjectMeta: metav1.ObjectMeta{Name: "version"},
				Spec: configv1.ClusterVersionSpec{
					ClusterID: "test-cluster-id",
					Channel:   "stable-4.14",
				},
				Status: configv1.ClusterVersionStatus{
					History: []configv1.UpdateHistory{
						{
							State:   configv1.CompletedUpdate,
							Version: "4.14.8",
						},
					},
					Conditions: []configv1.ClusterOperatorStatusCondition{
						{
							Type:   configv1.OperatorAvailable,
							Status: configv1.ConditionTrue,
						},
						{
							Type:   configv1.OperatorDegraded,
							Status: configv1.ConditionTrue,
						},
					},
				},
			},
			wantFindings: 5,
			wantStatus:   assessmentv1alpha1.FindingStatusFail,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			objects := []client.Object{}
			if tt.clusterVersion != nil {
				objects = append(objects, tt.clusterVersion)
			}

			fakeClient := fake.NewClientBuilder().
				WithScheme(scheme).
				WithObjects(objects...).
				Build()

			v := &VersionValidator{}
			profile := profiles.GetProfile("production")

			findings, err := v.Validate(context.Background(), fakeClient, profile)
			if err != nil {
				t.Fatalf("Validate() returned error: %v", err)
			}

			if len(findings) < 1 {
				t.Errorf("Expected at least 1 finding, got %d", len(findings))
			}

			// Check that we have findings
			t.Logf("Got %d findings", len(findings))
			for _, f := range findings {
				t.Logf("  - [%s] %s: %s", f.Status, f.ID, f.Title)
			}
		})
	}
}

func TestVersionValidator_NoClusterVersion(t *testing.T) {
	scheme := runtime.NewScheme()
	_ = configv1.AddToScheme(scheme)

	fakeClient := fake.NewClientBuilder().
		WithScheme(scheme).
		Build()

	v := &VersionValidator{}
	profile := profiles.GetProfile("production")

	_, err := v.Validate(context.Background(), fakeClient, profile)

	// Validator should return error when ClusterVersion is not found
	// This is correct behavior - it's a critical resource
	if err == nil {
		t.Error("Expected error when ClusterVersion is missing, got nil")
	}
}

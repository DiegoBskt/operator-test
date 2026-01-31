package controllers

import (
	"context"
	"os"
	"strings"
	"testing"

	assessmentv1alpha1 "github.com/openshift-assessment/cluster-assessment-operator/api/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

func TestExportToGit_SecretNamespace(t *testing.T) {
	// Register schemes
	scheme := runtime.NewScheme()
	_ = clientgoscheme.AddToScheme(scheme)
	_ = assessmentv1alpha1.AddToScheme(scheme)

	// Setup test data
	ns := "tenant-ns"
	secretName := "git-creds"

	// Secret is in the tenant namespace
	secret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      secretName,
			Namespace: ns,
		},
		Data: map[string][]byte{
			"username": []byte("user"),
			"password": []byte("pass"),
		},
	}

	assessment := &assessmentv1alpha1.ClusterAssessment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-assessment",
			Namespace: ns,
		},
		Spec: assessmentv1alpha1.ClusterAssessmentSpec{
			ReportStorage: assessmentv1alpha1.ReportStorageSpec{
				Git: &assessmentv1alpha1.GitStorageSpec{
					Enabled:   true,
					URL:       "https://example.com/repo.git", // Dummy URL
					Branch:    "main",
					SecretRef: secretName,
				},
			},
		},
	}

	// Create fake client with the secret in tenant-ns
	cl := fake.NewClientBuilder().
		WithScheme(scheme).
		WithObjects(secret, assessment).
		Build()

	r := &ClusterAssessmentReconciler{
		Client: cl,
		Scheme: scheme,
	}

	// Set POD_NAMESPACE to imitate running in operator namespace
	os.Setenv("POD_NAMESPACE", "operator-ns")
	defer os.Unsetenv("POD_NAMESPACE")

	// Execute exportToGit
	// We expect it to fail, but the error message tells us WHERE it failed (lookup vs clone)
	err := r.exportToGit(context.Background(), assessment)

	if err == nil {
		t.Fatal("Expected error, got nil")
	}

	// Analyze the error
	// If it contains "failed to get git secret" and "not found", it means it looked in the wrong namespace.
	// If it contains "failed to clone repository", it means it found the secret and proceeded to clone.

	errMsg := err.Error()
	t.Logf("Got error: %s", errMsg)

	if strings.Contains(errMsg, "failed to get git secret") && strings.Contains(errMsg, "not found") {
		t.Errorf("VULNERABILITY DETECTED: Secret lookup failed in operator namespace (should have looked in tenant namespace)")
	} else if strings.Contains(errMsg, "failed to clone repository") {
		t.Log("Success: Secret found in tenant namespace (proceeded to clone)")
	} else {
		t.Errorf("Unexpected error: %s", errMsg)
	}
}

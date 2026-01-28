package report

import (
	"testing"

	assessmentv1alpha1 "github.com/openshift-assessment/cluster-assessment-operator/api/v1alpha1"
	"github.com/openshift-assessment/cluster-assessment-operator/pkg/version"
)

func TestBuildReportPopulatesVersion(t *testing.T) {
	// Save original version and defer restoration
	originalVersion := version.Version
	defer func() { version.Version = originalVersion }()

	// Set a test version
	testVersion := "1.2.3-test"
	version.Version = testVersion

	// Create a dummy assessment
	assessment := &assessmentv1alpha1.ClusterAssessment{
		Spec: assessmentv1alpha1.ClusterAssessmentSpec{
			Profile: "default",
		},
		Status: assessmentv1alpha1.ClusterAssessmentStatus{},
	}

	// Build report
	report := buildReport(assessment)

	// Verify OperatorVersion
	if report.Metadata.OperatorVersion != testVersion {
		t.Errorf("Expected OperatorVersion to be %q, got %q", testVersion, report.Metadata.OperatorVersion)
	}
}

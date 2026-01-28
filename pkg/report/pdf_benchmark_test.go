package report

import (
	"fmt"
	"testing"

	assessmentv1alpha1 "github.com/openshift-assessment/cluster-assessment-operator/api/v1alpha1"
)

func BenchmarkGeneratePDF(b *testing.B) {
	// Create a large number of findings
	numFindings := 5000 // Increased to make the impact more visible
	findings := make([]assessmentv1alpha1.Finding, numFindings)
	statuses := []assessmentv1alpha1.FindingStatus{
		assessmentv1alpha1.FindingStatusPass,
		assessmentv1alpha1.FindingStatusWarn,
		assessmentv1alpha1.FindingStatusFail,
		assessmentv1alpha1.FindingStatusInfo,
	}

	for i := 0; i < numFindings; i++ {
		findings[i] = assessmentv1alpha1.Finding{
			Title:       fmt.Sprintf("Finding %d", i),
			Description: "This is a test finding description that is somewhat long to simulate real data.",
			Status:      statuses[i%4],
			Category:    "Test Category",
			Validator:   "TestValidator",
		}
	}

	assessment := &assessmentv1alpha1.ClusterAssessment{
		Status: assessmentv1alpha1.ClusterAssessmentStatus{
			Findings: findings,
			Summary: assessmentv1alpha1.AssessmentSummary{
				TotalChecks: numFindings,
			},
			ClusterInfo: assessmentv1alpha1.ClusterInfo{
				ClusterID:      "test-cluster",
				ClusterVersion: "4.14.0",
			},
		},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := GeneratePDF(assessment)
		if err != nil {
			b.Fatalf("GeneratePDF failed: %v", err)
		}
	}
}

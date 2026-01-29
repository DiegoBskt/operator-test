package report

import (
	"strings"
	"testing"

	assessmentv1alpha1 "github.com/openshift-assessment/cluster-assessment-operator/api/v1alpha1"
)

func TestGenerateHTML_XSS(t *testing.T) {
	// Create a dummy assessment with malicious content
	assessment := &assessmentv1alpha1.ClusterAssessment{
		Status: assessmentv1alpha1.ClusterAssessmentStatus{
			Findings: []assessmentv1alpha1.Finding{
				{
					Title:          "<script>alert('title')</script>",
					Description:    "Desc <img src=x onerror=alert(1)>",
					Category:       "Security",
					Validator:      "test-validator",
					Status:         assessmentv1alpha1.FindingStatusFail,
					Recommendation: "<b>Do not do this</b>",
					References:     []string{"javascript:alert(1)", "http://good.com"},
				},
			},
			Summary: assessmentv1alpha1.AssessmentSummary{
				FailCount: 1,
			},
		},
	}

	// Generate HTML
	htmlBytes, err := GenerateHTML(assessment)
	if err != nil {
		t.Fatalf("GenerateHTML failed: %v", err)
	}
	htmlStr := string(htmlBytes)

	// t.Logf("Generated HTML: %s", htmlStr)

	// Check for unescaped tags - SHOULD NOT EXIST
	if strings.Contains(htmlStr, "<script>") {
		t.Errorf("HTML contains unescaped script tag in title")
	}
	if strings.Contains(htmlStr, "<img src=x") {
		t.Errorf("HTML contains unescaped img tag in description")
	}
	if strings.Contains(htmlStr, "<b>") {
		t.Errorf("HTML contains unescaped b tag in recommendation")
	}

	// Check for escaped tags - SHOULD EXIST
	if !strings.Contains(htmlStr, "&lt;script&gt;") {
		t.Errorf("HTML should contain escaped script tag")
	}
	if !strings.Contains(htmlStr, "&lt;img src=x") {
		t.Errorf("HTML should contain escaped img tag")
	}
	if !strings.Contains(htmlStr, "&lt;b&gt;") {
		t.Errorf("HTML should contain escaped b tag")
	}

	// Check for javascript URL in href - SHOULD NOT EXIST
	if strings.Contains(htmlStr, "href=\"javascript:alert(1)\"") {
		t.Errorf("VULNERABILITY CONFIRMED: HTML contains javascript: URL in href")
	}
}

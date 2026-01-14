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

package controllers

import (
	"testing"

	assessmentv1alpha1 "github.com/openshift-assessment/cluster-assessment-operator/api/v1alpha1"
)

func TestFilterBySeverity(t *testing.T) {
	r := &ClusterAssessmentReconciler{}

	findings := []assessmentv1alpha1.Finding{
		{ID: "info-1", Status: assessmentv1alpha1.FindingStatusInfo, Title: "Info Finding 1"},
		{ID: "pass-1", Status: assessmentv1alpha1.FindingStatusPass, Title: "Pass Finding 1"},
		{ID: "warn-1", Status: assessmentv1alpha1.FindingStatusWarn, Title: "Warn Finding 1"},
		{ID: "fail-1", Status: assessmentv1alpha1.FindingStatusFail, Title: "Fail Finding 1"},
		{ID: "info-2", Status: assessmentv1alpha1.FindingStatusInfo, Title: "Info Finding 2"},
		{ID: "pass-2", Status: assessmentv1alpha1.FindingStatusPass, Title: "Pass Finding 2"},
		{ID: "warn-2", Status: assessmentv1alpha1.FindingStatusWarn, Title: "Warn Finding 2"},
		{ID: "fail-2", Status: assessmentv1alpha1.FindingStatusFail, Title: "Fail Finding 2"},
	}

	tests := []struct {
		name        string
		minSeverity string
		wantCount   int
	}{
		{
			name:        "filter INFO - include all",
			minSeverity: "INFO",
			wantCount:   8,
		},
		{
			name:        "filter PASS - exclude INFO",
			minSeverity: "PASS",
			wantCount:   6, // 2 PASS + 2 WARN + 2 FAIL
		},
		{
			name:        "filter WARN - include WARN and FAIL only",
			minSeverity: "WARN",
			wantCount:   4, // 2 WARN + 2 FAIL
		},
		{
			name:        "filter FAIL - include FAIL only",
			minSeverity: "FAIL",
			wantCount:   2, // 2 FAIL
		},
		{
			name:        "invalid severity - return all",
			minSeverity: "INVALID",
			wantCount:   8,
		},
		{
			name:        "empty severity - return all",
			minSeverity: "",
			wantCount:   8,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			filtered := r.filterBySeverity(findings, tt.minSeverity)
			if len(filtered) != tt.wantCount {
				t.Errorf("filterBySeverity(%s) returned %d findings, want %d",
					tt.minSeverity, len(filtered), tt.wantCount)
				for _, f := range filtered {
					t.Logf("  - %s: %s", f.Status, f.Title)
				}
			}
		})
	}
}

func TestCalculateSummary(t *testing.T) {
	r := &ClusterAssessmentReconciler{}

	findings := []assessmentv1alpha1.Finding{
		{ID: "info-1", Status: assessmentv1alpha1.FindingStatusInfo},
		{ID: "pass-1", Status: assessmentv1alpha1.FindingStatusPass},
		{ID: "pass-2", Status: assessmentv1alpha1.FindingStatusPass},
		{ID: "warn-1", Status: assessmentv1alpha1.FindingStatusWarn},
		{ID: "fail-1", Status: assessmentv1alpha1.FindingStatusFail},
	}

	summary := r.calculateSummary(findings, "production")

	if summary.TotalChecks != 5 {
		t.Errorf("Expected TotalChecks=5, got %d", summary.TotalChecks)
	}

	if summary.PassCount != 2 {
		t.Errorf("Expected PassCount=2, got %d", summary.PassCount)
	}

	if summary.WarnCount != 1 {
		t.Errorf("Expected WarnCount=1, got %d", summary.WarnCount)
	}

	if summary.FailCount != 1 {
		t.Errorf("Expected FailCount=1, got %d", summary.FailCount)
	}

	if summary.InfoCount != 1 {
		t.Errorf("Expected InfoCount=1, got %d", summary.InfoCount)
	}

	if summary.ProfileUsed != "production" {
		t.Errorf("Expected ProfileUsed='production', got '%s'", summary.ProfileUsed)
	}

	if summary.Score == nil {
		t.Error("Expected Score to be set")
	} else {
		// Score = (2*100 + 1*80 + 1*50) / 5 = 330/5 = 66
		expectedScore := 66
		if *summary.Score != expectedScore {
			t.Errorf("Expected Score=%d, got %d", expectedScore, *summary.Score)
		}
	}
}

func TestCalculateSummary_AllPass(t *testing.T) {
	r := &ClusterAssessmentReconciler{}

	findings := []assessmentv1alpha1.Finding{
		{ID: "pass-1", Status: assessmentv1alpha1.FindingStatusPass},
		{ID: "pass-2", Status: assessmentv1alpha1.FindingStatusPass},
		{ID: "pass-3", Status: assessmentv1alpha1.FindingStatusPass},
	}

	summary := r.calculateSummary(findings, "production")

	if summary.Score == nil {
		t.Error("Expected Score to be set")
	} else if *summary.Score != 100 {
		t.Errorf("Expected Score=100 for all PASS, got %d", *summary.Score)
	}
}

func TestCalculateSummary_AllFail(t *testing.T) {
	r := &ClusterAssessmentReconciler{}

	findings := []assessmentv1alpha1.Finding{
		{ID: "fail-1", Status: assessmentv1alpha1.FindingStatusFail},
		{ID: "fail-2", Status: assessmentv1alpha1.FindingStatusFail},
	}

	summary := r.calculateSummary(findings, "production")

	if summary.Score == nil {
		t.Error("Expected Score to be set")
	} else if *summary.Score != 0 {
		t.Errorf("Expected Score=0 for all FAIL, got %d", *summary.Score)
	}
}

func TestCalculateSummary_Empty(t *testing.T) {
	r := &ClusterAssessmentReconciler{}

	findings := []assessmentv1alpha1.Finding{}

	summary := r.calculateSummary(findings, "production")

	if summary.TotalChecks != 0 {
		t.Errorf("Expected TotalChecks=0, got %d", summary.TotalChecks)
	}

	if summary.Score != nil {
		t.Error("Expected Score to be nil for empty findings")
	}
}

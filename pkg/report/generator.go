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

package report

import (
	"encoding/json"
	"time"

	"gopkg.in/yaml.v3"

	assessmentv1alpha1 "github.com/openshift-assessment/cluster-assessment-operator/api/v1alpha1"
)

// Report represents the full assessment report structure.
type Report struct {
	// Metadata about the report
	Metadata ReportMetadata `json:"metadata" yaml:"metadata"`

	// ClusterInfo contains cluster metadata
	ClusterInfo assessmentv1alpha1.ClusterInfo `json:"clusterInfo" yaml:"clusterInfo"`

	// Summary provides an overview of results
	Summary assessmentv1alpha1.AssessmentSummary `json:"summary" yaml:"summary"`

	// Findings is the list of all findings
	Findings []assessmentv1alpha1.Finding `json:"findings" yaml:"findings"`

	// FindingsByCategory groups findings by category
	FindingsByCategory map[string][]assessmentv1alpha1.Finding `json:"findingsByCategory" yaml:"findingsByCategory"`

	// FindingsByStatus groups findings by status
	FindingsByStatus map[string][]assessmentv1alpha1.Finding `json:"findingsByStatus" yaml:"findingsByStatus"`
}

// ReportMetadata contains report metadata.
type ReportMetadata struct {
	// GeneratedAt is when the report was generated
	GeneratedAt time.Time `json:"generatedAt" yaml:"generatedAt"`

	// AssessmentName is the name of the ClusterAssessment CR
	AssessmentName string `json:"assessmentName" yaml:"assessmentName"`

	// Profile is the baseline profile used
	Profile string `json:"profile" yaml:"profile"`

	// OperatorVersion is the version of the operator
	OperatorVersion string `json:"operatorVersion" yaml:"operatorVersion"`
}

// GenerateJSON generates a JSON report from a ClusterAssessment.
func GenerateJSON(assessment *assessmentv1alpha1.ClusterAssessment) ([]byte, error) {
	report := buildReport(assessment)
	return json.MarshalIndent(report, "", "  ")
}

// GenerateYAML generates a YAML report from a ClusterAssessment.
func GenerateYAML(assessment *assessmentv1alpha1.ClusterAssessment) ([]byte, error) {
	report := buildReport(assessment)
	return yaml.Marshal(report)
}

// buildReport constructs the Report from a ClusterAssessment.
func buildReport(assessment *assessmentv1alpha1.ClusterAssessment) Report {
	report := Report{
		Metadata: ReportMetadata{
			GeneratedAt:     time.Now(),
			AssessmentName:  assessment.Name,
			Profile:         assessment.Spec.Profile,
			OperatorVersion: "1.0.0", // TODO: Get from build info
		},
		ClusterInfo:        assessment.Status.ClusterInfo,
		Summary:            assessment.Status.Summary,
		Findings:           assessment.Status.Findings,
		FindingsByCategory: make(map[string][]assessmentv1alpha1.Finding),
		FindingsByStatus:   make(map[string][]assessmentv1alpha1.Finding),
	}

	// Group findings by category
	for _, f := range assessment.Status.Findings {
		report.FindingsByCategory[f.Category] = append(report.FindingsByCategory[f.Category], f)
	}

	// Group findings by status
	for _, f := range assessment.Status.Findings {
		report.FindingsByStatus[string(f.Status)] = append(report.FindingsByStatus[string(f.Status)], f)
	}

	return report
}

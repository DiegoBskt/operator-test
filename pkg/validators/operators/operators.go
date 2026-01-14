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

package operators

import (
	"context"
	"fmt"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"sigs.k8s.io/controller-runtime/pkg/client"

	assessmentv1alpha1 "github.com/openshift-assessment/cluster-assessment-operator/api/v1alpha1"
	"github.com/openshift-assessment/cluster-assessment-operator/pkg/profiles"
	"github.com/openshift-assessment/cluster-assessment-operator/pkg/validator"
)

const (
	validatorName        = "operators"
	validatorDescription = "Validates ClusterServiceVersions (CSVs) for failed or pending operators"
	validatorCategory    = "Platform"
)

func init() {
	_ = validator.Register(&OperatorsValidator{})
}

// OperatorsValidator checks operator health via CSVs.
type OperatorsValidator struct{}

// Name returns the validator name.
func (v *OperatorsValidator) Name() string {
	return validatorName
}

// Description returns the validator description.
func (v *OperatorsValidator) Description() string {
	return validatorDescription
}

// Category returns the finding category.
func (v *OperatorsValidator) Category() string {
	return validatorCategory
}

// Validate performs operator health checks.
func (v *OperatorsValidator) Validate(ctx context.Context, c client.Client, profile profiles.Profile) ([]assessmentv1alpha1.Finding, error) {
	var findings []assessmentv1alpha1.Finding

	// Check ClusterServiceVersions
	csvGVK := schema.GroupVersionKind{
		Group:   "operators.coreos.com",
		Version: "v1alpha1",
		Kind:    "ClusterServiceVersionList",
	}

	csvList := &unstructured.UnstructuredList{}
	csvList.SetGroupVersionKind(csvGVK)

	if err := c.List(ctx, csvList); err != nil {
		findings = append(findings, assessmentv1alpha1.Finding{
			ID:          "operators-csv-error",
			Validator:   validatorName,
			Category:    validatorCategory,
			Status:      assessmentv1alpha1.FindingStatusInfo,
			Title:       "Unable to List CSVs",
			Description: fmt.Sprintf("Failed to list ClusterServiceVersions: %v", err),
		})
		return findings, nil
	}

	var failedCSVs []string
	var pendingCSVs []string
	var healthyCSVs int

	for _, csv := range csvList.Items {
		name, _, _ := unstructured.NestedString(csv.Object, "metadata", "name")
		namespace, _, _ := unstructured.NestedString(csv.Object, "metadata", "namespace")
		phase, _, _ := unstructured.NestedString(csv.Object, "status", "phase")

		fullName := fmt.Sprintf("%s/%s", namespace, name)

		switch phase {
		case "Succeeded":
			healthyCSVs++
		case "Failed":
			failedCSVs = append(failedCSVs, fullName)
		case "Pending", "Installing", "Replacing", "Deleting":
			pendingCSVs = append(pendingCSVs, fullName)
		default:
			// Unknown phase, treat as pending
			if phase != "" {
				pendingCSVs = append(pendingCSVs, fullName)
			}
		}
	}

	// Report on failed operators
	if len(failedCSVs) > 0 {
		findings = append(findings, assessmentv1alpha1.Finding{
			ID:             "operators-csv-failed",
			Validator:      validatorName,
			Category:       validatorCategory,
			Status:         assessmentv1alpha1.FindingStatusFail,
			Title:          "Failed Operators Detected",
			Description:    fmt.Sprintf("Found %d operators in Failed state: %v", len(failedCSVs), truncateList(failedCSVs, 5)),
			Impact:         "Failed operators may not provide expected functionality and could affect cluster operations.",
			Recommendation: "Check the operator logs and events to diagnose the failure. Consider removing and reinstalling the operator.",
		})
	}

	// Report on pending operators
	if len(pendingCSVs) > 0 {
		findings = append(findings, assessmentv1alpha1.Finding{
			ID:             "operators-csv-pending",
			Validator:      validatorName,
			Category:       validatorCategory,
			Status:         assessmentv1alpha1.FindingStatusWarn,
			Title:          "Operators in Pending State",
			Description:    fmt.Sprintf("Found %d operators pending installation: %v", len(pendingCSVs), truncateList(pendingCSVs, 5)),
			Impact:         "Pending operators may be waiting for dependencies or experiencing installation issues.",
			Recommendation: "Review the install plan and subscription status for blocked operators.",
		})
	}

	// Report healthy operators summary
	if len(failedCSVs) == 0 && len(pendingCSVs) == 0 {
		findings = append(findings, assessmentv1alpha1.Finding{
			ID:          "operators-csv-healthy",
			Validator:   validatorName,
			Category:    validatorCategory,
			Status:      assessmentv1alpha1.FindingStatusPass,
			Title:       "All Operators Healthy",
			Description: fmt.Sprintf("All %d installed operators are in Succeeded state.", healthyCSVs),
		})
	}

	// Check ClusterOperators
	findings = append(findings, v.checkClusterOperators(ctx, c)...)

	return findings, nil
}

// checkClusterOperators validates the built-in cluster operators.
func (v *OperatorsValidator) checkClusterOperators(ctx context.Context, c client.Client) []assessmentv1alpha1.Finding {
	var findings []assessmentv1alpha1.Finding

	coGVK := schema.GroupVersionKind{
		Group:   "config.openshift.io",
		Version: "v1",
		Kind:    "ClusterOperatorList",
	}

	coList := &unstructured.UnstructuredList{}
	coList.SetGroupVersionKind(coGVK)

	if err := c.List(ctx, coList); err != nil {
		return findings
	}

	var degradedOperators []string
	var unavailableOperators []string
	var progressingOperators []string

	for _, co := range coList.Items {
		name, _, _ := unstructured.NestedString(co.Object, "metadata", "name")
		conditions, found, _ := unstructured.NestedSlice(co.Object, "status", "conditions")
		if !found {
			continue
		}

		for _, cond := range conditions {
			condMap, ok := cond.(map[string]interface{})
			if !ok {
				continue
			}

			condType, _ := condMap["type"].(string)
			condStatus, _ := condMap["status"].(string)

			switch condType {
			case "Degraded":
				if condStatus == "True" {
					degradedOperators = append(degradedOperators, name)
				}
			case "Available":
				if condStatus == "False" {
					unavailableOperators = append(unavailableOperators, name)
				}
			case "Progressing":
				if condStatus == "True" {
					progressingOperators = append(progressingOperators, name)
				}
			}
		}
	}

	if len(degradedOperators) > 0 {
		findings = append(findings, assessmentv1alpha1.Finding{
			ID:             "operators-cluster-degraded",
			Validator:      validatorName,
			Category:       validatorCategory,
			Status:         assessmentv1alpha1.FindingStatusFail,
			Title:          "Degraded Cluster Operators",
			Description:    fmt.Sprintf("Found %d degraded cluster operators: %v", len(degradedOperators), degradedOperators),
			Impact:         "Degraded operators may not be fully functional and could affect cluster stability.",
			Recommendation: "Check the operator events and logs in the openshift-* namespaces.",
		})
	}

	if len(unavailableOperators) > 0 {
		findings = append(findings, assessmentv1alpha1.Finding{
			ID:             "operators-cluster-unavailable",
			Validator:      validatorName,
			Category:       validatorCategory,
			Status:         assessmentv1alpha1.FindingStatusFail,
			Title:          "Unavailable Cluster Operators",
			Description:    fmt.Sprintf("Found %d unavailable cluster operators: %v", len(unavailableOperators), unavailableOperators),
			Impact:         "Unavailable operators cannot perform their functions.",
			Recommendation: "Investigate the operator status and logs immediately.",
		})
	}

	if len(progressingOperators) > 0 {
		findings = append(findings, assessmentv1alpha1.Finding{
			ID:          "operators-cluster-progressing",
			Validator:   validatorName,
			Category:    validatorCategory,
			Status:      assessmentv1alpha1.FindingStatusInfo,
			Title:       "Cluster Operators Updating",
			Description: fmt.Sprintf("Found %d cluster operators currently updating: %v", len(progressingOperators), progressingOperators),
		})
	}

	if len(degradedOperators) == 0 && len(unavailableOperators) == 0 {
		findings = append(findings, assessmentv1alpha1.Finding{
			ID:          "operators-cluster-healthy",
			Validator:   validatorName,
			Category:    validatorCategory,
			Status:      assessmentv1alpha1.FindingStatusPass,
			Title:       "All Cluster Operators Healthy",
			Description: fmt.Sprintf("All %d cluster operators are available and not degraded.", len(coList.Items)),
		})
	}

	return findings
}

func truncateList(items []string, max int) []string {
	if len(items) <= max {
		return items
	}
	result := items[:max]
	result = append(result, fmt.Sprintf("... and %d more", len(items)-max))
	return result
}

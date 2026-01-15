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

package costoptimization

import (
	"context"
	"fmt"
	"strings"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	"sigs.k8s.io/controller-runtime/pkg/client"

	assessmentv1alpha1 "github.com/openshift-assessment/cluster-assessment-operator/api/v1alpha1"
	"github.com/openshift-assessment/cluster-assessment-operator/pkg/profiles"
	"github.com/openshift-assessment/cluster-assessment-operator/pkg/validator"
)

const (
	validatorName        = "costoptimization"
	validatorDescription = "Identifies resource optimization opportunities including orphan PVCs, idle deployments, and missing resource specifications"
	validatorCategory    = "Infrastructure"
)

func init() {
	_ = validator.Register(&CostOptimizationValidator{})
}

// CostOptimizationValidator checks for resource optimization opportunities.
type CostOptimizationValidator struct{}

// Name returns the validator name.
func (v *CostOptimizationValidator) Name() string {
	return validatorName
}

// Description returns the validator description.
func (v *CostOptimizationValidator) Description() string {
	return validatorDescription
}

// Category returns the finding category.
func (v *CostOptimizationValidator) Category() string {
	return validatorCategory
}

// Validate performs cost optimization checks.
func (v *CostOptimizationValidator) Validate(ctx context.Context, c client.Client, profile profiles.Profile) ([]assessmentv1alpha1.Finding, error) {
	var findings []assessmentv1alpha1.Finding

	// Check 1: Orphan PVCs
	findings = append(findings, v.checkOrphanPVCs(ctx, c)...)

	// Check 2: Idle deployments
	findings = append(findings, v.checkIdleDeployments(ctx, c)...)

	// Check 3: Pods without resource specifications
	findings = append(findings, v.checkResourceSpecifications(ctx, c)...)

	return findings, nil
}

// checkOrphanPVCs finds PVCs not bound to any pod.
func (v *CostOptimizationValidator) checkOrphanPVCs(ctx context.Context, c client.Client) []assessmentv1alpha1.Finding {
	var findings []assessmentv1alpha1.Finding

	// Get all PVCs
	pvcs := &corev1.PersistentVolumeClaimList{}
	if err := c.List(ctx, pvcs); err != nil {
		return []assessmentv1alpha1.Finding{{
			ID:          "costoptimization-pvc-error",
			Validator:   validatorName,
			Category:    validatorCategory,
			Status:      assessmentv1alpha1.FindingStatusFail,
			Title:       "Unable to Check PVCs",
			Description: fmt.Sprintf("Failed to list PVCs: %v", err),
		}}
	}

	// Get all pods to find which PVCs are in use
	pods := &corev1.PodList{}
	if err := c.List(ctx, pods); err != nil {
		return findings
	}

	// Build map of PVCs in use
	pvcInUse := make(map[string]bool)
	for _, pod := range pods.Items {
		for _, volume := range pod.Spec.Volumes {
			if volume.PersistentVolumeClaim != nil {
				key := fmt.Sprintf("%s/%s", pod.Namespace, volume.PersistentVolumeClaim.ClaimName)
				pvcInUse[key] = true
			}
		}
	}

	// Find orphan PVCs in user namespaces
	var orphanPVCs []string
	var totalOrphanSize resource.Quantity

	for _, pvc := range pvcs.Items {
		// Skip system namespaces
		if strings.HasPrefix(pvc.Namespace, "openshift-") || strings.HasPrefix(pvc.Namespace, "kube-") {
			continue
		}

		// Skip unbound PVCs (they're pending, not orphaned)
		if pvc.Status.Phase != corev1.ClaimBound {
			continue
		}

		key := fmt.Sprintf("%s/%s", pvc.Namespace, pvc.Name)
		if !pvcInUse[key] {
			orphanPVCs = append(orphanPVCs, key)
			if pvc.Status.Capacity != nil {
				if storage, ok := pvc.Status.Capacity[corev1.ResourceStorage]; ok {
					totalOrphanSize.Add(storage)
				}
			}
		}
	}

	if len(orphanPVCs) > 0 {
		sample := orphanPVCs
		if len(sample) > 5 {
			sample = sample[:5]
		}

		findings = append(findings, assessmentv1alpha1.Finding{
			ID:             "costoptimization-orphan-pvcs",
			Validator:      validatorName,
			Category:       validatorCategory,
			Status:         assessmentv1alpha1.FindingStatusWarn,
			Title:          "Orphan PVCs Detected",
			Description:    fmt.Sprintf("Found %d bound PVC(s) not attached to any pod (total size: %s): %s...", len(orphanPVCs), totalOrphanSize.String(), strings.Join(sample, ", ")),
			Impact:         "Orphan PVCs consume storage resources without being used.",
			Recommendation: "Review orphan PVCs and delete those no longer needed.",
		})
	} else {
		findings = append(findings, assessmentv1alpha1.Finding{
			ID:          "costoptimization-no-orphan-pvcs",
			Validator:   validatorName,
			Category:    validatorCategory,
			Status:      assessmentv1alpha1.FindingStatusPass,
			Title:       "No Orphan PVCs",
			Description: "All bound PVCs are attached to running pods.",
		})
	}

	return findings
}

// checkIdleDeployments finds deployments scaled to 0.
func (v *CostOptimizationValidator) checkIdleDeployments(ctx context.Context, c client.Client) []assessmentv1alpha1.Finding {
	var findings []assessmentv1alpha1.Finding

	deployments := &appsv1.DeploymentList{}
	if err := c.List(ctx, deployments); err != nil {
		return findings
	}

	var idleDeployments []string

	for _, deploy := range deployments.Items {
		// Skip system namespaces
		if strings.HasPrefix(deploy.Namespace, "openshift-") || strings.HasPrefix(deploy.Namespace, "kube-") {
			continue
		}

		// Check if scaled to 0
		if deploy.Spec.Replicas != nil && *deploy.Spec.Replicas == 0 {
			idleDeployments = append(idleDeployments, fmt.Sprintf("%s/%s", deploy.Namespace, deploy.Name))
		}
	}

	if len(idleDeployments) > 0 {
		sample := idleDeployments
		if len(sample) > 5 {
			sample = sample[:5]
		}

		findings = append(findings, assessmentv1alpha1.Finding{
			ID:             "costoptimization-idle-deployments",
			Validator:      validatorName,
			Category:       validatorCategory,
			Status:         assessmentv1alpha1.FindingStatusInfo,
			Title:          "Idle Deployments",
			Description:    fmt.Sprintf("Found %d deployment(s) scaled to 0 replicas: %s...", len(idleDeployments), strings.Join(sample, ", ")),
			Impact:         "Idle deployments may indicate unused applications or forgotten test resources.",
			Recommendation: "Review idle deployments and delete those no longer needed.",
		})
	}

	return findings
}

// checkResourceSpecifications finds pods without resource requests/limits.
func (v *CostOptimizationValidator) checkResourceSpecifications(ctx context.Context, c client.Client) []assessmentv1alpha1.Finding {
	var findings []assessmentv1alpha1.Finding

	pods := &corev1.PodList{}
	if err := c.List(ctx, pods); err != nil {
		return findings
	}

	var podsWithoutRequests []string
	var podsWithoutLimits []string

	for _, pod := range pods.Items {
		// Skip system namespaces
		if strings.HasPrefix(pod.Namespace, "openshift-") || strings.HasPrefix(pod.Namespace, "kube-") {
			continue
		}

		// Skip completed/failed pods
		if pod.Status.Phase == corev1.PodSucceeded || pod.Status.Phase == corev1.PodFailed {
			continue
		}

		hasRequests := true
		hasLimits := true

		for _, container := range pod.Spec.Containers {
			// Check requests
			if container.Resources.Requests == nil ||
				(container.Resources.Requests.Cpu().IsZero() && container.Resources.Requests.Memory().IsZero()) {
				hasRequests = false
			}
			// Check limits
			if container.Resources.Limits == nil ||
				(container.Resources.Limits.Cpu().IsZero() && container.Resources.Limits.Memory().IsZero()) {
				hasLimits = false
			}
		}

		if !hasRequests {
			podsWithoutRequests = append(podsWithoutRequests, fmt.Sprintf("%s/%s", pod.Namespace, pod.Name))
		}
		if !hasLimits {
			podsWithoutLimits = append(podsWithoutLimits, fmt.Sprintf("%s/%s", pod.Namespace, pod.Name))
		}
	}

	// Report pods without requests
	if len(podsWithoutRequests) > 0 {
		sample := podsWithoutRequests
		if len(sample) > 5 {
			sample = sample[:5]
		}

		findings = append(findings, assessmentv1alpha1.Finding{
			ID:             "costoptimization-no-requests",
			Validator:      validatorName,
			Category:       validatorCategory,
			Status:         assessmentv1alpha1.FindingStatusWarn,
			Title:          "Pods Without Resource Requests",
			Description:    fmt.Sprintf("Found %d pod(s) without CPU/memory requests: %s...", len(podsWithoutRequests), strings.Join(sample, ", ")),
			Impact:         "Pods without resource requests may cause scheduling and resource management issues.",
			Recommendation: "Define resource requests for all production workloads.",
			References: []string{
				"https://kubernetes.io/docs/concepts/configuration/manage-resources-containers/",
			},
		})
	} else {
		findings = append(findings, assessmentv1alpha1.Finding{
			ID:          "costoptimization-requests-defined",
			Validator:   validatorName,
			Category:    validatorCategory,
			Status:      assessmentv1alpha1.FindingStatusPass,
			Title:       "All Pods Have Resource Requests",
			Description: "All running pods have CPU/memory requests defined.",
		})
	}

	// Report pods without limits
	if len(podsWithoutLimits) > 0 {
		sample := podsWithoutLimits
		if len(sample) > 5 {
			sample = sample[:5]
		}

		findings = append(findings, assessmentv1alpha1.Finding{
			ID:             "costoptimization-no-limits",
			Validator:      validatorName,
			Category:       validatorCategory,
			Status:         assessmentv1alpha1.FindingStatusInfo,
			Title:          "Pods Without Resource Limits",
			Description:    fmt.Sprintf("Found %d pod(s) without CPU/memory limits: %s...", len(podsWithoutLimits), strings.Join(sample, ", ")),
			Impact:         "Pods without limits can consume all available node resources.",
			Recommendation: "Consider defining resource limits or using LimitRanges.",
		})
	}

	return findings
}

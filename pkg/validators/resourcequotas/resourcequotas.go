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

package resourcequotas

import (
	"context"
	"fmt"
	"strings"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"sigs.k8s.io/controller-runtime/pkg/client"

	assessmentv1alpha1 "github.com/openshift-assessment/cluster-assessment-operator/api/v1alpha1"
	"github.com/openshift-assessment/cluster-assessment-operator/pkg/profiles"
	"github.com/openshift-assessment/cluster-assessment-operator/pkg/validator"
)

const (
	validatorName        = "resourcequotas"
	validatorDescription = "Validates namespace resource governance including ResourceQuotas and LimitRanges"
	validatorCategory    = "Governance"
)

func init() {
	_ = validator.Register(&ResourceQuotasValidator{})
}

// ResourceQuotasValidator checks resource quota and limit range configuration.
type ResourceQuotasValidator struct{}

// Name returns the validator name.
func (v *ResourceQuotasValidator) Name() string {
	return validatorName
}

// Description returns the validator description.
func (v *ResourceQuotasValidator) Description() string {
	return validatorDescription
}

// Category returns the finding category.
func (v *ResourceQuotasValidator) Category() string {
	return validatorCategory
}

// Validate performs resource governance checks.
func (v *ResourceQuotasValidator) Validate(ctx context.Context, c client.Client, profile profiles.Profile) ([]assessmentv1alpha1.Finding, error) {
	var findings []assessmentv1alpha1.Finding

	// Optimized: List namespaces once using PartialObjectMetadataList
	nsList := &metav1.PartialObjectMetadataList{}
	nsList.SetGroupVersionKind(schema.GroupVersionKind{
		Group:   "",
		Version: "v1",
		Kind:    "NamespaceList",
	})

	if err := c.List(ctx, nsList); err != nil {
		return []assessmentv1alpha1.Finding{{
			ID:          "resourcequotas-ns-error",
			Validator:   validatorName,
			Category:    validatorCategory,
			Status:      assessmentv1alpha1.FindingStatusFail,
			Title:       "Unable to Check Namespaces",
			Description: fmt.Sprintf("Failed to list namespaces: %v", err),
		}}, nil
	}

	var userNamespaces []string
	for _, ns := range nsList.Items {
		// Skip system namespaces
		if strings.HasPrefix(ns.Name, "openshift-") || strings.HasPrefix(ns.Name, "kube-") || ns.Name == "default" {
			continue
		}
		userNamespaces = append(userNamespaces, ns.Name)
	}

	// Check 1: ResourceQuota coverage
	findings = append(findings, v.checkResourceQuotas(ctx, c, profile, userNamespaces)...)

	// Check 2: LimitRange coverage
	findings = append(findings, v.checkLimitRanges(ctx, c, profile, userNamespaces)...)

	return findings, nil
}

// checkResourceQuotas checks ResourceQuota configuration across namespaces.
func (v *ResourceQuotasValidator) checkResourceQuotas(ctx context.Context, c client.Client, profile profiles.Profile, userNamespaces []string) []assessmentv1alpha1.Finding {
	var findings []assessmentv1alpha1.Finding

	// Get all ResourceQuotas
	quotas := &corev1.ResourceQuotaList{}
	if err := c.List(ctx, quotas); err != nil {
		return []assessmentv1alpha1.Finding{{
			ID:          "resourcequotas-list-error",
			Validator:   validatorName,
			Category:    validatorCategory,
			Status:      assessmentv1alpha1.FindingStatusFail,
			Title:       "Unable to Check ResourceQuotas",
			Description: fmt.Sprintf("Failed to list ResourceQuotas: %v", err),
		}}
	}

	// Build map of namespaces with quotas
	nsWithQuota := make(map[string][]corev1.ResourceQuota)
	for _, quota := range quotas.Items {
		nsWithQuota[quota.Namespace] = append(nsWithQuota[quota.Namespace], quota)
	}

	var userNamespacesWithoutQuota []string
	var nearLimitQuotas []string

	for _, nsName := range userNamespaces {
		quotasInNs, hasQuota := nsWithQuota[nsName]
		if !hasQuota {
			userNamespacesWithoutQuota = append(userNamespacesWithoutQuota, nsName)
		} else {
			// Check quota utilization
			for _, quota := range quotasInNs {
				for resourceName, hard := range quota.Status.Hard {
					used, ok := quota.Status.Used[resourceName]
					if !ok {
						continue
					}

					// Calculate utilization percentage
					if hard.Value() > 0 {
						utilization := float64(used.Value()) / float64(hard.Value()) * 100
						if utilization >= 80 {
							nearLimitQuotas = append(nearLimitQuotas,
								fmt.Sprintf("%s/%s (%s: %.0f%%)", nsName, quota.Name, resourceName, utilization))
						}
					}
				}
			}
		}
	}

	// Report quota coverage
	totalUserNs := len(userNamespacesWithoutQuota) + len(nsWithQuota)
	if len(userNamespacesWithoutQuota) > 0 {
		status := assessmentv1alpha1.FindingStatusInfo
		if profile.Thresholds.RequireResourceQuotas {
			status = assessmentv1alpha1.FindingStatusWarn
		}

		sample := userNamespacesWithoutQuota
		if len(sample) > 5 {
			sample = sample[:5]
		}

		findings = append(findings, assessmentv1alpha1.Finding{
			ID:             "resourcequotas-coverage",
			Validator:      validatorName,
			Category:       validatorCategory,
			Status:         status,
			Title:          "Namespaces Without ResourceQuotas",
			Description:    fmt.Sprintf("%d of %d user namespace(s) have no ResourceQuota: %s...", len(userNamespacesWithoutQuota), totalUserNs, strings.Join(sample, ", ")),
			Impact:         "Namespaces without quotas can consume unbounded cluster resources.",
			Recommendation: "Define ResourceQuotas for user namespaces to prevent resource exhaustion.",
			References: []string{
				"https://kubernetes.io/docs/concepts/policy/resource-quotas/",
			},
		})
	} else if totalUserNs > 0 {
		findings = append(findings, assessmentv1alpha1.Finding{
			ID:          "resourcequotas-full-coverage",
			Validator:   validatorName,
			Category:    validatorCategory,
			Status:      assessmentv1alpha1.FindingStatusPass,
			Title:       "All User Namespaces Have ResourceQuotas",
			Description: fmt.Sprintf("All %d user namespace(s) have ResourceQuotas defined.", totalUserNs),
		})
	}

	// Report near-limit quotas
	if len(nearLimitQuotas) > 0 {
		sample := nearLimitQuotas
		if len(sample) > 5 {
			sample = sample[:5]
		}

		findings = append(findings, assessmentv1alpha1.Finding{
			ID:             "resourcequotas-near-limit",
			Validator:      validatorName,
			Category:       validatorCategory,
			Status:         assessmentv1alpha1.FindingStatusWarn,
			Title:          "ResourceQuotas Near Limit",
			Description:    fmt.Sprintf("%d ResourceQuota(s) are at or above 80%% utilization: %s", len(nearLimitQuotas), strings.Join(sample, ", ")),
			Impact:         "Workloads may be unable to scale or deploy new pods.",
			Recommendation: "Review and increase quota limits or optimize resource usage.",
		})
	}

	return findings
}

// checkLimitRanges checks LimitRange configuration across namespaces.
func (v *ResourceQuotasValidator) checkLimitRanges(ctx context.Context, c client.Client, profile profiles.Profile, userNamespaces []string) []assessmentv1alpha1.Finding {
	var findings []assessmentv1alpha1.Finding

	// Get all LimitRanges
	limitRanges := &corev1.LimitRangeList{}
	if err := c.List(ctx, limitRanges); err != nil {
		return findings
	}

	// Build map of namespaces with LimitRanges
	nsWithLimitRange := make(map[string]bool)
	for _, lr := range limitRanges.Items {
		nsWithLimitRange[lr.Namespace] = true
	}

	var userNamespacesWithoutLR []string
	var veryHighDefaultLimits []string

	for _, nsName := range userNamespaces {
		if !nsWithLimitRange[nsName] {
			userNamespacesWithoutLR = append(userNamespacesWithoutLR, nsName)
		}
	}

	// Check for very high default limits
	// Optimization: Hoist invariant parsing out of loop
	eightGi := resource.MustParse("8Gi")
	for _, lr := range limitRanges.Items {
		for _, item := range lr.Spec.Limits {
			if item.Type == corev1.LimitTypeContainer {
				if defaultMem, ok := item.Default[corev1.ResourceMemory]; ok {
					// Check if default memory is > 8Gi
					if defaultMem.Cmp(eightGi) > 0 {
						veryHighDefaultLimits = append(veryHighDefaultLimits,
							fmt.Sprintf("%s/%s (default memory: %s)", lr.Namespace, lr.Name, defaultMem.String()))
					}
				}
			}
		}
	}

	// Report LimitRange coverage
	totalUserNs := len(userNamespacesWithoutLR) + len(nsWithLimitRange)
	if len(userNamespacesWithoutLR) > 0 {
		status := assessmentv1alpha1.FindingStatusInfo
		if profile.Thresholds.RequireLimitRanges {
			status = assessmentv1alpha1.FindingStatusWarn
		}

		sample := userNamespacesWithoutLR
		if len(sample) > 5 {
			sample = sample[:5]
		}

		findings = append(findings, assessmentv1alpha1.Finding{
			ID:             "resourcequotas-limitrange-missing",
			Validator:      validatorName,
			Category:       validatorCategory,
			Status:         status,
			Title:          "Namespaces Without LimitRanges",
			Description:    fmt.Sprintf("%d of %d user namespace(s) have no LimitRange: %s...", len(userNamespacesWithoutLR), totalUserNs, strings.Join(sample, ", ")),
			Impact:         "Containers without limits may consume all available node resources.",
			Recommendation: "Define LimitRanges to set default CPU/memory limits for containers.",
			References: []string{
				"https://kubernetes.io/docs/concepts/policy/limit-range/",
			},
		})
	} else if totalUserNs > 0 {
		findings = append(findings, assessmentv1alpha1.Finding{
			ID:          "resourcequotas-limitrange-coverage",
			Validator:   validatorName,
			Category:    validatorCategory,
			Status:      assessmentv1alpha1.FindingStatusPass,
			Title:       "All User Namespaces Have LimitRanges",
			Description: fmt.Sprintf("All %d user namespace(s) have LimitRanges defined.", totalUserNs),
		})
	}

	// Report very high default limits
	if len(veryHighDefaultLimits) > 0 {
		findings = append(findings, assessmentv1alpha1.Finding{
			ID:             "resourcequotas-high-defaults",
			Validator:      validatorName,
			Category:       validatorCategory,
			Status:         assessmentv1alpha1.FindingStatusInfo,
			Title:          "LimitRanges with Very High Defaults",
			Description:    fmt.Sprintf("%d LimitRange(s) have default memory > 8Gi: %s", len(veryHighDefaultLimits), strings.Join(veryHighDefaultLimits, ", ")),
			Impact:         "High default limits may lead to inefficient resource allocation.",
			Recommendation: "Review default limits to ensure they match expected workload requirements.",
		})
	}

	return findings
}

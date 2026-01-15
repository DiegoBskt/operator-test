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

package networkpolicyaudit

import (
	"context"
	"fmt"
	"strings"

	corev1 "k8s.io/api/core/v1"
	networkingv1 "k8s.io/api/networking/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	assessmentv1alpha1 "github.com/openshift-assessment/cluster-assessment-operator/api/v1alpha1"
	"github.com/openshift-assessment/cluster-assessment-operator/pkg/profiles"
	"github.com/openshift-assessment/cluster-assessment-operator/pkg/validator"
)

const (
	validatorName        = "networkpolicyaudit"
	validatorDescription = "Audits NetworkPolicy configuration including coverage, allow-all detection, and policy effectiveness"
	validatorCategory    = "Networking"
)

func init() {
	_ = validator.Register(&NetworkPolicyAuditValidator{})
}

// NetworkPolicyAuditValidator audits NetworkPolicy configuration.
type NetworkPolicyAuditValidator struct{}

// Name returns the validator name.
func (v *NetworkPolicyAuditValidator) Name() string {
	return validatorName
}

// Description returns the validator description.
func (v *NetworkPolicyAuditValidator) Description() string {
	return validatorDescription
}

// Category returns the finding category.
func (v *NetworkPolicyAuditValidator) Category() string {
	return validatorCategory
}

// Validate performs NetworkPolicy audit checks.
func (v *NetworkPolicyAuditValidator) Validate(ctx context.Context, c client.Client, profile profiles.Profile) ([]assessmentv1alpha1.Finding, error) {
	var findings []assessmentv1alpha1.Finding

	// Check 1: NetworkPolicy coverage
	findings = append(findings, v.checkNetworkPolicyCoverage(ctx, c, profile)...)

	// Check 2: Allow-all policies
	findings = append(findings, v.checkAllowAllPolicies(ctx, c)...)

	// Check 3: Default deny policies
	findings = append(findings, v.checkDefaultDenyPolicies(ctx, c)...)

	return findings, nil
}

// checkNetworkPolicyCoverage checks which namespaces have NetworkPolicies.
func (v *NetworkPolicyAuditValidator) checkNetworkPolicyCoverage(ctx context.Context, c client.Client, profile profiles.Profile) []assessmentv1alpha1.Finding {
	var findings []assessmentv1alpha1.Finding

	// Get all namespaces
	namespaces := &corev1.NamespaceList{}
	if err := c.List(ctx, namespaces); err != nil {
		return []assessmentv1alpha1.Finding{{
			ID:          "networkpolicyaudit-ns-error",
			Validator:   validatorName,
			Category:    validatorCategory,
			Status:      assessmentv1alpha1.FindingStatusFail,
			Title:       "Unable to Check Namespaces",
			Description: fmt.Sprintf("Failed to list namespaces: %v", err),
		}}
	}

	// Get all NetworkPolicies
	networkPolicies := &networkingv1.NetworkPolicyList{}
	if err := c.List(ctx, networkPolicies); err != nil {
		return []assessmentv1alpha1.Finding{{
			ID:          "networkpolicyaudit-list-error",
			Validator:   validatorName,
			Category:    validatorCategory,
			Status:      assessmentv1alpha1.FindingStatusFail,
			Title:       "Unable to Check NetworkPolicies",
			Description: fmt.Sprintf("Failed to list NetworkPolicies: %v", err),
		}}
	}

	// Build map of namespaces with policies
	nsWithPolicy := make(map[string]int)
	for _, np := range networkPolicies.Items {
		nsWithPolicy[np.Namespace]++
	}

	var userNamespacesWithoutPolicy []string
	var userNamespacesWithPolicy []string

	for _, ns := range namespaces.Items {
		// Skip system namespaces
		if strings.HasPrefix(ns.Name, "openshift-") || strings.HasPrefix(ns.Name, "kube-") || ns.Name == "default" {
			continue
		}

		if nsWithPolicy[ns.Name] == 0 {
			userNamespacesWithoutPolicy = append(userNamespacesWithoutPolicy, ns.Name)
		} else {
			userNamespacesWithPolicy = append(userNamespacesWithPolicy, ns.Name)
		}
	}

	totalUserNs := len(userNamespacesWithoutPolicy) + len(userNamespacesWithPolicy)

	// Report coverage
	if len(userNamespacesWithoutPolicy) > 0 {
		status := assessmentv1alpha1.FindingStatusInfo
		if profile.Thresholds.RequireNetworkPolicy {
			status = assessmentv1alpha1.FindingStatusWarn
		}

		sample := userNamespacesWithoutPolicy
		if len(sample) > 5 {
			sample = sample[:5]
		}

		coveragePercent := 0
		if totalUserNs > 0 {
			coveragePercent = len(userNamespacesWithPolicy) * 100 / totalUserNs
		}

		findings = append(findings, assessmentv1alpha1.Finding{
			ID:             "networkpolicyaudit-coverage",
			Validator:      validatorName,
			Category:       validatorCategory,
			Status:         status,
			Title:          "NetworkPolicy Coverage",
			Description:    fmt.Sprintf("%d%% of user namespaces have NetworkPolicies (%d/%d). Without: %s...", coveragePercent, len(userNamespacesWithPolicy), totalUserNs, strings.Join(sample, ", ")),
			Impact:         "Namespaces without NetworkPolicies allow all pod-to-pod traffic.",
			Recommendation: "Define NetworkPolicies for user namespaces to implement network segmentation.",
			References: []string{
				"https://kubernetes.io/docs/concepts/services-networking/network-policies/",
			},
		})
	} else if totalUserNs > 0 {
		findings = append(findings, assessmentv1alpha1.Finding{
			ID:          "networkpolicyaudit-full-coverage",
			Validator:   validatorName,
			Category:    validatorCategory,
			Status:      assessmentv1alpha1.FindingStatusPass,
			Title:       "Full NetworkPolicy Coverage",
			Description: fmt.Sprintf("All %d user namespace(s) have NetworkPolicies defined.", totalUserNs),
		})
	}

	return findings
}

// checkAllowAllPolicies detects overly permissive NetworkPolicies.
func (v *NetworkPolicyAuditValidator) checkAllowAllPolicies(ctx context.Context, c client.Client) []assessmentv1alpha1.Finding {
	var findings []assessmentv1alpha1.Finding

	networkPolicies := &networkingv1.NetworkPolicyList{}
	if err := c.List(ctx, networkPolicies); err != nil {
		return findings
	}

	var allowAllIngress []string
	var allowAllEgress []string

	for _, np := range networkPolicies.Items {
		// Skip system namespaces
		if strings.HasPrefix(np.Namespace, "openshift-") || strings.HasPrefix(np.Namespace, "kube-") {
			continue
		}

		// Check for allow-all ingress
		for _, ingress := range np.Spec.Ingress {
			if len(ingress.From) == 0 && len(ingress.Ports) == 0 {
				// Empty From and Ports means allow all
				allowAllIngress = append(allowAllIngress, fmt.Sprintf("%s/%s", np.Namespace, np.Name))
				break
			}
			// Check for empty podSelector and namespaceSelector (allows from anywhere)
			for _, from := range ingress.From {
				if from.PodSelector != nil && len(from.PodSelector.MatchLabels) == 0 && len(from.PodSelector.MatchExpressions) == 0 {
					if from.NamespaceSelector != nil && len(from.NamespaceSelector.MatchLabels) == 0 && len(from.NamespaceSelector.MatchExpressions) == 0 {
						allowAllIngress = append(allowAllIngress, fmt.Sprintf("%s/%s", np.Namespace, np.Name))
						break
					}
				}
			}
		}

		// Check for allow-all egress
		for _, egress := range np.Spec.Egress {
			if len(egress.To) == 0 && len(egress.Ports) == 0 {
				// Empty To and Ports means allow all
				allowAllEgress = append(allowAllEgress, fmt.Sprintf("%s/%s", np.Namespace, np.Name))
				break
			}
		}
	}

	// Report allow-all ingress policies
	if len(allowAllIngress) > 0 {
		sample := allowAllIngress
		if len(sample) > 5 {
			sample = sample[:5]
		}

		findings = append(findings, assessmentv1alpha1.Finding{
			ID:             "networkpolicyaudit-allow-all-ingress",
			Validator:      validatorName,
			Category:       validatorCategory,
			Status:         assessmentv1alpha1.FindingStatusWarn,
			Title:          "Allow-All Ingress NetworkPolicies",
			Description:    fmt.Sprintf("Found %d NetworkPolicy(ies) that allow all ingress traffic: %s", len(allowAllIngress), strings.Join(sample, ", ")),
			Impact:         "Overly permissive policies may not provide meaningful network isolation.",
			Recommendation: "Review and tighten NetworkPolicies to allow only necessary traffic.",
		})
	}

	// Report allow-all egress policies
	if len(allowAllEgress) > 0 {
		sample := allowAllEgress
		if len(sample) > 5 {
			sample = sample[:5]
		}

		findings = append(findings, assessmentv1alpha1.Finding{
			ID:             "networkpolicyaudit-allow-all-egress",
			Validator:      validatorName,
			Category:       validatorCategory,
			Status:         assessmentv1alpha1.FindingStatusInfo,
			Title:          "Allow-All Egress NetworkPolicies",
			Description:    fmt.Sprintf("Found %d NetworkPolicy(ies) that allow all egress traffic: %s", len(allowAllEgress), strings.Join(sample, ", ")),
			Impact:         "Pods can connect to any destination, including external networks.",
			Recommendation: "Consider restricting egress to known destinations for sensitive workloads.",
		})
	}

	return findings
}

// checkDefaultDenyPolicies checks for default deny policies.
func (v *NetworkPolicyAuditValidator) checkDefaultDenyPolicies(ctx context.Context, c client.Client) []assessmentv1alpha1.Finding {
	var findings []assessmentv1alpha1.Finding

	networkPolicies := &networkingv1.NetworkPolicyList{}
	if err := c.List(ctx, networkPolicies); err != nil {
		return findings
	}

	var namespacesWithDenyAll []string
	seenNamespaces := make(map[string]bool)

	for _, np := range networkPolicies.Items {
		// Skip system namespaces
		if strings.HasPrefix(np.Namespace, "openshift-") || strings.HasPrefix(np.Namespace, "kube-") {
			continue
		}

		if seenNamespaces[np.Namespace] {
			continue
		}

		// Check for default deny policy:
		// - Empty podSelector (applies to all pods)
		// - PolicyTypes includes Ingress
		// - No Ingress rules (denies all ingress)
		if len(np.Spec.PodSelector.MatchLabels) == 0 && len(np.Spec.PodSelector.MatchExpressions) == 0 {
			hasDenyIngress := false
			hasDenyEgress := false

			for _, policyType := range np.Spec.PolicyTypes {
				if policyType == networkingv1.PolicyTypeIngress && len(np.Spec.Ingress) == 0 {
					hasDenyIngress = true
				}
				if policyType == networkingv1.PolicyTypeEgress && len(np.Spec.Egress) == 0 {
					hasDenyEgress = true
				}
			}

			if hasDenyIngress || hasDenyEgress {
				namespacesWithDenyAll = append(namespacesWithDenyAll, np.Namespace)
				seenNamespaces[np.Namespace] = true
			}
		}
	}

	if len(namespacesWithDenyAll) > 0 {
		findings = append(findings, assessmentv1alpha1.Finding{
			ID:          "networkpolicyaudit-deny-default",
			Validator:   validatorName,
			Category:    validatorCategory,
			Status:      assessmentv1alpha1.FindingStatusPass,
			Title:       "Default Deny Policies Found",
			Description: fmt.Sprintf("%d namespace(s) have default deny NetworkPolicies: %s", len(namespacesWithDenyAll), strings.Join(namespacesWithDenyAll, ", ")),
		})
	} else {
		findings = append(findings, assessmentv1alpha1.Finding{
			ID:             "networkpolicyaudit-no-deny-default",
			Validator:      validatorName,
			Category:       validatorCategory,
			Status:         assessmentv1alpha1.FindingStatusInfo,
			Title:          "No Default Deny Policies",
			Description:    "No namespaces have default-deny NetworkPolicies configured.",
			Impact:         "Without default deny, pods accept traffic unless explicitly blocked.",
			Recommendation: "Consider implementing default-deny policies with explicit allow rules.",
			References: []string{
				"https://kubernetes.io/docs/concepts/services-networking/network-policies/#default-deny-all-ingress-traffic",
			},
		})
	}

	return findings
}

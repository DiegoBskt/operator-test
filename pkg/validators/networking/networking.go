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

package networking

import (
	"context"
	"fmt"
	"strings"

	configv1 "github.com/openshift/api/config/v1"
	networkingv1 "k8s.io/api/networking/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	assessmentv1alpha1 "github.com/openshift-assessment/cluster-assessment-operator/api/v1alpha1"
	"github.com/openshift-assessment/cluster-assessment-operator/pkg/profiles"
	"github.com/openshift-assessment/cluster-assessment-operator/pkg/validator"
)

const (
	validatorName        = "networking"
	validatorDescription = "Validates networking configuration including CNI and network policies"
	validatorCategory    = "Networking"
)

func init() {
	_ = validator.Register(&NetworkingValidator{})
}

// NetworkingValidator checks networking configuration.
type NetworkingValidator struct{}

// Name returns the validator name.
func (v *NetworkingValidator) Name() string {
	return validatorName
}

// Description returns the validator description.
func (v *NetworkingValidator) Description() string {
	return validatorDescription
}

// Category returns the finding category.
func (v *NetworkingValidator) Category() string {
	return validatorCategory
}

// Validate performs networking checks.
func (v *NetworkingValidator) Validate(ctx context.Context, c client.Client, profile profiles.Profile) ([]assessmentv1alpha1.Finding, error) {
	var findings []assessmentv1alpha1.Finding

	// Check 1: Network configuration
	findings = append(findings, v.checkNetworkConfig(ctx, c)...)

	// Check 2: Network policies
	findings = append(findings, v.checkNetworkPolicies(ctx, c, profile)...)

	// Check 3: Ingress configuration
	findings = append(findings, v.checkIngressConfig(ctx, c)...)

	return findings, nil
}

// checkNetworkConfig validates the cluster network configuration.
func (v *NetworkingValidator) checkNetworkConfig(ctx context.Context, c client.Client) []assessmentv1alpha1.Finding {
	var findings []assessmentv1alpha1.Finding

	network := &configv1.Network{}
	if err := c.Get(ctx, client.ObjectKey{Name: "cluster"}, network); err != nil {
		return []assessmentv1alpha1.Finding{{
			ID:          "networking-config-error",
			Validator:   validatorName,
			Category:    validatorCategory,
			Status:      assessmentv1alpha1.FindingStatusFail,
			Title:       "Unable to Check Network Configuration",
			Description: fmt.Sprintf("Failed to get Network configuration: %v", err),
		}}
	}

	// Report network type
	networkType := string(network.Status.NetworkType)
	findings = append(findings, assessmentv1alpha1.Finding{
		ID:          "networking-type",
		Validator:   validatorName,
		Category:    validatorCategory,
		Status:      assessmentv1alpha1.FindingStatusInfo,
		Title:       "Cluster Network Type",
		Description: fmt.Sprintf("Cluster is using %s networking.", networkType),
	})

	// Check for supported network types
	supportedTypes := []string{"OpenShiftSDN", "OVNKubernetes"}
	isSupported := false
	for _, t := range supportedTypes {
		if strings.EqualFold(networkType, t) {
			isSupported = true
			break
		}
	}

	if !isSupported {
		findings = append(findings, assessmentv1alpha1.Finding{
			ID:             "networking-unsupported-type",
			Validator:      validatorName,
			Category:       validatorCategory,
			Status:         assessmentv1alpha1.FindingStatusWarn,
			Title:          "Non-Standard Network Type",
			Description:    fmt.Sprintf("Cluster is using %s, which is not one of the standard OpenShift network types.", networkType),
			Impact:         "Non-standard network types may have different support levels and capabilities.",
			Recommendation: "Consider using OpenShiftSDN or OVNKubernetes for full OpenShift support.",
		})
	} else {
		findings = append(findings, assessmentv1alpha1.Finding{
			ID:          "networking-supported-type",
			Validator:   validatorName,
			Category:    validatorCategory,
			Status:      assessmentv1alpha1.FindingStatusPass,
			Title:       "Supported Network Type",
			Description: fmt.Sprintf("Cluster is using supported %s networking.", networkType),
		})
	}

	// Report cluster networks
	if len(network.Status.ClusterNetwork) > 0 {
		var cidrs []string
		for _, cn := range network.Status.ClusterNetwork {
			cidrs = append(cidrs, cn.CIDR)
		}
		findings = append(findings, assessmentv1alpha1.Finding{
			ID:          "networking-cluster-cidr",
			Validator:   validatorName,
			Category:    validatorCategory,
			Status:      assessmentv1alpha1.FindingStatusInfo,
			Title:       "Cluster Network CIDRs",
			Description: fmt.Sprintf("Cluster network CIDRs: %s", strings.Join(cidrs, ", ")),
		})
	}

	// Report service network
	if len(network.Status.ServiceNetwork) > 0 {
		findings = append(findings, assessmentv1alpha1.Finding{
			ID:          "networking-service-cidr",
			Validator:   validatorName,
			Category:    validatorCategory,
			Status:      assessmentv1alpha1.FindingStatusInfo,
			Title:       "Service Network CIDRs",
			Description: fmt.Sprintf("Service network CIDRs: %s", strings.Join(network.Status.ServiceNetwork, ", ")),
		})
	}

	return findings
}

// checkNetworkPolicies validates NetworkPolicy usage.
func (v *NetworkingValidator) checkNetworkPolicies(ctx context.Context, c client.Client, profile profiles.Profile) []assessmentv1alpha1.Finding {
	var findings []assessmentv1alpha1.Finding

	policies := &networkingv1.NetworkPolicyList{}
	if err := c.List(ctx, policies); err != nil {
		return []assessmentv1alpha1.Finding{{
			ID:          "networking-policies-error",
			Validator:   validatorName,
			Category:    validatorCategory,
			Status:      assessmentv1alpha1.FindingStatusInfo,
			Title:       "Unable to Check NetworkPolicies",
			Description: fmt.Sprintf("Failed to list NetworkPolicies: %v", err),
		}}
	}

	if len(policies.Items) == 0 {
		status := assessmentv1alpha1.FindingStatusInfo
		if profile.Thresholds.RequireNetworkPolicy {
			status = assessmentv1alpha1.FindingStatusWarn
		}
		findings = append(findings, assessmentv1alpha1.Finding{
			ID:             "networking-no-policies",
			Validator:      validatorName,
			Category:       validatorCategory,
			Status:         status,
			Title:          "No NetworkPolicies Configured",
			Description:    "No NetworkPolicies are configured in the cluster.",
			Impact:         "Without NetworkPolicies, all pods can communicate with each other without restrictions.",
			Recommendation: "Consider implementing NetworkPolicies to restrict pod-to-pod communication based on your security requirements.",
			References: []string{
				"https://docs.openshift.com/container-platform/latest/networking/network_policy/about-network-policy.html",
			},
		})
	} else {
		// Count policies per namespace
		policyCount := make(map[string]int)
		for _, policy := range policies.Items {
			policyCount[policy.Namespace]++
		}

		findings = append(findings, assessmentv1alpha1.Finding{
			ID:          "networking-policies-found",
			Validator:   validatorName,
			Category:    validatorCategory,
			Status:      assessmentv1alpha1.FindingStatusPass,
			Title:       "NetworkPolicies Configured",
			Description: fmt.Sprintf("Found %d NetworkPolicy(ies) across %d namespace(s).", len(policies.Items), len(policyCount)),
		})
	}

	return findings
}

// checkIngressConfig validates ingress configuration.
func (v *NetworkingValidator) checkIngressConfig(ctx context.Context, c client.Client) []assessmentv1alpha1.Finding {
	var findings []assessmentv1alpha1.Finding

	ingress := &configv1.Ingress{}
	if err := c.Get(ctx, client.ObjectKey{Name: "cluster"}, ingress); err != nil {
		return []assessmentv1alpha1.Finding{{
			ID:          "networking-ingress-error",
			Validator:   validatorName,
			Category:    validatorCategory,
			Status:      assessmentv1alpha1.FindingStatusInfo,
			Title:       "Unable to Check Ingress Configuration",
			Description: fmt.Sprintf("Failed to get Ingress configuration: %v", err),
		}}
	}

	// Report ingress domain
	findings = append(findings, assessmentv1alpha1.Finding{
		ID:          "networking-ingress-domain",
		Validator:   validatorName,
		Category:    validatorCategory,
		Status:      assessmentv1alpha1.FindingStatusInfo,
		Title:       "Ingress Domain",
		Description: fmt.Sprintf("Cluster ingress domain: %s", ingress.Spec.Domain),
	})

	return findings
}

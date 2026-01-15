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

package compliance

import (
	"context"
	"fmt"
	"strings"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"sigs.k8s.io/controller-runtime/pkg/client"

	assessmentv1alpha1 "github.com/openshift-assessment/cluster-assessment-operator/api/v1alpha1"
	"github.com/openshift-assessment/cluster-assessment-operator/pkg/profiles"
	"github.com/openshift-assessment/cluster-assessment-operator/pkg/validator"
)

const (
	validatorName        = "compliance"
	validatorDescription = "Validates security compliance settings including Pod Security Admission, OAuth, and authentication"
	validatorCategory    = "Security"
)

func init() {
	_ = validator.Register(&ComplianceValidator{})
}

// ComplianceValidator checks security compliance configuration.
type ComplianceValidator struct{}

// Name returns the validator name.
func (v *ComplianceValidator) Name() string {
	return validatorName
}

// Description returns the validator description.
func (v *ComplianceValidator) Description() string {
	return validatorDescription
}

// Category returns the finding category.
func (v *ComplianceValidator) Category() string {
	return validatorCategory
}

// Validate performs compliance checks.
func (v *ComplianceValidator) Validate(ctx context.Context, c client.Client, profile profiles.Profile) ([]assessmentv1alpha1.Finding, error) {
	var findings []assessmentv1alpha1.Finding

	// Check 1: Pod Security Admission labels
	findings = append(findings, v.checkPodSecurityAdmission(ctx, c, profile)...)

	// Check 2: OAuth configuration
	findings = append(findings, v.checkOAuthConfiguration(ctx, c)...)

	// Check 3: Kubeadmin user
	findings = append(findings, v.checkKubeadminUser(ctx, c, profile)...)

	return findings, nil
}

// checkPodSecurityAdmission checks for Pod Security Admission labels on namespaces.
func (v *ComplianceValidator) checkPodSecurityAdmission(ctx context.Context, c client.Client, profile profiles.Profile) []assessmentv1alpha1.Finding {
	var findings []assessmentv1alpha1.Finding

	namespaces := &corev1.NamespaceList{}
	if err := c.List(ctx, namespaces); err != nil {
		return []assessmentv1alpha1.Finding{{
			ID:          "compliance-psa-error",
			Validator:   validatorName,
			Category:    validatorCategory,
			Status:      assessmentv1alpha1.FindingStatusFail,
			Title:       "Unable to Check Namespaces",
			Description: fmt.Sprintf("Failed to list namespaces: %v", err),
		}}
	}

	var namespacesWithEnforce []string
	var userNamespacesWithoutPSA []string

	for _, ns := range namespaces.Items {
		// Skip system namespaces
		if strings.HasPrefix(ns.Name, "openshift-") || strings.HasPrefix(ns.Name, "kube-") || ns.Name == "default" {
			continue
		}

		// Check for PSA enforce label (primary security control)
		hasEnforce := false
		if val, ok := ns.Labels["pod-security.kubernetes.io/enforce"]; ok {
			hasEnforce = true
			if val == "restricted" || val == "baseline" {
				namespacesWithEnforce = append(namespacesWithEnforce, ns.Name)
			}
		}

		// Also check for audit/warn labels as fallback indicators
		hasAudit := ns.Labels["pod-security.kubernetes.io/audit"] != ""
		hasWarn := ns.Labels["pod-security.kubernetes.io/warn"] != ""

		if !hasEnforce && !hasAudit && !hasWarn {
			userNamespacesWithoutPSA = append(userNamespacesWithoutPSA, ns.Name)
		}
	}

	// Report PSA adoption
	totalUserNs := len(namespacesWithEnforce) + len(userNamespacesWithoutPSA)
	if len(namespacesWithEnforce) > 0 {
		findings = append(findings, assessmentv1alpha1.Finding{
			ID:          "compliance-psa-enforce",
			Validator:   validatorName,
			Category:    validatorCategory,
			Status:      assessmentv1alpha1.FindingStatusPass,
			Title:       "Pod Security Admission Enforced",
			Description: fmt.Sprintf("%d of %d user namespace(s) have PSA enforce labels.", len(namespacesWithEnforce), totalUserNs),
		})
	}

	if len(userNamespacesWithoutPSA) > 0 {
		status := assessmentv1alpha1.FindingStatusInfo
		if profile.Name == profiles.ProfileProduction {
			status = assessmentv1alpha1.FindingStatusWarn
		}

		sample := userNamespacesWithoutPSA
		if len(sample) > 5 {
			sample = sample[:5]
		}

		findings = append(findings, assessmentv1alpha1.Finding{
			ID:             "compliance-psa-missing",
			Validator:      validatorName,
			Category:       validatorCategory,
			Status:         status,
			Title:          "Namespaces Without Pod Security Admission",
			Description:    fmt.Sprintf("%d user namespace(s) have no PSA labels: %s...", len(userNamespacesWithoutPSA), strings.Join(sample, ", ")),
			Impact:         "Namespaces without PSA labels use the cluster-wide default policy.",
			Recommendation: "Consider adding pod-security.kubernetes.io/enforce labels to user namespaces.",
			References: []string{
				"https://kubernetes.io/docs/concepts/security/pod-security-admission/",
			},
		})
	}

	return findings
}

// checkOAuthConfiguration checks OAuth provider configuration.
func (v *ComplianceValidator) checkOAuthConfiguration(ctx context.Context, c client.Client) []assessmentv1alpha1.Finding {
	var findings []assessmentv1alpha1.Finding

	// Get OAuth cluster configuration
	oauth := &unstructured.Unstructured{}
	oauth.SetGroupVersionKind(schema.GroupVersionKind{
		Group:   "config.openshift.io",
		Version: "v1",
		Kind:    "OAuth",
	})

	if err := c.Get(ctx, client.ObjectKey{Name: "cluster"}, oauth); err != nil {
		return []assessmentv1alpha1.Finding{{
			ID:          "compliance-oauth-error",
			Validator:   validatorName,
			Category:    validatorCategory,
			Status:      assessmentv1alpha1.FindingStatusInfo,
			Title:       "Unable to Check OAuth Configuration",
			Description: fmt.Sprintf("Failed to get OAuth config: %v", err),
		}}
	}

	// Check identity providers
	identityProviders, found, _ := unstructured.NestedSlice(oauth.Object, "spec", "identityProviders")
	if !found || len(identityProviders) == 0 {
		findings = append(findings, assessmentv1alpha1.Finding{
			ID:             "compliance-oauth-no-idp",
			Validator:      validatorName,
			Category:       validatorCategory,
			Status:         assessmentv1alpha1.FindingStatusWarn,
			Title:          "No Identity Providers Configured",
			Description:    "No OAuth identity providers are configured.",
			Impact:         "Only kubeadmin or service accounts can authenticate to the cluster.",
			Recommendation: "Configure at least one identity provider (LDAP, OIDC, HTPasswd, etc.).",
			References: []string{
				"https://docs.openshift.com/container-platform/latest/authentication/understanding-identity-provider.html",
			},
		})
	} else {
		var idpNames []string
		var hasHTPasswd bool
		for _, idp := range identityProviders {
			idpMap, ok := idp.(map[string]interface{})
			if !ok {
				continue
			}
			if name, ok := idpMap["name"].(string); ok {
				idpNames = append(idpNames, name)
			}
			if idpMap["htpasswd"] != nil {
				hasHTPasswd = true
			}
		}

		findings = append(findings, assessmentv1alpha1.Finding{
			ID:          "compliance-oauth-idp-configured",
			Validator:   validatorName,
			Category:    validatorCategory,
			Status:      assessmentv1alpha1.FindingStatusPass,
			Title:       "Identity Providers Configured",
			Description: fmt.Sprintf("%d identity provider(s) configured: %s", len(identityProviders), strings.Join(idpNames, ", ")),
		})

		// Warn about HTPasswd in production
		if hasHTPasswd {
			findings = append(findings, assessmentv1alpha1.Finding{
				ID:             "compliance-oauth-htpasswd",
				Validator:      validatorName,
				Category:       validatorCategory,
				Status:         assessmentv1alpha1.FindingStatusInfo,
				Title:          "HTPasswd Identity Provider in Use",
				Description:    "An HTPasswd identity provider is configured.",
				Impact:         "HTPasswd requires manual user management and password resets.",
				Recommendation: "Consider using LDAP, OIDC, or other centralized identity providers for production.",
			})
		}
	}

	// Check token configuration
	tokenConfig, found, _ := unstructured.NestedMap(oauth.Object, "spec", "tokenConfig")
	if found {
		if accessTokenMaxAge, ok := tokenConfig["accessTokenMaxAgeSeconds"].(int64); ok {
			if accessTokenMaxAge > 86400 { // More than 24 hours
				findings = append(findings, assessmentv1alpha1.Finding{
					ID:             "compliance-oauth-token-age",
					Validator:      validatorName,
					Category:       validatorCategory,
					Status:         assessmentv1alpha1.FindingStatusInfo,
					Title:          "Long Access Token Lifetime",
					Description:    fmt.Sprintf("Access token max age is set to %d seconds (%d hours).", accessTokenMaxAge, accessTokenMaxAge/3600),
					Impact:         "Longer token lifetimes increase the window of opportunity for token theft.",
					Recommendation: "Consider reducing token lifetime for sensitive environments.",
				})
			}
		}
	}

	return findings
}

// checkKubeadminUser checks if the kubeadmin user still exists.
func (v *ComplianceValidator) checkKubeadminUser(ctx context.Context, c client.Client, profile profiles.Profile) []assessmentv1alpha1.Finding {
	var findings []assessmentv1alpha1.Finding

	// Check for kubeadmin secret
	kubeadminSecret := &corev1.Secret{}
	if err := c.Get(ctx, client.ObjectKey{Namespace: "kube-system", Name: "kubeadmin"}, kubeadminSecret); err == nil {
		// Secret exists - kubeadmin is still present
		status := assessmentv1alpha1.FindingStatusInfo
		if profile.Name == profiles.ProfileProduction {
			status = assessmentv1alpha1.FindingStatusWarn
		}

		findings = append(findings, assessmentv1alpha1.Finding{
			ID:             "compliance-kubeadmin-exists",
			Validator:      validatorName,
			Category:       validatorCategory,
			Status:         status,
			Title:          "Kubeadmin User Still Exists",
			Description:    "The kubeadmin user has not been removed.",
			Impact:         "Kubeadmin provides cluster-admin access with a static password.",
			Recommendation: "After configuring identity providers, remove the kubeadmin user: oc delete secret kubeadmin -n kube-system",
			References: []string{
				"https://docs.openshift.com/container-platform/latest/authentication/remove-kubeadmin.html",
			},
		})
	} else {
		findings = append(findings, assessmentv1alpha1.Finding{
			ID:          "compliance-kubeadmin-removed",
			Validator:   validatorName,
			Category:    validatorCategory,
			Status:      assessmentv1alpha1.FindingStatusPass,
			Title:       "Kubeadmin User Removed",
			Description: "The kubeadmin user has been properly removed.",
		})
	}

	return findings
}

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

package security

import (
	"context"
	"fmt"
	"strings"

	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	assessmentv1alpha1 "github.com/openshift-assessment/cluster-assessment-operator/api/v1alpha1"
	"github.com/openshift-assessment/cluster-assessment-operator/pkg/profiles"
	"github.com/openshift-assessment/cluster-assessment-operator/pkg/validator"
)

const (
	validatorName        = "security"
	validatorDescription = "Validates security configuration including SCCs, RBAC, and privileged workloads"
	validatorCategory    = "Security"
)

// Namespaces that are expected to have cluster-admin or privileged access
var systemNamespaces = map[string]bool{
	"openshift-apiserver":                    true,
	"openshift-controller-manager":           true,
	"openshift-etcd":                         true,
	"openshift-kube-apiserver":               true,
	"openshift-kube-controller-manager":      true,
	"openshift-kube-scheduler":               true,
	"openshift-machine-api":                  true,
	"openshift-machine-config-operator":      true,
	"openshift-monitoring":                   true,
	"openshift-network-operator":             true,
	"openshift-sdn":                          true,
	"openshift-ovn-kubernetes":               true,
	"openshift-operator-lifecycle-manager":   true,
	"openshift-operators":                    true,
	"openshift-cluster-version":              true,
	"openshift-ingress":                      true,
	"openshift-dns":                          true,
	"openshift-image-registry":               true,
	"openshift-authentication":               true,
	"openshift-oauth-apiserver":              true,
	"kube-system":                            true,
	"openshift-cluster-node-tuning-operator": true,
	"openshift-cluster-storage-operator":     true,
	"openshift-multus":                       true,
}

func init() {
	_ = validator.Register(&SecurityValidator{})
}

// SecurityValidator checks security configuration.
type SecurityValidator struct{}

// Name returns the validator name.
func (v *SecurityValidator) Name() string {
	return validatorName
}

// Description returns the validator description.
func (v *SecurityValidator) Description() string {
	return validatorDescription
}

// Category returns the finding category.
func (v *SecurityValidator) Category() string {
	return validatorCategory
}

// Validate performs security checks.
func (v *SecurityValidator) Validate(ctx context.Context, c client.Client, profile profiles.Profile) ([]assessmentv1alpha1.Finding, error) {
	var findings []assessmentv1alpha1.Finding

	// Check 1: Cluster-admin bindings
	findings = append(findings, v.checkClusterAdminBindings(ctx, c, profile)...)

	// Check 2: Privileged pods
	findings = append(findings, v.checkPrivilegedPods(ctx, c, profile)...)

	// Check 3: Service account token automation
	findings = append(findings, v.checkServiceAccountTokenAutomation(ctx, c)...)

	// Check 4: Risky RBAC patterns
	findings = append(findings, v.checkRiskyRBACPatterns(ctx, c)...)

	return findings, nil
}

// checkClusterAdminBindings checks for excessive cluster-admin usage.
func (v *SecurityValidator) checkClusterAdminBindings(ctx context.Context, c client.Client, profile profiles.Profile) []assessmentv1alpha1.Finding {
	var findings []assessmentv1alpha1.Finding

	// Get ClusterRoleBindings
	crbs := &rbacv1.ClusterRoleBindingList{}
	if err := c.List(ctx, crbs); err != nil {
		return []assessmentv1alpha1.Finding{{
			ID:          "security-crb-error",
			Validator:   validatorName,
			Category:    validatorCategory,
			Status:      assessmentv1alpha1.FindingStatusFail,
			Title:       "Unable to Check ClusterRoleBindings",
			Description: fmt.Sprintf("Failed to list ClusterRoleBindings: %v", err),
		}}
	}

	var clusterAdminBindings []string
	var nonSystemClusterAdminBindings []string

	for _, crb := range crbs.Items {
		if crb.RoleRef.Name == "cluster-admin" {
			clusterAdminBindings = append(clusterAdminBindings, crb.Name)

			// Check if it's binding to non-system subjects
			for _, subject := range crb.Subjects {
				switch subject.Kind {
				case "ServiceAccount":
					if !systemNamespaces[subject.Namespace] {
						nonSystemClusterAdminBindings = append(nonSystemClusterAdminBindings,
							fmt.Sprintf("%s (SA: %s/%s)", crb.Name, subject.Namespace, subject.Name))
					}
				case "User", "Group":
					// Users and groups with cluster-admin
					if !strings.HasPrefix(subject.Name, "system:") {
						nonSystemClusterAdminBindings = append(nonSystemClusterAdminBindings,
							fmt.Sprintf("%s (%s: %s)", crb.Name, subject.Kind, subject.Name))
					}
				}
			}
		}
	}

	// Report total cluster-admin bindings
	findings = append(findings, assessmentv1alpha1.Finding{
		ID:          "security-cluster-admin-total",
		Validator:   validatorName,
		Category:    validatorCategory,
		Status:      assessmentv1alpha1.FindingStatusInfo,
		Title:       "Cluster-Admin Bindings",
		Description: fmt.Sprintf("Found %d ClusterRoleBindings referencing cluster-admin.", len(clusterAdminBindings)),
	})

	// Check non-system cluster-admin bindings
	if len(nonSystemClusterAdminBindings) > profile.Thresholds.MaxClusterAdminBindings {
		findings = append(findings, assessmentv1alpha1.Finding{
			ID:             "security-cluster-admin-excessive",
			Validator:      validatorName,
			Category:       validatorCategory,
			Status:         assessmentv1alpha1.FindingStatusWarn,
			Title:          "Excessive Non-System Cluster-Admin Bindings",
			Description:    fmt.Sprintf("Found %d non-system cluster-admin bindings (threshold: %d): %s", len(nonSystemClusterAdminBindings), profile.Thresholds.MaxClusterAdminBindings, strings.Join(nonSystemClusterAdminBindings, ", ")),
			Impact:         "Excessive cluster-admin permissions increase the attack surface and risk of privilege escalation.",
			Recommendation: "Review cluster-admin bindings and apply least privilege principle. Consider using more specific ClusterRoles.",
			References: []string{
				"https://docs.openshift.com/container-platform/latest/authentication/using-rbac.html",
			},
		})
	} else if len(nonSystemClusterAdminBindings) > 0 {
		findings = append(findings, assessmentv1alpha1.Finding{
			ID:          "security-cluster-admin-found",
			Validator:   validatorName,
			Category:    validatorCategory,
			Status:      assessmentv1alpha1.FindingStatusInfo,
			Title:       "Non-System Cluster-Admin Bindings",
			Description: fmt.Sprintf("Found %d non-system cluster-admin bindings: %s", len(nonSystemClusterAdminBindings), strings.Join(nonSystemClusterAdminBindings, ", ")),
		})
	} else {
		findings = append(findings, assessmentv1alpha1.Finding{
			ID:          "security-cluster-admin-minimal",
			Validator:   validatorName,
			Category:    validatorCategory,
			Status:      assessmentv1alpha1.FindingStatusPass,
			Title:       "Minimal Cluster-Admin Usage",
			Description: "No non-system cluster-admin bindings found.",
		})
	}

	return findings
}

// checkPrivilegedPods checks for privileged containers.
func (v *SecurityValidator) checkPrivilegedPods(ctx context.Context, c client.Client, profile profiles.Profile) []assessmentv1alpha1.Finding {
	var findings []assessmentv1alpha1.Finding

	pods := &corev1.PodList{}
	if err := c.List(ctx, pods); err != nil {
		return []assessmentv1alpha1.Finding{{
			ID:          "security-pods-error",
			Validator:   validatorName,
			Category:    validatorCategory,
			Status:      assessmentv1alpha1.FindingStatusFail,
			Title:       "Unable to Check Pods",
			Description: fmt.Sprintf("Failed to list pods: %v", err),
		}}
	}

	var privilegedPods []string
	var hostNetworkPods []string
	var hostPIDPods []string

	for _, pod := range pods.Items {
		// Skip system namespaces
		if systemNamespaces[pod.Namespace] || strings.HasPrefix(pod.Namespace, "openshift-") {
			continue
		}

		// Check for privileged containers
		isPrivileged := false
		for _, container := range pod.Spec.Containers {
			if container.SecurityContext != nil && container.SecurityContext.Privileged != nil && *container.SecurityContext.Privileged {
				isPrivileged = true
				break
			}
		}
		if isPrivileged {
			privilegedPods = append(privilegedPods, fmt.Sprintf("%s/%s", pod.Namespace, pod.Name))
		}

		// Check for host network
		if pod.Spec.HostNetwork {
			hostNetworkPods = append(hostNetworkPods, fmt.Sprintf("%s/%s", pod.Namespace, pod.Name))
		}

		// Check for host PID
		if pod.Spec.HostPID {
			hostPIDPods = append(hostPIDPods, fmt.Sprintf("%s/%s", pod.Namespace, pod.Name))
		}
	}

	// Report privileged pods
	if len(privilegedPods) > 0 {
		status := assessmentv1alpha1.FindingStatusInfo
		if !profile.Thresholds.AllowPrivilegedContainers {
			status = assessmentv1alpha1.FindingStatusWarn
		}

		sample := privilegedPods
		if len(sample) > 5 {
			sample = sample[:5]
		}

		findings = append(findings, assessmentv1alpha1.Finding{
			ID:             "security-privileged-pods",
			Validator:      validatorName,
			Category:       validatorCategory,
			Status:         status,
			Title:          "Privileged Containers in User Namespaces",
			Description:    fmt.Sprintf("Found %d pod(s) with privileged containers in user namespaces: %s...", len(privilegedPods), strings.Join(sample, ", ")),
			Impact:         "Privileged containers have elevated access to the host and bypass many security controls.",
			Recommendation: "Review if privileged access is necessary. Consider using specific capabilities instead of full privileged mode.",
		})
	} else {
		findings = append(findings, assessmentv1alpha1.Finding{
			ID:          "security-no-privileged-pods",
			Validator:   validatorName,
			Category:    validatorCategory,
			Status:      assessmentv1alpha1.FindingStatusPass,
			Title:       "No Privileged Containers in User Namespaces",
			Description: "No privileged containers found in user namespaces.",
		})
	}

	// Report host network pods
	if len(hostNetworkPods) > 0 {
		sample := hostNetworkPods
		if len(sample) > 5 {
			sample = sample[:5]
		}
		findings = append(findings, assessmentv1alpha1.Finding{
			ID:             "security-host-network",
			Validator:      validatorName,
			Category:       validatorCategory,
			Status:         assessmentv1alpha1.FindingStatusWarn,
			Title:          "Pods Using Host Network",
			Description:    fmt.Sprintf("Found %d pod(s) using host network in user namespaces: %s...", len(hostNetworkPods), strings.Join(sample, ", ")),
			Impact:         "Pods with host network access can see all network traffic on the node.",
			Recommendation: "Review if host network access is necessary. Use CNI networking when possible.",
		})
	}

	// Report host PID pods
	if len(hostPIDPods) > 0 {
		sample := hostPIDPods
		if len(sample) > 5 {
			sample = sample[:5]
		}
		findings = append(findings, assessmentv1alpha1.Finding{
			ID:             "security-host-pid",
			Validator:      validatorName,
			Category:       validatorCategory,
			Status:         assessmentv1alpha1.FindingStatusWarn,
			Title:          "Pods Using Host PID",
			Description:    fmt.Sprintf("Found %d pod(s) using host PID namespace in user namespaces: %s...", len(hostPIDPods), strings.Join(sample, ", ")),
			Impact:         "Pods with host PID access can see and potentially interact with all processes on the node.",
			Recommendation: "Review if host PID namespace access is necessary.",
		})
	}

	return findings
}

// checkServiceAccountTokenAutomation checks for service account token mount settings.
func (v *SecurityValidator) checkServiceAccountTokenAutomation(ctx context.Context, c client.Client) []assessmentv1alpha1.Finding {
	var findings []assessmentv1alpha1.Finding

	// Check if default service accounts have automount disabled
	namespaces := &corev1.NamespaceList{}
	if err := c.List(ctx, namespaces); err != nil {
		return findings
	}

	var automountEnabledNamespaces []string

	for _, ns := range namespaces.Items {
		// Skip system namespaces
		if systemNamespaces[ns.Name] || strings.HasPrefix(ns.Name, "openshift-") || strings.HasPrefix(ns.Name, "kube-") {
			continue
		}

		// Get default service account
		sa := &corev1.ServiceAccount{}
		if err := c.Get(ctx, client.ObjectKey{Namespace: ns.Name, Name: "default"}, sa); err != nil {
			continue
		}

		// Check if automount is not explicitly disabled
		if sa.AutomountServiceAccountToken == nil || *sa.AutomountServiceAccountToken {
			automountEnabledNamespaces = append(automountEnabledNamespaces, ns.Name)
		}
	}

	if len(automountEnabledNamespaces) > 0 {
		findings = append(findings, assessmentv1alpha1.Finding{
			ID:             "security-sa-automount",
			Validator:      validatorName,
			Category:       validatorCategory,
			Status:         assessmentv1alpha1.FindingStatusInfo,
			Title:          "Service Account Token Automount Enabled",
			Description:    fmt.Sprintf("%d user namespace(s) have default service accounts with token automount enabled.", len(automountEnabledNamespaces)),
			Impact:         "Pods automatically receive service account tokens which may not always be necessary.",
			Recommendation: "Consider disabling automountServiceAccountToken on default service accounts where not needed.",
			References: []string{
				"https://kubernetes.io/docs/tasks/configure-pod-container/configure-service-account/",
			},
		})
	}

	return findings
}

// checkRiskyRBACPatterns checks for risky RBAC configurations.
func (v *SecurityValidator) checkRiskyRBACPatterns(ctx context.Context, c client.Client) []assessmentv1alpha1.Finding {
	var findings []assessmentv1alpha1.Finding

	// Get ClusterRoles
	clusterRoles := &rbacv1.ClusterRoleList{}
	if err := c.List(ctx, clusterRoles); err != nil {
		return findings
	}

	var wildcardRoles []string
	var secretsAccessRoles []string

	for _, cr := range clusterRoles.Items {
		// Skip system roles
		if strings.HasPrefix(cr.Name, "system:") || strings.HasPrefix(cr.Name, "openshift") {
			continue
		}

		for _, rule := range cr.Rules {
			// Check for wildcard permissions
			for _, verb := range rule.Verbs {
				if verb == "*" {
					for _, resource := range rule.Resources {
						if resource == "*" {
							wildcardRoles = append(wildcardRoles, cr.Name)
							break
						}
					}
				}
			}

			// Check for secrets access
			for _, resource := range rule.Resources {
				if resource == "secrets" || resource == "*" {
					for _, verb := range rule.Verbs {
						if verb == "get" || verb == "list" || verb == "watch" || verb == "*" {
							secretsAccessRoles = append(secretsAccessRoles, cr.Name)
							break
						}
					}
				}
			}
		}
	}

	// Remove duplicates
	wildcardRoles = unique(wildcardRoles)
	secretsAccessRoles = unique(secretsAccessRoles)

	if len(wildcardRoles) > 0 {
		findings = append(findings, assessmentv1alpha1.Finding{
			ID:             "security-rbac-wildcard",
			Validator:      validatorName,
			Category:       validatorCategory,
			Status:         assessmentv1alpha1.FindingStatusWarn,
			Title:          "ClusterRoles with Wildcard Permissions",
			Description:    fmt.Sprintf("Found %d custom ClusterRole(s) with wildcard (*) permissions: %s", len(wildcardRoles), strings.Join(wildcardRoles, ", ")),
			Impact:         "Wildcard permissions grant excessive access and violate the principle of least privilege.",
			Recommendation: "Refine ClusterRoles to specify only the necessary resources and verbs.",
		})
	}

	if len(secretsAccessRoles) > 0 {
		findings = append(findings, assessmentv1alpha1.Finding{
			ID:             "security-rbac-secrets",
			Validator:      validatorName,
			Category:       validatorCategory,
			Status:         assessmentv1alpha1.FindingStatusInfo,
			Title:          "ClusterRoles with Secrets Access",
			Description:    fmt.Sprintf("Found %d custom ClusterRole(s) with secrets access: %s", len(secretsAccessRoles), strings.Join(secretsAccessRoles, ", ")),
			Impact:         "Access to secrets allows reading sensitive data including credentials and tokens.",
			Recommendation: "Review if secrets access is necessary and limit to specific namespaces if possible.",
		})
	}

	return findings
}

// unique removes duplicates from a string slice.
func unique(slice []string) []string {
	seen := make(map[string]bool)
	result := []string{}
	for _, s := range slice {
		if !seen[s] {
			seen[s] = true
			result = append(result, s)
		}
	}
	return result
}

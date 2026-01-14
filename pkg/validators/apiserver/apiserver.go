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

package apiserver

import (
	"context"
	"fmt"

	configv1 "github.com/openshift/api/config/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	assessmentv1alpha1 "github.com/openshift-assessment/cluster-assessment-operator/api/v1alpha1"
	"github.com/openshift-assessment/cluster-assessment-operator/pkg/profiles"
	"github.com/openshift-assessment/cluster-assessment-operator/pkg/validator"
)

const (
	validatorName        = "apiserver"
	validatorDescription = "Validates API server and etcd configuration (read-only inspection)"
	validatorCategory    = "Platform"
)

func init() {
	_ = validator.Register(&APIServerValidator{})
}

// APIServerValidator checks API server and etcd configuration.
type APIServerValidator struct{}

// Name returns the validator name.
func (v *APIServerValidator) Name() string {
	return validatorName
}

// Description returns the validator description.
func (v *APIServerValidator) Description() string {
	return validatorDescription
}

// Category returns the finding category.
func (v *APIServerValidator) Category() string {
	return validatorCategory
}

// Validate performs API server and etcd checks.
func (v *APIServerValidator) Validate(ctx context.Context, c client.Client, profile profiles.Profile) ([]assessmentv1alpha1.Finding, error) {
	var findings []assessmentv1alpha1.Finding

	// Check 1: API Server configuration
	findings = append(findings, v.checkAPIServer(ctx, c)...)

	// Check 2: etcd status via ClusterOperator
	findings = append(findings, v.checkEtcd(ctx, c)...)

	// Check 3: API Server encryption
	findings = append(findings, v.checkEncryption(ctx, c)...)

	// Check 4: Audit logging configuration
	findings = append(findings, v.checkAuditPolicy(ctx, c)...)

	return findings, nil
}

// checkAPIServer validates API server ClusterOperator status.
func (v *APIServerValidator) checkAPIServer(ctx context.Context, c client.Client) []assessmentv1alpha1.Finding {
	var findings []assessmentv1alpha1.Finding

	// Check kube-apiserver ClusterOperator
	co := &configv1.ClusterOperator{}
	if err := c.Get(ctx, client.ObjectKey{Name: "kube-apiserver"}, co); err != nil {
		return []assessmentv1alpha1.Finding{{
			ID:          "apiserver-operator-error",
			Validator:   validatorName,
			Category:    validatorCategory,
			Status:      assessmentv1alpha1.FindingStatusFail,
			Title:       "Unable to Check API Server Operator",
			Description: fmt.Sprintf("Failed to get kube-apiserver ClusterOperator: %v", err),
		}}
	}

	var isAvailable, isDegraded, isProgressing bool

	for _, condition := range co.Status.Conditions {
		switch condition.Type {
		case configv1.OperatorAvailable:
			isAvailable = condition.Status == configv1.ConditionTrue
		case configv1.OperatorDegraded:
			isDegraded = condition.Status == configv1.ConditionTrue
		case configv1.OperatorProgressing:
			isProgressing = condition.Status == configv1.ConditionTrue
		}
	}

	if isDegraded {
		findings = append(findings, assessmentv1alpha1.Finding{
			ID:             "apiserver-degraded",
			Validator:      validatorName,
			Category:       validatorCategory,
			Status:         assessmentv1alpha1.FindingStatusFail,
			Title:          "API Server Degraded",
			Description:    "The kube-apiserver ClusterOperator is in a degraded state.",
			Impact:         "A degraded API server may affect cluster operations and API availability.",
			Recommendation: "Check the kube-apiserver operator logs and events for issues.",
		})
	} else if !isAvailable {
		findings = append(findings, assessmentv1alpha1.Finding{
			ID:             "apiserver-unavailable",
			Validator:      validatorName,
			Category:       validatorCategory,
			Status:         assessmentv1alpha1.FindingStatusFail,
			Title:          "API Server Not Available",
			Description:    "The kube-apiserver ClusterOperator is not available.",
			Impact:         "API server unavailability will affect cluster operations.",
			Recommendation: "Investigate the openshift-kube-apiserver namespace for issues.",
		})
	} else if isProgressing {
		findings = append(findings, assessmentv1alpha1.Finding{
			ID:          "apiserver-progressing",
			Validator:   validatorName,
			Category:    validatorCategory,
			Status:      assessmentv1alpha1.FindingStatusInfo,
			Title:       "API Server Updating",
			Description: "The kube-apiserver ClusterOperator is currently updating.",
		})
	} else {
		findings = append(findings, assessmentv1alpha1.Finding{
			ID:          "apiserver-healthy",
			Validator:   validatorName,
			Category:    validatorCategory,
			Status:      assessmentv1alpha1.FindingStatusPass,
			Title:       "API Server Healthy",
			Description: "The kube-apiserver ClusterOperator is available and not degraded.",
		})
	}

	return findings
}

// checkEtcd validates etcd ClusterOperator status.
func (v *APIServerValidator) checkEtcd(ctx context.Context, c client.Client) []assessmentv1alpha1.Finding {
	var findings []assessmentv1alpha1.Finding

	// Check etcd ClusterOperator
	co := &configv1.ClusterOperator{}
	if err := c.Get(ctx, client.ObjectKey{Name: "etcd"}, co); err != nil {
		return []assessmentv1alpha1.Finding{{
			ID:          "etcd-operator-error",
			Validator:   validatorName,
			Category:    validatorCategory,
			Status:      assessmentv1alpha1.FindingStatusFail,
			Title:       "Unable to Check etcd Operator",
			Description: fmt.Sprintf("Failed to get etcd ClusterOperator: %v", err),
		}}
	}

	var isAvailable, isDegraded, isProgressing bool

	for _, condition := range co.Status.Conditions {
		switch condition.Type {
		case configv1.OperatorAvailable:
			isAvailable = condition.Status == configv1.ConditionTrue
		case configv1.OperatorDegraded:
			isDegraded = condition.Status == configv1.ConditionTrue
		case configv1.OperatorProgressing:
			isProgressing = condition.Status == configv1.ConditionTrue
		}
	}

	if isDegraded {
		findings = append(findings, assessmentv1alpha1.Finding{
			ID:             "etcd-degraded",
			Validator:      validatorName,
			Category:       validatorCategory,
			Status:         assessmentv1alpha1.FindingStatusFail,
			Title:          "etcd Degraded",
			Description:    "The etcd ClusterOperator is in a degraded state.",
			Impact:         "A degraded etcd affects cluster data storage and may cause data inconsistencies.",
			Recommendation: "Check etcd pod logs in openshift-etcd namespace and verify etcd member health.",
		})
	} else if !isAvailable {
		findings = append(findings, assessmentv1alpha1.Finding{
			ID:             "etcd-unavailable",
			Validator:      validatorName,
			Category:       validatorCategory,
			Status:         assessmentv1alpha1.FindingStatusFail,
			Title:          "etcd Not Available",
			Description:    "The etcd ClusterOperator is not available.",
			Impact:         "etcd unavailability will cause cluster-wide failures.",
			Recommendation: "Immediately investigate etcd pods in openshift-etcd namespace.",
		})
	} else if isProgressing {
		findings = append(findings, assessmentv1alpha1.Finding{
			ID:          "etcd-progressing",
			Validator:   validatorName,
			Category:    validatorCategory,
			Status:      assessmentv1alpha1.FindingStatusInfo,
			Title:       "etcd Updating",
			Description: "The etcd ClusterOperator is currently updating.",
		})
	} else {
		findings = append(findings, assessmentv1alpha1.Finding{
			ID:          "etcd-healthy",
			Validator:   validatorName,
			Category:    validatorCategory,
			Status:      assessmentv1alpha1.FindingStatusPass,
			Title:       "etcd Healthy",
			Description: "The etcd ClusterOperator is available and not degraded.",
		})
	}

	return findings
}

// checkEncryption checks API server encryption configuration.
func (v *APIServerValidator) checkEncryption(ctx context.Context, c client.Client) []assessmentv1alpha1.Finding {
	var findings []assessmentv1alpha1.Finding

	// Check APIServer config for encryption
	apiserver := &configv1.APIServer{}
	if err := c.Get(ctx, client.ObjectKey{Name: "cluster"}, apiserver); err != nil {
		return []assessmentv1alpha1.Finding{{
			ID:          "apiserver-encryption-error",
			Validator:   validatorName,
			Category:    validatorCategory,
			Status:      assessmentv1alpha1.FindingStatusInfo,
			Title:       "Unable to Check Encryption Configuration",
			Description: fmt.Sprintf("Failed to get APIServer configuration: %v", err),
		}}
	}

	// Check encryption type
	encryptionType := apiserver.Spec.Encryption.Type
	if encryptionType == "" || encryptionType == "identity" {
		findings = append(findings, assessmentv1alpha1.Finding{
			ID:             "apiserver-no-encryption",
			Validator:      validatorName,
			Category:       validatorCategory,
			Status:         assessmentv1alpha1.FindingStatusWarn,
			Title:          "etcd Encryption Not Enabled",
			Description:    "etcd encryption at rest is not enabled or using identity (no encryption).",
			Impact:         "Sensitive data in etcd (secrets, configmaps) is not encrypted at rest.",
			Recommendation: "Consider enabling etcd encryption with 'aescbc' or 'aesgcm' for sensitive workloads.",
			References: []string{
				"https://docs.openshift.com/container-platform/latest/security/encrypting-etcd.html",
			},
		})
	} else {
		findings = append(findings, assessmentv1alpha1.Finding{
			ID:          "apiserver-encryption-enabled",
			Validator:   validatorName,
			Category:    validatorCategory,
			Status:      assessmentv1alpha1.FindingStatusPass,
			Title:       "etcd Encryption Enabled",
			Description: fmt.Sprintf("etcd encryption at rest is enabled with type: %s", encryptionType),
		})
	}

	return findings
}

// checkAuditPolicy checks audit logging configuration.
func (v *APIServerValidator) checkAuditPolicy(ctx context.Context, c client.Client) []assessmentv1alpha1.Finding {
	var findings []assessmentv1alpha1.Finding

	// Check APIServer config for audit
	apiserver := &configv1.APIServer{}
	if err := c.Get(ctx, client.ObjectKey{Name: "cluster"}, apiserver); err != nil {
		return findings
	}

	// Check audit profile
	auditProfile := apiserver.Spec.Audit.Profile
	if auditProfile == "" {
		auditProfile = "Default"
	}

	switch auditProfile {
	case "None":
		findings = append(findings, assessmentv1alpha1.Finding{
			ID:             "apiserver-audit-disabled",
			Validator:      validatorName,
			Category:       validatorCategory,
			Status:         assessmentv1alpha1.FindingStatusWarn,
			Title:          "Audit Logging Disabled",
			Description:    "API server audit logging is disabled.",
			Impact:         "Without audit logs, security events and API calls cannot be reviewed for compliance or incident investigation.",
			Recommendation: "Enable audit logging with at least 'Default' profile for security visibility.",
			References: []string{
				"https://docs.openshift.com/container-platform/latest/security/audit-log-policy-config.html",
			},
		})
	case "Default", "WriteRequestBodies", "AllRequestBodies":
		findings = append(findings, assessmentv1alpha1.Finding{
			ID:          "apiserver-audit-enabled",
			Validator:   validatorName,
			Category:    validatorCategory,
			Status:      assessmentv1alpha1.FindingStatusPass,
			Title:       "Audit Logging Enabled",
			Description: fmt.Sprintf("API server audit logging is enabled with profile: %s", auditProfile),
		})
	default:
		findings = append(findings, assessmentv1alpha1.Finding{
			ID:          "apiserver-audit-custom",
			Validator:   validatorName,
			Category:    validatorCategory,
			Status:      assessmentv1alpha1.FindingStatusInfo,
			Title:       "Custom Audit Profile",
			Description: fmt.Sprintf("API server is using custom audit profile: %s", auditProfile),
		})
	}

	return findings
}

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

package monitoring

import (
	"context"
	"fmt"

	configv1 "github.com/openshift/api/config/v1"
	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	assessmentv1alpha1 "github.com/openshift-assessment/cluster-assessment-operator/api/v1alpha1"
	"github.com/openshift-assessment/cluster-assessment-operator/pkg/profiles"
	"github.com/openshift-assessment/cluster-assessment-operator/pkg/validator"
)

const (
	validatorName        = "monitoring"
	validatorDescription = "Validates monitoring and logging stack configuration"
	validatorCategory    = "Observability"
)

func init() {
	_ = validator.Register(&MonitoringValidator{})
}

// MonitoringValidator checks monitoring configuration.
type MonitoringValidator struct{}

// Name returns the validator name.
func (v *MonitoringValidator) Name() string {
	return validatorName
}

// Description returns the validator description.
func (v *MonitoringValidator) Description() string {
	return validatorDescription
}

// Category returns the finding category.
func (v *MonitoringValidator) Category() string {
	return validatorCategory
}

// Validate performs monitoring checks.
func (v *MonitoringValidator) Validate(ctx context.Context, c client.Client, profile profiles.Profile) ([]assessmentv1alpha1.Finding, error) {
	var findings []assessmentv1alpha1.Finding

	// Check 1: Cluster monitoring config
	findings = append(findings, v.checkClusterMonitoringConfig(ctx, c)...)

	// Check 2: User workload monitoring
	findings = append(findings, v.checkUserWorkloadMonitoring(ctx, c)...)

	// Check 3: ClusterOperator status
	findings = append(findings, v.checkMonitoringOperator(ctx, c)...)

	return findings, nil
}

// checkClusterMonitoringConfig checks cluster monitoring configuration.
func (v *MonitoringValidator) checkClusterMonitoringConfig(ctx context.Context, c client.Client) []assessmentv1alpha1.Finding {
	var findings []assessmentv1alpha1.Finding

	// Check for cluster-monitoring-config ConfigMap
	cm := &corev1.ConfigMap{}
	err := c.Get(ctx, client.ObjectKey{Namespace: "openshift-monitoring", Name: "cluster-monitoring-config"}, cm)

	if err != nil {
		findings = append(findings, assessmentv1alpha1.Finding{
			ID:             "monitoring-no-custom-config",
			Validator:      validatorName,
			Category:       validatorCategory,
			Status:         assessmentv1alpha1.FindingStatusInfo,
			Title:          "Default Monitoring Configuration",
			Description:    "Using default cluster monitoring configuration (no cluster-monitoring-config ConfigMap).",
			Recommendation: "Consider customizing monitoring configuration for retention, storage, and resource limits.",
			References: []string{
				"https://docs.openshift.com/container-platform/latest/monitoring/configuring-the-monitoring-stack.html",
			},
		})
	} else {
		findings = append(findings, assessmentv1alpha1.Finding{
			ID:          "monitoring-custom-config",
			Validator:   validatorName,
			Category:    validatorCategory,
			Status:      assessmentv1alpha1.FindingStatusInfo,
			Title:       "Custom Monitoring Configuration",
			Description: "Cluster monitoring has custom configuration in cluster-monitoring-config ConfigMap.",
		})

		// Check if persistent storage is configured
		if configYAML, ok := cm.Data["config.yaml"]; ok {
			if len(configYAML) > 0 {
				// Simple check for persistent storage keywords
				if containsAny(configYAML, []string{"volumeClaimTemplate", "storage", "pvc"}) {
					findings = append(findings, assessmentv1alpha1.Finding{
						ID:          "monitoring-persistent-storage",
						Validator:   validatorName,
						Category:    validatorCategory,
						Status:      assessmentv1alpha1.FindingStatusPass,
						Title:       "Monitoring Persistent Storage Configured",
						Description: "Monitoring configuration includes persistent storage settings.",
					})
				} else {
					findings = append(findings, assessmentv1alpha1.Finding{
						ID:             "monitoring-no-persistent-storage",
						Validator:      validatorName,
						Category:       validatorCategory,
						Status:         assessmentv1alpha1.FindingStatusWarn,
						Title:          "No Persistent Storage for Monitoring",
						Description:    "Monitoring configuration does not appear to include persistent storage.",
						Impact:         "Metrics data will be lost when Prometheus pods restart.",
						Recommendation: "Configure persistent storage for Prometheus to retain metrics across restarts.",
					})
				}
			}
		}
	}

	return findings
}

// checkUserWorkloadMonitoring checks user workload monitoring status.
func (v *MonitoringValidator) checkUserWorkloadMonitoring(ctx context.Context, c client.Client) []assessmentv1alpha1.Finding {
	var findings []assessmentv1alpha1.Finding

	// Check for user-workload-monitoring-config ConfigMap
	cm := &corev1.ConfigMap{}
	err := c.Get(ctx, client.ObjectKey{Namespace: "openshift-user-workload-monitoring", Name: "user-workload-monitoring-config"}, cm)

	if err != nil {
		findings = append(findings, assessmentv1alpha1.Finding{
			ID:             "monitoring-user-workload-disabled",
			Validator:      validatorName,
			Category:       validatorCategory,
			Status:         assessmentv1alpha1.FindingStatusInfo,
			Title:          "User Workload Monitoring Not Configured",
			Description:    "User workload monitoring is not configured (no user-workload-monitoring-config ConfigMap).",
			Impact:         "User workload monitoring allows monitoring of custom application metrics.",
			Recommendation: "Consider enabling user workload monitoring for application observability.",
			References: []string{
				"https://docs.openshift.com/container-platform/latest/monitoring/enabling-monitoring-for-user-defined-projects.html",
			},
		})
	} else {
		findings = append(findings, assessmentv1alpha1.Finding{
			ID:          "monitoring-user-workload-enabled",
			Validator:   validatorName,
			Category:    validatorCategory,
			Status:      assessmentv1alpha1.FindingStatusPass,
			Title:       "User Workload Monitoring Configured",
			Description: "User workload monitoring is configured.",
		})
	}

	return findings
}

// checkMonitoringOperator checks the monitoring ClusterOperator status.
func (v *MonitoringValidator) checkMonitoringOperator(ctx context.Context, c client.Client) []assessmentv1alpha1.Finding {
	var findings []assessmentv1alpha1.Finding

	co := &configv1.ClusterOperator{}
	if err := c.Get(ctx, client.ObjectKey{Name: "monitoring"}, co); err != nil {
		return []assessmentv1alpha1.Finding{{
			ID:          "monitoring-operator-error",
			Validator:   validatorName,
			Category:    validatorCategory,
			Status:      assessmentv1alpha1.FindingStatusFail,
			Title:       "Unable to Check Monitoring Operator",
			Description: fmt.Sprintf("Failed to get monitoring ClusterOperator: %v", err),
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
			ID:             "monitoring-operator-degraded",
			Validator:      validatorName,
			Category:       validatorCategory,
			Status:         assessmentv1alpha1.FindingStatusFail,
			Title:          "Monitoring Operator Degraded",
			Description:    "The monitoring ClusterOperator is in a degraded state.",
			Impact:         "Degraded monitoring may result in missing metrics or alerts.",
			Recommendation: "Investigate the monitoring operator logs and events.",
		})
	} else if !isAvailable {
		findings = append(findings, assessmentv1alpha1.Finding{
			ID:             "monitoring-operator-unavailable",
			Validator:      validatorName,
			Category:       validatorCategory,
			Status:         assessmentv1alpha1.FindingStatusFail,
			Title:          "Monitoring Operator Not Available",
			Description:    "The monitoring ClusterOperator is not available.",
			Impact:         "Monitoring capabilities may be impaired.",
			Recommendation: "Check the openshift-monitoring namespace for issues.",
		})
	} else if isProgressing {
		findings = append(findings, assessmentv1alpha1.Finding{
			ID:          "monitoring-operator-progressing",
			Validator:   validatorName,
			Category:    validatorCategory,
			Status:      assessmentv1alpha1.FindingStatusInfo,
			Title:       "Monitoring Operator Updating",
			Description: "The monitoring ClusterOperator is currently updating.",
		})
	} else {
		findings = append(findings, assessmentv1alpha1.Finding{
			ID:          "monitoring-operator-healthy",
			Validator:   validatorName,
			Category:    validatorCategory,
			Status:      assessmentv1alpha1.FindingStatusPass,
			Title:       "Monitoring Operator Healthy",
			Description: "The monitoring ClusterOperator is available and not degraded.",
		})
	}

	return findings
}

// containsAny checks if s contains any of the substrings.
func containsAny(s string, substrs []string) bool {
	for _, substr := range substrs {
		if len(substr) > 0 && len(s) >= len(substr) {
			for i := 0; i <= len(s)-len(substr); i++ {
				if s[i:i+len(substr)] == substr {
					return true
				}
			}
		}
	}
	return false
}

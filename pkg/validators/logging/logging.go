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

package logging

import (
	"context"
	"fmt"
	"strings"

	appsv1 "k8s.io/api/apps/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"sigs.k8s.io/controller-runtime/pkg/client"

	assessmentv1alpha1 "github.com/openshift-assessment/cluster-assessment-operator/api/v1alpha1"
	"github.com/openshift-assessment/cluster-assessment-operator/pkg/profiles"
	"github.com/openshift-assessment/cluster-assessment-operator/pkg/validator"
)

const (
	validatorName        = "logging"
	validatorDescription = "Validates cluster logging configuration including ClusterLogging operator, log forwarding, and collector health"
	validatorCategory    = "Observability"
)

func init() {
	_ = validator.Register(&LoggingValidator{})
}

// LoggingValidator checks cluster logging configuration.
type LoggingValidator struct{}

// Name returns the validator name.
func (v *LoggingValidator) Name() string {
	return validatorName
}

// Description returns the validator description.
func (v *LoggingValidator) Description() string {
	return validatorDescription
}

// Category returns the finding category.
func (v *LoggingValidator) Category() string {
	return validatorCategory
}

// Validate performs logging checks.
func (v *LoggingValidator) Validate(ctx context.Context, c client.Client, profile profiles.Profile) ([]assessmentv1alpha1.Finding, error) {
	var findings []assessmentv1alpha1.Finding

	// Check 1: ClusterLogging operator installation
	findings = append(findings, v.checkLoggingOperator(ctx, c)...)

	// Check 2: ClusterLogging CR
	findings = append(findings, v.checkClusterLogging(ctx, c)...)

	// Check 3: ClusterLogForwarder
	findings = append(findings, v.checkLogForwarder(ctx, c)...)

	// Check 4: Collector health
	findings = append(findings, v.checkCollectorHealth(ctx, c)...)

	return findings, nil
}

// checkLoggingOperator checks if the cluster-logging operator is installed.
func (v *LoggingValidator) checkLoggingOperator(ctx context.Context, c client.Client) []assessmentv1alpha1.Finding {
	var findings []assessmentv1alpha1.Finding

	// Check for CSV in openshift-logging namespace
	csvList := &unstructured.UnstructuredList{}
	csvList.SetGroupVersionKind(schema.GroupVersionKind{
		Group:   "operators.coreos.com",
		Version: "v1alpha1",
		Kind:    "ClusterServiceVersionList",
	})

	if err := c.List(ctx, csvList, client.InNamespace("openshift-logging")); err != nil {
		// Namespace might not exist
		findings = append(findings, assessmentv1alpha1.Finding{
			ID:             "logging-operator-missing",
			Validator:      validatorName,
			Category:       validatorCategory,
			Status:         assessmentv1alpha1.FindingStatusInfo,
			Title:          "Cluster Logging Operator Not Installed",
			Description:    "The cluster-logging operator is not installed or the openshift-logging namespace does not exist.",
			Impact:         "Cluster logging is not configured. Application and infrastructure logs are not being collected centrally.",
			Recommendation: "Consider installing the Red Hat OpenShift Logging operator for centralized log management.",
			References: []string{
				"https://docs.openshift.com/container-platform/latest/logging/cluster-logging-deploying.html",
			},
		})
		return findings
	}

	// Check if any logging-related CSV exists
	loggingInstalled := false
	var loggingVersion string
	for _, csv := range csvList.Items {
		name := csv.GetName()
		if strings.Contains(name, "cluster-logging") || strings.Contains(name, "loki-operator") {
			loggingInstalled = true
			loggingVersion = name
			break
		}
	}

	if loggingInstalled {
		findings = append(findings, assessmentv1alpha1.Finding{
			ID:          "logging-operator-installed",
			Validator:   validatorName,
			Category:    validatorCategory,
			Status:      assessmentv1alpha1.FindingStatusPass,
			Title:       "Cluster Logging Operator Installed",
			Description: fmt.Sprintf("The cluster logging operator is installed: %s", loggingVersion),
		})
	} else {
		findings = append(findings, assessmentv1alpha1.Finding{
			ID:             "logging-operator-not-found",
			Validator:      validatorName,
			Category:       validatorCategory,
			Status:         assessmentv1alpha1.FindingStatusInfo,
			Title:          "Cluster Logging Operator Not Found",
			Description:    "No cluster-logging or loki-operator CSV found in openshift-logging namespace.",
			Impact:         "Centralized logging may not be configured.",
			Recommendation: "Install the Red Hat OpenShift Logging operator if centralized logging is required.",
		})
	}

	return findings
}

// checkClusterLogging checks for ClusterLogging CR configuration.
func (v *LoggingValidator) checkClusterLogging(ctx context.Context, c client.Client) []assessmentv1alpha1.Finding {
	var findings []assessmentv1alpha1.Finding

	// Try ClusterLogging (legacy) and ClusterLogForwarder (new)
	clusterLogging := &unstructured.Unstructured{}
	clusterLogging.SetGroupVersionKind(schema.GroupVersionKind{
		Group:   "logging.openshift.io",
		Version: "v1",
		Kind:    "ClusterLogging",
	})

	if err := c.Get(ctx, client.ObjectKey{Namespace: "openshift-logging", Name: "instance"}, clusterLogging); err != nil {
		// No ClusterLogging instance
		return findings // Skip if not configured
	}

	// Check management state
	managementState, found, _ := unstructured.NestedString(clusterLogging.Object, "spec", "managementState")
	if found && managementState == "Unmanaged" {
		findings = append(findings, assessmentv1alpha1.Finding{
			ID:             "logging-unmanaged",
			Validator:      validatorName,
			Category:       validatorCategory,
			Status:         assessmentv1alpha1.FindingStatusInfo,
			Title:          "ClusterLogging Unmanaged",
			Description:    "ClusterLogging is set to Unmanaged state.",
			Impact:         "The logging operator will not reconcile logging components.",
			Recommendation: "Set managementState to Managed if automatic management is desired.",
		})
	}

	// Check collection type
	collectionType, found, _ := unstructured.NestedString(clusterLogging.Object, "spec", "collection", "type")
	if found {
		findings = append(findings, assessmentv1alpha1.Finding{
			ID:          "logging-collection-type",
			Validator:   validatorName,
			Category:    validatorCategory,
			Status:      assessmentv1alpha1.FindingStatusInfo,
			Title:       "Log Collection Type",
			Description: fmt.Sprintf("Log collection is configured with type: %s", collectionType),
		})
	}

	// Check log store type
	logStore, found, _ := unstructured.NestedMap(clusterLogging.Object, "spec", "logStore")
	if found {
		logStoreType, _, _ := unstructured.NestedString(logStore, "type")
		retentionDays, _, _ := unstructured.NestedInt64(logStore, "retentionPolicy", "application", "maxAge")

		if logStoreType != "" {
			findings = append(findings, assessmentv1alpha1.Finding{
				ID:          "logging-store-type",
				Validator:   validatorName,
				Category:    validatorCategory,
				Status:      assessmentv1alpha1.FindingStatusPass,
				Title:       "Log Store Configured",
				Description: fmt.Sprintf("Log store is configured with type: %s", logStoreType),
			})
		}

		if retentionDays > 0 {
			findings = append(findings, assessmentv1alpha1.Finding{
				ID:          "logging-retention",
				Validator:   validatorName,
				Category:    validatorCategory,
				Status:      assessmentv1alpha1.FindingStatusInfo,
				Title:       "Log Retention Policy",
				Description: fmt.Sprintf("Application log retention is set to %d days.", retentionDays),
			})
		}
	}

	return findings
}

// checkLogForwarder checks ClusterLogForwarder configuration.
func (v *LoggingValidator) checkLogForwarder(ctx context.Context, c client.Client) []assessmentv1alpha1.Finding {
	var findings []assessmentv1alpha1.Finding

	// Check for ClusterLogForwarder
	forwarder := &unstructured.Unstructured{}
	forwarder.SetGroupVersionKind(schema.GroupVersionKind{
		Group:   "logging.openshift.io",
		Version: "v1",
		Kind:    "ClusterLogForwarder",
	})

	if err := c.Get(ctx, client.ObjectKey{Namespace: "openshift-logging", Name: "instance"}, forwarder); err != nil {
		// Try collector namespace for newer versions
		if err := c.Get(ctx, client.ObjectKey{Namespace: "openshift-logging", Name: "collector"}, forwarder); err != nil {
			return findings // No forwarder configured
		}
	}

	// Check outputs
	outputs, found, _ := unstructured.NestedSlice(forwarder.Object, "spec", "outputs")
	if found && len(outputs) > 0 {
		var outputNames []string
		for _, output := range outputs {
			outputMap, ok := output.(map[string]interface{})
			if !ok {
				continue
			}
			if name, ok := outputMap["name"].(string); ok {
				outputNames = append(outputNames, name)
			}
		}

		findings = append(findings, assessmentv1alpha1.Finding{
			ID:          "logging-forwarder-outputs",
			Validator:   validatorName,
			Category:    validatorCategory,
			Status:      assessmentv1alpha1.FindingStatusPass,
			Title:       "Log Forwarding Configured",
			Description: fmt.Sprintf("Log forwarding is configured with %d output(s): %s", len(outputs), strings.Join(outputNames, ", ")),
		})
	}

	// Check pipelines
	pipelines, found, _ := unstructured.NestedSlice(forwarder.Object, "spec", "pipelines")
	if found && len(pipelines) > 0 {
		findings = append(findings, assessmentv1alpha1.Finding{
			ID:          "logging-forwarder-pipelines",
			Validator:   validatorName,
			Category:    validatorCategory,
			Status:      assessmentv1alpha1.FindingStatusInfo,
			Title:       "Log Forwarding Pipelines",
			Description: fmt.Sprintf("%d log forwarding pipeline(s) configured.", len(pipelines)),
		})
	}

	return findings
}

// checkCollectorHealth checks the health of log collector pods.
func (v *LoggingValidator) checkCollectorHealth(ctx context.Context, c client.Client) []assessmentv1alpha1.Finding {
	var findings []assessmentv1alpha1.Finding

	// Check for collector DaemonSet
	daemonsets := &appsv1.DaemonSetList{}
	if err := c.List(ctx, daemonsets, client.InNamespace("openshift-logging")); err != nil {
		return findings
	}

	for _, ds := range daemonsets.Items {
		if strings.Contains(ds.Name, "collector") || strings.Contains(ds.Name, "fluentd") || strings.Contains(ds.Name, "vector") {
			desiredPods := ds.Status.DesiredNumberScheduled
			readyPods := ds.Status.NumberReady

			if desiredPods == 0 {
				continue
			}

			if readyPods < desiredPods {
				findings = append(findings, assessmentv1alpha1.Finding{
					ID:             "logging-collector-unhealthy",
					Validator:      validatorName,
					Category:       validatorCategory,
					Status:         assessmentv1alpha1.FindingStatusWarn,
					Title:          "Log Collector Not Fully Ready",
					Description:    fmt.Sprintf("Log collector %s has %d/%d pods ready.", ds.Name, readyPods, desiredPods),
					Impact:         "Some nodes may not be collecting logs.",
					Recommendation: "Check collector pod logs and events for errors.",
				})
			} else {
				findings = append(findings, assessmentv1alpha1.Finding{
					ID:          "logging-collector-healthy",
					Validator:   validatorName,
					Category:    validatorCategory,
					Status:      assessmentv1alpha1.FindingStatusPass,
					Title:       "Log Collector Healthy",
					Description: fmt.Sprintf("Log collector %s has all %d pods ready.", ds.Name, readyPods),
				})
			}
		}
	}

	return findings
}

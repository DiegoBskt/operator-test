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

package deprecation

import (
	"context"
	"fmt"
	"strings"

	appsv1 "k8s.io/api/apps/v1"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	networkingv1 "k8s.io/api/networking/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	assessmentv1alpha1 "github.com/openshift-assessment/cluster-assessment-operator/api/v1alpha1"
	"github.com/openshift-assessment/cluster-assessment-operator/pkg/profiles"
	"github.com/openshift-assessment/cluster-assessment-operator/pkg/validator"
)

const (
	validatorName        = "deprecation"
	validatorDescription = "Detects deprecated APIs and features in use"
	validatorCategory    = "Compatibility"
)

// Define deprecated API patterns
type deprecatedAPI struct {
	Group        string
	Version      string
	Kind         string
	RemovedIn    string
	Alternative  string
	Description  string
}

// Known deprecated APIs
var deprecatedAPIs = []deprecatedAPI{
	{
		Group:       "extensions",
		Version:     "v1beta1",
		Kind:        "Ingress",
		RemovedIn:   "1.22",
		Alternative: "networking.k8s.io/v1 Ingress",
		Description: "extensions/v1beta1 Ingress is deprecated",
	},
	{
		Group:       "networking.k8s.io",
		Version:     "v1beta1",
		Kind:        "Ingress",
		RemovedIn:   "1.22",
		Alternative: "networking.k8s.io/v1 Ingress",
		Description: "networking.k8s.io/v1beta1 Ingress is deprecated",
	},
	{
		Group:       "policy",
		Version:     "v1beta1",
		Kind:        "PodSecurityPolicy",
		RemovedIn:   "1.25",
		Alternative: "Pod Security Admission",
		Description: "PodSecurityPolicy is deprecated and removed in Kubernetes 1.25",
	},
	{
		Group:       "batch",
		Version:     "v1beta1",
		Kind:        "CronJob",
		RemovedIn:   "1.25",
		Alternative: "batch/v1 CronJob",
		Description: "batch/v1beta1 CronJob is deprecated",
	},
	{
		Group:       "autoscaling",
		Version:     "v2beta1",
		Kind:        "HorizontalPodAutoscaler",
		RemovedIn:   "1.25",
		Alternative: "autoscaling/v2 HorizontalPodAutoscaler",
		Description: "autoscaling/v2beta1 HPA is deprecated",
	},
}

func init() {
	validator.Register(&DeprecationValidator{})
}

// DeprecationValidator checks for deprecated APIs and features.
type DeprecationValidator struct{}

// Name returns the validator name.
func (v *DeprecationValidator) Name() string {
	return validatorName
}

// Description returns the validator description.
func (v *DeprecationValidator) Description() string {
	return validatorDescription
}

// Category returns the finding category.
func (v *DeprecationValidator) Category() string {
	return validatorCategory
}

// Validate performs deprecation checks.
func (v *DeprecationValidator) Validate(ctx context.Context, c client.Client, profile profiles.Profile) ([]assessmentv1alpha1.Finding, error) {
	var findings []assessmentv1alpha1.Finding

	// Check 1: Deprecated workload patterns
	findings = append(findings, v.checkDeprecatedPatterns(ctx, c)...)

	// Check 2: Resources without recommended fields
	findings = append(findings, v.checkMissingRecommendedFields(ctx, c)...)

	return findings, nil
}

// checkDeprecatedPatterns checks for deprecated configuration patterns.
func (v *DeprecationValidator) checkDeprecatedPatterns(ctx context.Context, c client.Client) []assessmentv1alpha1.Finding {
	var findings []assessmentv1alpha1.Finding

	// Check for Ingresses without IngressClassName (deprecated pattern)
	ingresses := &networkingv1.IngressList{}
	if err := c.List(ctx, ingresses); err == nil {
		var noClassName []string
		for _, ing := range ingresses.Items {
			if ing.Spec.IngressClassName == nil && ing.Annotations["kubernetes.io/ingress.class"] == "" {
				noClassName = append(noClassName, fmt.Sprintf("%s/%s", ing.Namespace, ing.Name))
			}
		}
		if len(noClassName) > 0 {
			sample := noClassName
			if len(sample) > 5 {
				sample = sample[:5]
			}
			findings = append(findings, assessmentv1alpha1.Finding{
				ID:             "deprecation-ingress-no-class",
				Validator:      validatorName,
				Category:       validatorCategory,
				Status:         assessmentv1alpha1.FindingStatusWarn,
				Title:          "Ingresses Without IngressClassName",
				Description:    fmt.Sprintf("Found %d Ingress(es) without IngressClassName: %s", len(noClassName), strings.Join(sample, ", ")),
				Impact:         "Ingresses without IngressClassName may not be processed correctly in future versions.",
				Recommendation: "Set spec.ingressClassName on all Ingresses.",
				References: []string{
					"https://kubernetes.io/docs/concepts/services-networking/ingress/",
				},
			})
		}
	}

	// Check for Deployments with deprecated fields
	deployments := &appsv1.DeploymentList{}
	if err := c.List(ctx, deployments); err == nil {
		var noProbes []string
		var noResources []string

		for _, deploy := range deployments.Items {
			// Skip system namespaces
			if strings.HasPrefix(deploy.Namespace, "openshift-") || strings.HasPrefix(deploy.Namespace, "kube-") {
				continue
			}

			for _, container := range deploy.Spec.Template.Spec.Containers {
				if container.LivenessProbe == nil && container.ReadinessProbe == nil {
					noProbes = append(noProbes, fmt.Sprintf("%s/%s:%s", deploy.Namespace, deploy.Name, container.Name))
				}
				if container.Resources.Requests == nil && container.Resources.Limits == nil {
					noResources = append(noResources, fmt.Sprintf("%s/%s:%s", deploy.Namespace, deploy.Name, container.Name))
				}
			}
		}

		if len(noProbes) > 0 {
			sample := noProbes
			if len(sample) > 5 {
				sample = sample[:5]
			}
			findings = append(findings, assessmentv1alpha1.Finding{
				ID:             "deprecation-no-probes",
				Validator:      validatorName,
				Category:       validatorCategory,
				Status:         assessmentv1alpha1.FindingStatusWarn,
				Title:          "Containers Without Health Probes",
				Description:    fmt.Sprintf("Found %d container(s) without liveness or readiness probes: %s...", len(noProbes), strings.Join(sample, ", ")),
				Impact:         "Containers without probes may not be properly managed during failures or updates.",
				Recommendation: "Configure appropriate liveness and readiness probes for all containers.",
			})
		}

		if len(noResources) > 0 {
			sample := noResources
			if len(sample) > 5 {
				sample = sample[:5]
			}
			findings = append(findings, assessmentv1alpha1.Finding{
				ID:             "deprecation-no-resources",
				Validator:      validatorName,
				Category:       validatorCategory,
				Status:         assessmentv1alpha1.FindingStatusWarn,
				Title:          "Containers Without Resource Requests/Limits",
				Description:    fmt.Sprintf("Found %d container(s) without resource requests or limits: %s...", len(noResources), strings.Join(sample, ", ")),
				Impact:         "Containers without resource specifications may cause resource contention.",
				Recommendation: "Configure appropriate resource requests and limits for all containers.",
			})
		}
	}

	return findings
}

// checkMissingRecommendedFields checks for resources missing recommended fields.
func (v *DeprecationValidator) checkMissingRecommendedFields(ctx context.Context, c client.Client) []assessmentv1alpha1.Finding {
	var findings []assessmentv1alpha1.Finding

	// Check for pods without proper labels
	pods := &corev1.PodList{}
	if err := c.List(ctx, pods); err == nil {
		var noAppLabel []string
		for _, pod := range pods.Items {
			// Skip system namespaces
			if strings.HasPrefix(pod.Namespace, "openshift-") || strings.HasPrefix(pod.Namespace, "kube-") {
				continue
			}
			// Skip completed pods
			if pod.Status.Phase == corev1.PodSucceeded || pod.Status.Phase == corev1.PodFailed {
				continue
			}

			hasAppLabel := false
			for key := range pod.Labels {
				if strings.Contains(key, "app") || strings.Contains(key, "name") {
					hasAppLabel = true
					break
				}
			}
			if !hasAppLabel {
				noAppLabel = append(noAppLabel, fmt.Sprintf("%s/%s", pod.Namespace, pod.Name))
			}
		}

		if len(noAppLabel) > 10 { // Only report if significant
			sample := noAppLabel
			if len(sample) > 5 {
				sample = sample[:5]
			}
			findings = append(findings, assessmentv1alpha1.Finding{
				ID:             "deprecation-no-app-label",
				Validator:      validatorName,
				Category:       validatorCategory,
				Status:         assessmentv1alpha1.FindingStatusInfo,
				Title:          "Pods Without App Labels",
				Description:    fmt.Sprintf("Found %d pod(s) without app-related labels: %s...", len(noAppLabel), strings.Join(sample, ", ")),
				Recommendation: "Use consistent labeling (app.kubernetes.io/name, app.kubernetes.io/component) for better observability.",
			})
		}
	}

	// Check for CronJobs configurations
	cronJobs := &batchv1.CronJobList{}
	if err := c.List(ctx, cronJobs); err == nil {
		var noSuccessLimit []string
		var noFailedLimit []string

		for _, cj := range cronJobs.Items {
			if strings.HasPrefix(cj.Namespace, "openshift-") || strings.HasPrefix(cj.Namespace, "kube-") {
				continue
			}

			if cj.Spec.SuccessfulJobsHistoryLimit == nil || *cj.Spec.SuccessfulJobsHistoryLimit > 5 {
				noSuccessLimit = append(noSuccessLimit, fmt.Sprintf("%s/%s", cj.Namespace, cj.Name))
			}
			if cj.Spec.FailedJobsHistoryLimit == nil || *cj.Spec.FailedJobsHistoryLimit > 5 {
				noFailedLimit = append(noFailedLimit, fmt.Sprintf("%s/%s", cj.Namespace, cj.Name))
			}
		}

		if len(noSuccessLimit) > 0 || len(noFailedLimit) > 0 {
			totalCount := len(noSuccessLimit) + len(noFailedLimit)
			findings = append(findings, assessmentv1alpha1.Finding{
				ID:             "deprecation-cronjob-history",
				Validator:      validatorName,
				Category:       validatorCategory,
				Status:         assessmentv1alpha1.FindingStatusInfo,
				Title:          "CronJobs Without History Limits",
				Description:    fmt.Sprintf("Found %d CronJob(s) without optimal history retention limits.", totalCount),
				Impact:         "CronJobs without history limits may accumulate many completed job resources.",
				Recommendation: "Set successfulJobsHistoryLimit and failedJobsHistoryLimit to reasonable values (e.g., 3-5).",
			})
		}
	}

	return findings
}

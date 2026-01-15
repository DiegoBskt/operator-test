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

package imageregistry

import (
	"context"
	"fmt"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"sigs.k8s.io/controller-runtime/pkg/client"

	assessmentv1alpha1 "github.com/openshift-assessment/cluster-assessment-operator/api/v1alpha1"
	"github.com/openshift-assessment/cluster-assessment-operator/pkg/profiles"
	"github.com/openshift-assessment/cluster-assessment-operator/pkg/validator"
)

const (
	validatorName        = "imageregistry"
	validatorDescription = "Validates OpenShift internal image registry configuration, storage backend, and pruning settings"
	validatorCategory    = "Platform"
)

func init() {
	_ = validator.Register(&ImageRegistryValidator{})
}

// ImageRegistryValidator checks image registry configuration.
type ImageRegistryValidator struct{}

// Name returns the validator name.
func (v *ImageRegistryValidator) Name() string {
	return validatorName
}

// Description returns the validator description.
func (v *ImageRegistryValidator) Description() string {
	return validatorDescription
}

// Category returns the finding category.
func (v *ImageRegistryValidator) Category() string {
	return validatorCategory
}

// Validate performs image registry checks.
func (v *ImageRegistryValidator) Validate(ctx context.Context, c client.Client, profile profiles.Profile) ([]assessmentv1alpha1.Finding, error) {
	var findings []assessmentv1alpha1.Finding

	// Check 1: Image registry configuration
	findings = append(findings, v.checkRegistryConfig(ctx, c, profile)...)

	// Check 2: Image pruner configuration
	findings = append(findings, v.checkImagePruner(ctx, c)...)

	return findings, nil
}

// checkRegistryConfig checks the image registry operator configuration.
func (v *ImageRegistryValidator) checkRegistryConfig(ctx context.Context, c client.Client, profile profiles.Profile) []assessmentv1alpha1.Finding {
	var findings []assessmentv1alpha1.Finding

	// Get the image registry config
	registryConfig := &unstructured.Unstructured{}
	registryConfig.SetGroupVersionKind(schema.GroupVersionKind{
		Group:   "imageregistry.operator.openshift.io",
		Version: "v1",
		Kind:    "Config",
	})

	if err := c.Get(ctx, client.ObjectKey{Name: "cluster"}, registryConfig); err != nil {
		return []assessmentv1alpha1.Finding{{
			ID:          "imageregistry-config-error",
			Validator:   validatorName,
			Category:    validatorCategory,
			Status:      assessmentv1alpha1.FindingStatusFail,
			Title:       "Unable to Check Image Registry",
			Description: fmt.Sprintf("Failed to get image registry config: %v", err),
		}}
	}

	// Check management state
	managementState, found, _ := unstructured.NestedString(registryConfig.Object, "spec", "managementState")
	if !found {
		managementState = "Unknown"
	}

	switch managementState {
	case "Removed":
		findings = append(findings, assessmentv1alpha1.Finding{
			ID:             "imageregistry-removed",
			Validator:      validatorName,
			Category:       validatorCategory,
			Status:         assessmentv1alpha1.FindingStatusWarn,
			Title:          "Image Registry Removed",
			Description:    "The internal image registry is set to Removed state.",
			Impact:         "Internal image builds and image streams will not function.",
			Recommendation: "If internal registry is needed, set managementState to Managed.",
			References: []string{
				"https://docs.openshift.com/container-platform/latest/registry/configuring-registry-operator.html",
			},
		})
	case "Managed":
		findings = append(findings, assessmentv1alpha1.Finding{
			ID:          "imageregistry-managed",
			Validator:   validatorName,
			Category:    validatorCategory,
			Status:      assessmentv1alpha1.FindingStatusPass,
			Title:       "Image Registry Managed",
			Description: "The internal image registry is in Managed state.",
		})
	case "Unmanaged":
		findings = append(findings, assessmentv1alpha1.Finding{
			ID:             "imageregistry-unmanaged",
			Validator:      validatorName,
			Category:       validatorCategory,
			Status:         assessmentv1alpha1.FindingStatusInfo,
			Title:          "Image Registry Unmanaged",
			Description:    "The internal image registry is in Unmanaged state.",
			Impact:         "The registry operator will not manage the registry configuration.",
			Recommendation: "Ensure manual management is intentional and documented.",
		})
	}

	// Check storage configuration
	storage, found, _ := unstructured.NestedMap(registryConfig.Object, "spec", "storage")
	if !found || len(storage) == 0 {
		findings = append(findings, assessmentv1alpha1.Finding{
			ID:             "imageregistry-no-storage",
			Validator:      validatorName,
			Category:       validatorCategory,
			Status:         assessmentv1alpha1.FindingStatusWarn,
			Title:          "Image Registry Storage Not Configured",
			Description:    "The image registry does not have storage configured.",
			Impact:         "Registry may use emptyDir which loses data on pod restart.",
			Recommendation: "Configure persistent storage for the image registry.",
			References: []string{
				"https://docs.openshift.com/container-platform/latest/registry/configuring_registry_storage/configuring-registry-storage-baremetal.html",
			},
		})
	} else {
		// Check for emptyDir
		if _, hasEmptyDir := storage["emptyDir"]; hasEmptyDir {
			status := assessmentv1alpha1.FindingStatusWarn
			if profile.Name == profiles.ProfileDevelopment {
				status = assessmentv1alpha1.FindingStatusInfo
			}
			findings = append(findings, assessmentv1alpha1.Finding{
				ID:             "imageregistry-emptydir",
				Validator:      validatorName,
				Category:       validatorCategory,
				Status:         status,
				Title:          "Image Registry Using EmptyDir Storage",
				Description:    "The image registry is configured with emptyDir storage.",
				Impact:         "All images will be lost when the registry pod restarts.",
				Recommendation: "Configure persistent storage (PVC, S3, Azure Blob, GCS) for production use.",
			})
		} else {
			// Determine storage type
			storageType := "unknown"
			if _, ok := storage["pvc"]; ok {
				storageType = "PVC"
			} else if _, ok := storage["s3"]; ok {
				storageType = "S3"
			} else if _, ok := storage["azure"]; ok {
				storageType = "Azure Blob"
			} else if _, ok := storage["gcs"]; ok {
				storageType = "GCS"
			} else if _, ok := storage["swift"]; ok {
				storageType = "Swift"
			}

			findings = append(findings, assessmentv1alpha1.Finding{
				ID:          "imageregistry-storage-configured",
				Validator:   validatorName,
				Category:    validatorCategory,
				Status:      assessmentv1alpha1.FindingStatusPass,
				Title:       "Image Registry Storage Configured",
				Description: fmt.Sprintf("The image registry is using %s storage.", storageType),
			})
		}
	}

	// Check replicas
	replicas, found, _ := unstructured.NestedInt64(registryConfig.Object, "spec", "replicas")
	if found {
		if replicas < 2 && profile.Name == profiles.ProfileProduction {
			findings = append(findings, assessmentv1alpha1.Finding{
				ID:             "imageregistry-single-replica",
				Validator:      validatorName,
				Category:       validatorCategory,
				Status:         assessmentv1alpha1.FindingStatusWarn,
				Title:          "Image Registry Single Replica",
				Description:    fmt.Sprintf("The image registry is running with %d replica(s).", replicas),
				Impact:         "Single replica reduces availability during updates or failures.",
				Recommendation: "Configure at least 2 replicas for high availability in production.",
			})
		} else if replicas >= 2 {
			findings = append(findings, assessmentv1alpha1.Finding{
				ID:          "imageregistry-ha",
				Validator:   validatorName,
				Category:    validatorCategory,
				Status:      assessmentv1alpha1.FindingStatusPass,
				Title:       "Image Registry High Availability",
				Description: fmt.Sprintf("The image registry is running with %d replicas.", replicas),
			})
		}
	}

	return findings
}

// checkImagePruner checks the image pruner configuration.
func (v *ImageRegistryValidator) checkImagePruner(ctx context.Context, c client.Client) []assessmentv1alpha1.Finding {
	var findings []assessmentv1alpha1.Finding

	// Get the image pruner config
	prunerConfig := &unstructured.Unstructured{}
	prunerConfig.SetGroupVersionKind(schema.GroupVersionKind{
		Group:   "imageregistry.operator.openshift.io",
		Version: "v1",
		Kind:    "ImagePruner",
	})

	if err := c.Get(ctx, client.ObjectKey{Name: "cluster"}, prunerConfig); err != nil {
		findings = append(findings, assessmentv1alpha1.Finding{
			ID:             "imageregistry-pruner-missing",
			Validator:      validatorName,
			Category:       validatorCategory,
			Status:         assessmentv1alpha1.FindingStatusInfo,
			Title:          "Image Pruner Not Configured",
			Description:    "No image pruner configuration found.",
			Impact:         "Old images may accumulate and consume storage.",
			Recommendation: "Configure image pruning to manage registry storage.",
			References: []string{
				"https://docs.openshift.com/container-platform/latest/applications/pruning-objects.html",
			},
		})
		return findings
	}

	// Check if pruning is suspended
	suspend, _, _ := unstructured.NestedBool(prunerConfig.Object, "spec", "suspend")
	if suspend {
		findings = append(findings, assessmentv1alpha1.Finding{
			ID:             "imageregistry-pruner-suspended",
			Validator:      validatorName,
			Category:       validatorCategory,
			Status:         assessmentv1alpha1.FindingStatusInfo,
			Title:          "Image Pruner Suspended",
			Description:    "Image pruning is suspended.",
			Impact:         "Old images will not be automatically cleaned up.",
			Recommendation: "Enable image pruning if storage growth is a concern.",
		})
	} else {
		schedule, found, _ := unstructured.NestedString(prunerConfig.Object, "spec", "schedule")
		if !found || schedule == "" {
			schedule = "default"
		}
		findings = append(findings, assessmentv1alpha1.Finding{
			ID:          "imageregistry-pruner-active",
			Validator:   validatorName,
			Category:    validatorCategory,
			Status:      assessmentv1alpha1.FindingStatusPass,
			Title:       "Image Pruner Active",
			Description: fmt.Sprintf("Image pruning is enabled with schedule: %s", schedule),
		})
	}

	return findings
}

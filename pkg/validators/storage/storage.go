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

package storage

import (
	"context"
	"fmt"
	"strings"

	storagev1 "k8s.io/api/storage/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	assessmentv1alpha1 "github.com/openshift-assessment/cluster-assessment-operator/api/v1alpha1"
	"github.com/openshift-assessment/cluster-assessment-operator/pkg/profiles"
	"github.com/openshift-assessment/cluster-assessment-operator/pkg/validator"
)

const (
	validatorName        = "storage"
	validatorDescription = "Validates storage configuration including StorageClasses and CSI drivers"
	validatorCategory    = "Storage"
)

// List of known supported CSI drivers
var supportedCSIDrivers = map[string]bool{
	"ebs.csi.aws.com":                   true,
	"disk.csi.azure.com":                true,
	"file.csi.azure.com":                true,
	"pd.csi.storage.gke.io":             true,
	"csi.vsphere.vmware.com":            true,
	"kubernetes.io/aws-ebs":             true,
	"kubernetes.io/azure-disk":          true,
	"kubernetes.io/azure-file":          true,
	"kubernetes.io/gce-pd":              true,
	"kubernetes.io/vsphere-volume":      true,
	"cinder.csi.openstack.org":          true,
	"manila.csi.openstack.org":          true,
	"odf.csi.ceph.com":                  true,
	"openshift-storage.rbd.csi.ceph.com": true,
	"openshift-storage.cephfs.csi.ceph.com": true,
	"nfs.csi.k8s.io":                    true,
}

func init() {
	validator.Register(&StorageValidator{})
}

// StorageValidator checks storage configuration.
type StorageValidator struct{}

// Name returns the validator name.
func (v *StorageValidator) Name() string {
	return validatorName
}

// Description returns the validator description.
func (v *StorageValidator) Description() string {
	return validatorDescription
}

// Category returns the finding category.
func (v *StorageValidator) Category() string {
	return validatorCategory
}

// Validate performs storage checks.
func (v *StorageValidator) Validate(ctx context.Context, c client.Client, profile profiles.Profile) ([]assessmentv1alpha1.Finding, error) {
	var findings []assessmentv1alpha1.Finding

	// Check 1: StorageClasses
	findings = append(findings, v.checkStorageClasses(ctx, c, profile)...)

	// Check 2: CSI Drivers
	findings = append(findings, v.checkCSIDrivers(ctx, c)...)

	return findings, nil
}

// checkStorageClasses validates StorageClass configuration.
func (v *StorageValidator) checkStorageClasses(ctx context.Context, c client.Client, profile profiles.Profile) []assessmentv1alpha1.Finding {
	var findings []assessmentv1alpha1.Finding

	scs := &storagev1.StorageClassList{}
	if err := c.List(ctx, scs); err != nil {
		return []assessmentv1alpha1.Finding{{
			ID:          "storage-sc-error",
			Validator:   validatorName,
			Category:    validatorCategory,
			Status:      assessmentv1alpha1.FindingStatusFail,
			Title:       "Unable to Check StorageClasses",
			Description: fmt.Sprintf("Failed to list StorageClasses: %v", err),
		}}
	}

	if len(scs.Items) == 0 {
		return []assessmentv1alpha1.Finding{{
			ID:             "storage-no-sc",
			Validator:      validatorName,
			Category:       validatorCategory,
			Status:         assessmentv1alpha1.FindingStatusWarn,
			Title:          "No StorageClasses Configured",
			Description:    "No StorageClasses are configured in the cluster.",
			Impact:         "Without StorageClasses, PersistentVolumeClaims cannot be dynamically provisioned.",
			Recommendation: "Configure appropriate StorageClasses for your storage backend.",
		}}
	}

	// Check for default StorageClass
	var defaultSC *storagev1.StorageClass
	var defaultSCCount int
	for i := range scs.Items {
		sc := &scs.Items[i]
		if sc.Annotations["storageclass.kubernetes.io/is-default-class"] == "true" {
			defaultSC = sc
			defaultSCCount++
		}
	}

	if defaultSCCount == 0 && profile.Thresholds.RequireDefaultStorageClass {
		findings = append(findings, assessmentv1alpha1.Finding{
			ID:             "storage-no-default-sc",
			Validator:      validatorName,
			Category:       validatorCategory,
			Status:         assessmentv1alpha1.FindingStatusWarn,
			Title:          "No Default StorageClass",
			Description:    "No default StorageClass is configured.",
			Impact:         "PVCs without explicit StorageClass will fail to provision.",
			Recommendation: "Set a default StorageClass with 'kubectl patch storageclass <name> -p '{\"metadata\": {\"annotations\":{\"storageclass.kubernetes.io/is-default-class\":\"true\"}}}'",
		})
	} else if defaultSCCount > 1 {
		findings = append(findings, assessmentv1alpha1.Finding{
			ID:             "storage-multiple-default-sc",
			Validator:      validatorName,
			Category:       validatorCategory,
			Status:         assessmentv1alpha1.FindingStatusWarn,
			Title:          "Multiple Default StorageClasses",
			Description:    fmt.Sprintf("%d StorageClasses are marked as default.", defaultSCCount),
			Impact:         "Having multiple default StorageClasses can cause unpredictable behavior.",
			Recommendation: "Ensure only one StorageClass is marked as default.",
		})
	} else if defaultSC != nil {
		findings = append(findings, assessmentv1alpha1.Finding{
			ID:          "storage-default-sc",
			Validator:   validatorName,
			Category:    validatorCategory,
			Status:      assessmentv1alpha1.FindingStatusPass,
			Title:       "Default StorageClass Configured",
			Description: fmt.Sprintf("Default StorageClass: %s (provisioner: %s)", defaultSC.Name, defaultSC.Provisioner),
		})
	}

	// Report all StorageClasses
	var scNames []string
	for _, sc := range scs.Items {
		scNames = append(scNames, fmt.Sprintf("%s (%s)", sc.Name, sc.Provisioner))
	}
	findings = append(findings, assessmentv1alpha1.Finding{
		ID:          "storage-sc-list",
		Validator:   validatorName,
		Category:    validatorCategory,
		Status:      assessmentv1alpha1.FindingStatusInfo,
		Title:       "Available StorageClasses",
		Description: fmt.Sprintf("Found %d StorageClass(es): %s", len(scs.Items), strings.Join(scNames, ", ")),
	})

	// Check for volume expansion support
	var noExpansion []string
	for _, sc := range scs.Items {
		if sc.AllowVolumeExpansion == nil || !*sc.AllowVolumeExpansion {
			noExpansion = append(noExpansion, sc.Name)
		}
	}
	if len(noExpansion) > 0 {
		findings = append(findings, assessmentv1alpha1.Finding{
			ID:             "storage-no-expansion",
			Validator:      validatorName,
			Category:       validatorCategory,
			Status:         assessmentv1alpha1.FindingStatusInfo,
			Title:          "StorageClasses Without Volume Expansion",
			Description:    fmt.Sprintf("%d StorageClass(es) do not support volume expansion: %s", len(noExpansion), strings.Join(noExpansion, ", ")),
			Recommendation: "Consider enabling volume expansion for StorageClasses if supported by the provisioner.",
		})
	}

	return findings
}

// checkCSIDrivers validates CSI driver configuration.
func (v *StorageValidator) checkCSIDrivers(ctx context.Context, c client.Client) []assessmentv1alpha1.Finding {
	var findings []assessmentv1alpha1.Finding

	drivers := &storagev1.CSIDriverList{}
	if err := c.List(ctx, drivers); err != nil {
		return []assessmentv1alpha1.Finding{{
			ID:          "storage-csi-error",
			Validator:   validatorName,
			Category:    validatorCategory,
			Status:      assessmentv1alpha1.FindingStatusInfo,
			Title:       "Unable to Check CSI Drivers",
			Description: fmt.Sprintf("Failed to list CSI drivers: %v", err),
		}}
	}

	if len(drivers.Items) == 0 {
		return []assessmentv1alpha1.Finding{{
			ID:          "storage-no-csi",
			Validator:   validatorName,
			Category:    validatorCategory,
			Status:      assessmentv1alpha1.FindingStatusInfo,
			Title:       "No CSI Drivers Installed",
			Description: "No CSI drivers are installed in the cluster.",
		}}
	}

	// Categorize drivers
	var supportedDrivers []string
	var unknownDrivers []string

	for _, driver := range drivers.Items {
		if supportedCSIDrivers[driver.Name] {
			supportedDrivers = append(supportedDrivers, driver.Name)
		} else {
			// Check for known patterns
			if strings.Contains(driver.Name, "openshift") || strings.Contains(driver.Name, "redhat") {
				supportedDrivers = append(supportedDrivers, driver.Name)
			} else {
				unknownDrivers = append(unknownDrivers, driver.Name)
			}
		}
	}

	findings = append(findings, assessmentv1alpha1.Finding{
		ID:          "storage-csi-drivers",
		Validator:   validatorName,
		Category:    validatorCategory,
		Status:      assessmentv1alpha1.FindingStatusInfo,
		Title:       "CSI Drivers Installed",
		Description: fmt.Sprintf("Found %d CSI driver(s) installed.", len(drivers.Items)),
	})

	if len(supportedDrivers) > 0 {
		findings = append(findings, assessmentv1alpha1.Finding{
			ID:          "storage-csi-supported",
			Validator:   validatorName,
			Category:    validatorCategory,
			Status:      assessmentv1alpha1.FindingStatusPass,
			Title:       "Supported CSI Drivers",
			Description: fmt.Sprintf("Found %d known/supported CSI driver(s): %s", len(supportedDrivers), strings.Join(supportedDrivers, ", ")),
		})
	}

	if len(unknownDrivers) > 0 {
		findings = append(findings, assessmentv1alpha1.Finding{
			ID:             "storage-csi-unknown",
			Validator:      validatorName,
			Category:       validatorCategory,
			Status:         assessmentv1alpha1.FindingStatusInfo,
			Title:          "Third-Party CSI Drivers",
			Description:    fmt.Sprintf("Found %d third-party CSI driver(s): %s", len(unknownDrivers), strings.Join(unknownDrivers, ", ")),
			Impact:         "Third-party CSI drivers may have different support levels and update schedules.",
			Recommendation: "Ensure third-party CSI drivers are maintained and compatible with your OpenShift version.",
		})
	}

	return findings
}

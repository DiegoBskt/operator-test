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

package machineconfig

import (
	"context"
	"fmt"
	"strings"

	"sigs.k8s.io/controller-runtime/pkg/client"

	assessmentv1alpha1 "github.com/openshift-assessment/cluster-assessment-operator/api/v1alpha1"
	mcv1 "github.com/openshift-assessment/cluster-assessment-operator/pkg/machineconfig"
	"github.com/openshift-assessment/cluster-assessment-operator/pkg/profiles"
	"github.com/openshift-assessment/cluster-assessment-operator/pkg/validator"
)

const (
	validatorName        = "machineconfig"
	validatorDescription = "Validates MachineConfig and MachineConfigPool health and configuration"
	validatorCategory    = "Platform"
)

func init() {
	_ = validator.Register(&MachineConfigValidator{})
}

// MachineConfigValidator checks MachineConfig and MachineConfigPool configurations.
type MachineConfigValidator struct{}

// Name returns the validator name.
func (v *MachineConfigValidator) Name() string {
	return validatorName
}

// Description returns the validator description.
func (v *MachineConfigValidator) Description() string {
	return validatorDescription
}

// Category returns the finding category.
func (v *MachineConfigValidator) Category() string {
	return validatorCategory
}

// Validate performs MachineConfig checks.
func (v *MachineConfigValidator) Validate(ctx context.Context, c client.Client, profile profiles.Profile) ([]assessmentv1alpha1.Finding, error) {
	var findings []assessmentv1alpha1.Finding

	// Check 1: MachineConfigPool status
	findings = append(findings, v.checkMachineConfigPools(ctx, c)...)

	// Check 2: Custom MachineConfigs
	findings = append(findings, v.checkCustomMachineConfigs(ctx, c)...)

	return findings, nil
}

// checkMachineConfigPools validates MachineConfigPool health.
func (v *MachineConfigValidator) checkMachineConfigPools(ctx context.Context, c client.Client) []assessmentv1alpha1.Finding {
	var findings []assessmentv1alpha1.Finding

	mcps := &mcv1.MachineConfigPoolList{}
	if err := c.List(ctx, mcps); err != nil {
		return []assessmentv1alpha1.Finding{{
			ID:          "machineconfig-mcp-error",
			Validator:   validatorName,
			Category:    validatorCategory,
			Status:      assessmentv1alpha1.FindingStatusFail,
			Title:       "Unable to Check MachineConfigPools",
			Description: fmt.Sprintf("Failed to list MachineConfigPools: %v", err),
		}}
	}

	var degradedPools []string
	var updatingPools []string
	var healthyPools []string

	for _, mcp := range mcps.Items {
		isDegraded := false
		isUpdating := false

		for _, condition := range mcp.Status.Conditions {
			switch condition.Type {
			case mcv1.MachineConfigPoolDegraded:
				if condition.Status == "True" {
					isDegraded = true
					degradedPools = append(degradedPools, fmt.Sprintf("%s (%s)", mcp.Name, condition.Message))
				}
			case mcv1.MachineConfigPoolUpdating:
				if condition.Status == "True" {
					isUpdating = true
					updatingPools = append(updatingPools, mcp.Name)
				}
			}
		}

		if !isDegraded && !isUpdating {
			healthyPools = append(healthyPools, mcp.Name)
		}
	}

	// Report degraded pools
	if len(degradedPools) > 0 {
		findings = append(findings, assessmentv1alpha1.Finding{
			ID:             "machineconfig-mcp-degraded",
			Validator:      validatorName,
			Category:       validatorCategory,
			Status:         assessmentv1alpha1.FindingStatusFail,
			Title:          "Degraded MachineConfigPools",
			Description:    fmt.Sprintf("%d MachineConfigPool(s) are degraded: %s", len(degradedPools), strings.Join(degradedPools, "; ")),
			Impact:         "Degraded MachineConfigPools indicate nodes that failed to apply configuration and may be in an inconsistent state.",
			Recommendation: "Investigate the degraded nodes. Check MachineConfigDaemon logs and node status.",
		})
	}

	// Report updating pools
	if len(updatingPools) > 0 {
		findings = append(findings, assessmentv1alpha1.Finding{
			ID:          "machineconfig-mcp-updating",
			Validator:   validatorName,
			Category:    validatorCategory,
			Status:      assessmentv1alpha1.FindingStatusInfo,
			Title:       "MachineConfigPools Updating",
			Description: fmt.Sprintf("%d MachineConfigPool(s) are currently updating: %s", len(updatingPools), strings.Join(updatingPools, ", ")),
		})
	}

	// Report healthy pools
	if len(healthyPools) > 0 {
		findings = append(findings, assessmentv1alpha1.Finding{
			ID:          "machineconfig-mcp-healthy",
			Validator:   validatorName,
			Category:    validatorCategory,
			Status:      assessmentv1alpha1.FindingStatusPass,
			Title:       "Healthy MachineConfigPools",
			Description: fmt.Sprintf("%d MachineConfigPool(s) are healthy: %s", len(healthyPools), strings.Join(healthyPools, ", ")),
		})
	}

	// Check for pending machines
	for _, mcp := range mcps.Items {
		if mcp.Status.MachineCount != mcp.Status.UpdatedMachineCount {
			pending := mcp.Status.MachineCount - mcp.Status.UpdatedMachineCount
			findings = append(findings, assessmentv1alpha1.Finding{
				ID:          fmt.Sprintf("machineconfig-pending-%s", mcp.Name),
				Validator:   validatorName,
				Category:    validatorCategory,
				Status:      assessmentv1alpha1.FindingStatusInfo,
				Title:       fmt.Sprintf("Pending Updates in %s", mcp.Name),
				Description: fmt.Sprintf("%s has %d machine(s) pending update (%d/%d updated)", mcp.Name, pending, mcp.Status.UpdatedMachineCount, mcp.Status.MachineCount),
			})
		}
	}

	return findings
}

// checkCustomMachineConfigs checks for custom MachineConfigs.
func (v *MachineConfigValidator) checkCustomMachineConfigs(ctx context.Context, c client.Client) []assessmentv1alpha1.Finding {
	var findings []assessmentv1alpha1.Finding

	mcs := &mcv1.MachineConfigList{}
	if err := c.List(ctx, mcs); err != nil {
		return findings
	}

	var customMCs []string
	for _, mc := range mcs.Items {
		// Skip rendered and system configs
		if strings.HasPrefix(mc.Name, "rendered-") ||
			strings.HasPrefix(mc.Name, "00-") ||
			strings.HasPrefix(mc.Name, "01-") {
			continue
		}
		customMCs = append(customMCs, mc.Name)
	}

	if len(customMCs) > 0 {
		findings = append(findings, assessmentv1alpha1.Finding{
			ID:             "machineconfig-custom",
			Validator:      validatorName,
			Category:       validatorCategory,
			Status:         assessmentv1alpha1.FindingStatusInfo,
			Title:          "Custom MachineConfigs",
			Description:    fmt.Sprintf("Found %d custom MachineConfig(s): %s", len(customMCs), strings.Join(customMCs, ", ")),
			Impact:         "Custom MachineConfigs modify node configuration and should be reviewed for supportability.",
			Recommendation: "Ensure custom MachineConfigs are documented and aligned with Red Hat support policies.",
			References: []string{
				"https://docs.openshift.com/container-platform/latest/post_installation_configuration/machine-configuration-tasks.html",
			},
		})
	} else {
		findings = append(findings, assessmentv1alpha1.Finding{
			ID:          "machineconfig-no-custom",
			Validator:   validatorName,
			Category:    validatorCategory,
			Status:      assessmentv1alpha1.FindingStatusPass,
			Title:       "No Custom MachineConfigs",
			Description: "No custom MachineConfigs detected beyond default configurations.",
		})
	}

	return findings
}

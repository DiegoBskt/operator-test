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

package version

import (
	"context"
	"fmt"
	"strings"
	"time"

	configv1 "github.com/openshift/api/config/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	assessmentv1alpha1 "github.com/openshift-assessment/cluster-assessment-operator/api/v1alpha1"
	"github.com/openshift-assessment/cluster-assessment-operator/pkg/profiles"
	"github.com/openshift-assessment/cluster-assessment-operator/pkg/validator"
)

const (
	validatorName        = "version"
	validatorDescription = "Validates OpenShift version, upgrade channel, and lifecycle status"
	validatorCategory    = "Platform"
)

func init() {
	_ = validator.Register(&VersionValidator{})
}

// VersionValidator checks OpenShift version and lifecycle status.
type VersionValidator struct{}

// Name returns the validator name.
func (v *VersionValidator) Name() string {
	return validatorName
}

// Description returns the validator description.
func (v *VersionValidator) Description() string {
	return validatorDescription
}

// Category returns the finding category.
func (v *VersionValidator) Category() string {
	return validatorCategory
}

// Validate performs version checks.
func (v *VersionValidator) Validate(ctx context.Context, c client.Client, profile profiles.Profile) ([]assessmentv1alpha1.Finding, error) {
	var findings []assessmentv1alpha1.Finding

	// Get ClusterVersion
	cv := &configv1.ClusterVersion{}
	if err := c.Get(ctx, client.ObjectKey{Name: "version"}, cv); err != nil {
		return nil, fmt.Errorf("failed to get ClusterVersion: %w", err)
	}

	// Check 1: Version information
	findings = append(findings, v.checkVersion(cv))

	// Check 2: Upgrade channel
	findings = append(findings, v.checkChannel(cv, profile))

	// Check 3: Cluster conditions
	findings = append(findings, v.checkConditions(cv)...)

	// Check 4: Update availability
	findings = append(findings, v.checkUpdates(cv, profile))

	// Check 5: Version age
	findings = append(findings, v.checkVersionAge(cv, profile))

	return findings, nil
}

// checkVersion reports the current version.
func (v *VersionValidator) checkVersion(cv *configv1.ClusterVersion) assessmentv1alpha1.Finding {
	version := "unknown"
	if len(cv.Status.History) > 0 {
		version = cv.Status.History[0].Version
	}

	return assessmentv1alpha1.Finding{
		ID:          "version-current",
		Validator:   validatorName,
		Category:    validatorCategory,
		Status:      assessmentv1alpha1.FindingStatusInfo,
		Title:       "OpenShift Version",
		Description: fmt.Sprintf("Cluster is running OpenShift version %s", version),
		References: []string{
			"https://access.redhat.com/support/policy/updates/openshift",
		},
	}
}

// checkChannel validates the upgrade channel configuration.
func (v *VersionValidator) checkChannel(cv *configv1.ClusterVersion, profile profiles.Profile) assessmentv1alpha1.Finding {
	channel := cv.Spec.Channel
	if channel == "" {
		return assessmentv1alpha1.Finding{
			ID:             "version-channel-missing",
			Validator:      validatorName,
			Category:       validatorCategory,
			Status:         assessmentv1alpha1.FindingStatusWarn,
			Title:          "No Upgrade Channel Configured",
			Description:    "The cluster does not have an upgrade channel configured. This prevents receiving update recommendations.",
			Impact:         "Without an upgrade channel, the cluster will not receive update recommendations and may miss critical security patches.",
			Recommendation: "Configure an appropriate upgrade channel (stable, fast, or eus) using: oc adm upgrade channel <channel-name>",
			References: []string{
				"https://docs.openshift.com/container-platform/latest/updating/updating-cluster-cli.html",
			},
		}
	}

	// Check for EUS channel in production
	status := assessmentv1alpha1.FindingStatusPass
	recommendation := ""
	if profile.Name == profiles.ProfileProduction && !strings.Contains(strings.ToLower(channel), "stable") && !strings.Contains(strings.ToLower(channel), "eus") {
		status = assessmentv1alpha1.FindingStatusWarn
		recommendation = "For production environments, consider using stable or EUS (Extended Update Support) channels for better stability."
	}

	return assessmentv1alpha1.Finding{
		ID:             "version-channel",
		Validator:      validatorName,
		Category:       validatorCategory,
		Status:         status,
		Title:          "Upgrade Channel Configuration",
		Description:    fmt.Sprintf("Cluster is configured with upgrade channel: %s", channel),
		Recommendation: recommendation,
		References: []string{
			"https://docs.openshift.com/container-platform/latest/updating/understanding-upgrade-channels-release.html",
		},
	}
}

// checkConditions evaluates cluster version conditions.
func (v *VersionValidator) checkConditions(cv *configv1.ClusterVersion) []assessmentv1alpha1.Finding {
	var findings []assessmentv1alpha1.Finding

	for _, condition := range cv.Status.Conditions {
		switch condition.Type {
		case "Available":
			if condition.Status != configv1.ConditionTrue {
				findings = append(findings, assessmentv1alpha1.Finding{
					ID:             "version-not-available",
					Validator:      validatorName,
					Category:       validatorCategory,
					Status:         assessmentv1alpha1.FindingStatusFail,
					Title:          "Cluster Version Not Available",
					Description:    fmt.Sprintf("ClusterVersion reports not available: %s", condition.Message),
					Impact:         "The cluster may be experiencing issues that affect its availability.",
					Recommendation: "Investigate the cluster operators and resolve any issues. Check 'oc get co' for details.",
				})
			}

		case "Progressing":
			if condition.Status == configv1.ConditionTrue {
				findings = append(findings, assessmentv1alpha1.Finding{
					ID:          "version-progressing",
					Validator:   validatorName,
					Category:    validatorCategory,
					Status:      assessmentv1alpha1.FindingStatusInfo,
					Title:       "Cluster Update In Progress",
					Description: fmt.Sprintf("Cluster is currently updating: %s", condition.Message),
				})
			}

		case "Degraded":
			if condition.Status == configv1.ConditionTrue {
				findings = append(findings, assessmentv1alpha1.Finding{
					ID:             "version-degraded",
					Validator:      validatorName,
					Category:       validatorCategory,
					Status:         assessmentv1alpha1.FindingStatusFail,
					Title:          "Cluster Version Degraded",
					Description:    fmt.Sprintf("ClusterVersion reports degraded state: %s", condition.Message),
					Impact:         "A degraded cluster version indicates issues with cluster operators that may affect stability.",
					Recommendation: "Check degraded cluster operators with 'oc get co' and review their logs for issues.",
				})
			}

		case "RetrievedUpdates":
			if condition.Status != configv1.ConditionTrue {
				findings = append(findings, assessmentv1alpha1.Finding{
					ID:             "version-update-check-failed",
					Validator:      validatorName,
					Category:       validatorCategory,
					Status:         assessmentv1alpha1.FindingStatusWarn,
					Title:          "Unable to Retrieve Updates",
					Description:    fmt.Sprintf("Cannot check for available updates: %s", condition.Message),
					Impact:         "The cluster cannot check for available updates, which may delay applying security patches.",
					Recommendation: "Verify network connectivity to the update server and check proxy settings.",
				})
			}
		}
	}

	// If no issues found, add a pass finding
	if len(findings) == 0 {
		findings = append(findings, assessmentv1alpha1.Finding{
			ID:          "version-conditions-healthy",
			Validator:   validatorName,
			Category:    validatorCategory,
			Status:      assessmentv1alpha1.FindingStatusPass,
			Title:       "Cluster Version Healthy",
			Description: "All ClusterVersion conditions are healthy.",
		})
	}

	return findings
}

// checkUpdates checks for available updates.
func (v *VersionValidator) checkUpdates(cv *configv1.ClusterVersion, profile profiles.Profile) assessmentv1alpha1.Finding {
	availableUpdates := cv.Status.AvailableUpdates

	if len(availableUpdates) == 0 {
		return assessmentv1alpha1.Finding{
			ID:          "version-up-to-date",
			Validator:   validatorName,
			Category:    validatorCategory,
			Status:      assessmentv1alpha1.FindingStatusPass,
			Title:       "Cluster Up to Date",
			Description: "No updates available for the current channel.",
		}
	}

	// List available updates
	var updateVersions []string
	for _, u := range availableUpdates {
		updateVersions = append(updateVersions, u.Version)
	}

	status := assessmentv1alpha1.FindingStatusInfo
	if profile.Name == profiles.ProfileProduction {
		status = assessmentv1alpha1.FindingStatusWarn
	}

	return assessmentv1alpha1.Finding{
		ID:             "version-updates-available",
		Validator:      validatorName,
		Category:       validatorCategory,
		Status:         status,
		Title:          "Updates Available",
		Description:    fmt.Sprintf("Updates available: %s", strings.Join(updateVersions, ", ")),
		Impact:         "Running an older version may mean missing security patches and bug fixes.",
		Recommendation: "Review available updates and plan an upgrade during a maintenance window.",
		References: []string{
			"https://docs.openshift.com/container-platform/latest/updating/updating-cluster-cli.html",
		},
	}
}

// checkVersionAge checks how long since the last update.
func (v *VersionValidator) checkVersionAge(cv *configv1.ClusterVersion, profile profiles.Profile) assessmentv1alpha1.Finding {
	if len(cv.Status.History) == 0 {
		return assessmentv1alpha1.Finding{
			ID:          "version-age-unknown",
			Validator:   validatorName,
			Category:    validatorCategory,
			Status:      assessmentv1alpha1.FindingStatusInfo,
			Title:       "Version Age Unknown",
			Description: "Unable to determine when the cluster was last updated.",
		}
	}

	lastUpdate := cv.Status.History[0].CompletionTime
	if lastUpdate == nil {
		return assessmentv1alpha1.Finding{
			ID:          "version-age-unknown",
			Validator:   validatorName,
			Category:    validatorCategory,
			Status:      assessmentv1alpha1.FindingStatusInfo,
			Title:       "Version Age Unknown",
			Description: "Last update completion time is not available.",
		}
	}

	daysSinceUpdate := int(time.Since(lastUpdate.Time).Hours() / 24)

	if daysSinceUpdate > profile.Thresholds.MaxDaysWithoutUpdate {
		return assessmentv1alpha1.Finding{
			ID:             "version-age-old",
			Validator:      validatorName,
			Category:       validatorCategory,
			Status:         assessmentv1alpha1.FindingStatusWarn,
			Title:          "Cluster Not Updated Recently",
			Description:    fmt.Sprintf("It has been %d days since the last cluster update.", daysSinceUpdate),
			Impact:         "Long periods without updates may indicate missing security patches or improvements.",
			Recommendation: fmt.Sprintf("Consider updating the cluster. For %s environments, updates are recommended every %d days.", profile.Name, profile.Thresholds.MaxDaysWithoutUpdate),
		}
	}

	return assessmentv1alpha1.Finding{
		ID:          "version-age-recent",
		Validator:   validatorName,
		Category:    validatorCategory,
		Status:      assessmentv1alpha1.FindingStatusPass,
		Title:       "Cluster Recently Updated",
		Description: fmt.Sprintf("Cluster was last updated %d days ago.", daysSinceUpdate),
	}
}

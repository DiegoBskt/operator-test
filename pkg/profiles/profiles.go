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

package profiles

// ProfileName represents the name of a baseline profile.
type ProfileName string

const (
	// ProfileProduction is the production baseline profile with strict checks.
	ProfileProduction ProfileName = "production"

	// ProfileDevelopment is the development baseline profile with relaxed checks.
	ProfileDevelopment ProfileName = "development"
)

// Profile defines a baseline configuration profile for assessments.
type Profile struct {
	// Name is the profile identifier.
	Name ProfileName `json:"name"`

	// Description explains the profile's purpose.
	Description string `json:"description"`

	// Strictness indicates how strict the profile is (1-10).
	Strictness int `json:"strictness"`

	// EnabledValidators lists which validators are enabled for this profile.
	// Empty means all validators are enabled.
	EnabledValidators []string `json:"enabledValidators,omitempty"`

	// DisabledChecks lists specific checks to skip.
	DisabledChecks []string `json:"disabledChecks,omitempty"`

	// Thresholds configures check-specific thresholds.
	Thresholds ProfileThresholds `json:"thresholds"`
}

// ProfileThresholds contains configurable thresholds for various checks.
type ProfileThresholds struct {
	// MinControlPlaneNodes is the minimum expected control plane nodes.
	MinControlPlaneNodes int `json:"minControlPlaneNodes"`

	// MinWorkerNodes is the minimum expected worker nodes.
	MinWorkerNodes int `json:"minWorkerNodes"`

	// MaxPodsPerNode is the maximum recommended pods per node.
	MaxPodsPerNode int `json:"maxPodsPerNode"`

	// MaxClusterAdminBindings is the maximum acceptable cluster-admin bindings.
	MaxClusterAdminBindings int `json:"maxClusterAdminBindings"`

	// RequireNetworkPolicy requires NetworkPolicy in namespaces.
	RequireNetworkPolicy bool `json:"requireNetworkPolicy"`

	// RequireResourceQuotas requires ResourceQuotas in namespaces.
	RequireResourceQuotas bool `json:"requireResourceQuotas"`

	// RequireLimitRanges requires LimitRanges in namespaces.
	RequireLimitRanges bool `json:"requireLimitRanges"`

	// MaxDaysWithoutUpdate is the maximum days since the last cluster update.
	MaxDaysWithoutUpdate int `json:"maxDaysWithoutUpdate"`

	// AllowPrivilegedContainers determines if privileged containers trigger warnings.
	AllowPrivilegedContainers bool `json:"allowPrivilegedContainers"`

	// RequireDefaultStorageClass requires a default StorageClass.
	RequireDefaultStorageClass bool `json:"requireDefaultStorageClass"`
}

// GetProfile returns the profile configuration for the given profile name.
func GetProfile(name string) Profile {
	switch ProfileName(name) {
	case ProfileDevelopment:
		return developmentProfile
	case ProfileProduction:
		fallthrough
	default:
		return productionProfile
	}
}

// ListProfiles returns all available profile names.
func ListProfiles() []ProfileName {
	return []ProfileName{ProfileProduction, ProfileDevelopment}
}

// productionProfile is the production baseline with strict checks.
var productionProfile = Profile{
	Name:        ProfileProduction,
	Description: "Production baseline with strict enterprise requirements for high availability, security, and supportability.",
	Strictness:  9,
	Thresholds: ProfileThresholds{
		MinControlPlaneNodes:       3,
		MinWorkerNodes:             3,
		MaxPodsPerNode:             250,
		MaxClusterAdminBindings:    5,
		RequireNetworkPolicy:       true,
		RequireResourceQuotas:      true,
		RequireLimitRanges:         true,
		MaxDaysWithoutUpdate:       90,
		AllowPrivilegedContainers:  false,
		RequireDefaultStorageClass: true,
	},
}

// developmentProfile is the development baseline with relaxed checks.
var developmentProfile = Profile{
	Name:        ProfileDevelopment,
	Description: "Development baseline with relaxed requirements suitable for dev/test environments.",
	Strictness:  4,
	Thresholds: ProfileThresholds{
		MinControlPlaneNodes:       1,
		MinWorkerNodes:             1,
		MaxPodsPerNode:             250,
		MaxClusterAdminBindings:    20,
		RequireNetworkPolicy:       false,
		RequireResourceQuotas:      false,
		RequireLimitRanges:         false,
		MaxDaysWithoutUpdate:       180,
		AllowPrivilegedContainers:  true,
		RequireDefaultStorageClass: false,
	},
}

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

package validator

import (
	"context"

	assessmentv1alpha1 "github.com/openshift-assessment/cluster-assessment-operator/api/v1alpha1"
	"github.com/openshift-assessment/cluster-assessment-operator/pkg/profiles"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// Validator is the interface that all assessment validators must implement.
// Each validator performs read-only checks against the cluster and produces findings.
type Validator interface {
	// Name returns the unique identifier for this validator.
	Name() string

	// Description returns a human-readable description of what this validator checks.
	Description() string

	// Category returns the category grouping for this validator's findings
	// (e.g., "Security", "Networking", "Storage").
	Category() string

	// Validate performs the validation checks and returns findings.
	// The implementation must be strictly read-only.
	Validate(ctx context.Context, client client.Client, profile profiles.Profile) ([]assessmentv1alpha1.Finding, error)
}

// ValidatorConfig provides configuration options for validators.
type ValidatorConfig struct {
	// Profile is the baseline profile being used for the assessment.
	Profile profiles.Profile

	// EnabledChecks specifies which specific checks within a validator to run.
	// If empty, all checks are run.
	EnabledChecks []string

	// Parameters provides validator-specific configuration.
	Parameters map[string]interface{}
}

// ValidatorMetadata provides metadata about a validator for registration and discovery.
type ValidatorMetadata struct {
	// Name is the unique identifier.
	Name string

	// Description explains what the validator checks.
	Description string

	// Category is the finding category.
	Category string

	// SupportedProfiles lists which profiles this validator supports.
	SupportedProfiles []string

	// CheckCount is the number of individual checks this validator performs.
	CheckCount int
}

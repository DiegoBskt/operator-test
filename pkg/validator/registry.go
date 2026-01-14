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
	"fmt"
	"sync"

	assessmentv1alpha1 "github.com/openshift-assessment/cluster-assessment-operator/api/v1alpha1"
	"github.com/openshift-assessment/cluster-assessment-operator/pkg/profiles"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

// Registry manages validator registration and discovery.
type Registry struct {
	mu         sync.RWMutex
	validators map[string]Validator
}

// NewRegistry creates a new validator registry.
func NewRegistry() *Registry {
	return &Registry{
		validators: make(map[string]Validator),
	}
}

// Register adds a validator to the registry.
func (r *Registry) Register(v Validator) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	name := v.Name()
	if _, exists := r.validators[name]; exists {
		return fmt.Errorf("validator %q already registered", name)
	}

	r.validators[name] = v
	return nil
}

// Get retrieves a validator by name.
func (r *Registry) Get(name string) (Validator, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	v, ok := r.validators[name]
	return v, ok
}

// List returns all registered validators.
func (r *Registry) List() []Validator {
	r.mu.RLock()
	defer r.mu.RUnlock()

	validators := make([]Validator, 0, len(r.validators))
	for _, v := range r.validators {
		validators = append(validators, v)
	}
	return validators
}

// Names returns the names of all registered validators.
func (r *Registry) Names() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()

	names := make([]string, 0, len(r.validators))
	for name := range r.validators {
		names = append(names, name)
	}
	return names
}

// Runner executes validators and collects findings.
type Runner struct {
	registry *Registry
	client   client.Client
}

// NewRunner creates a new validator runner.
func NewRunner(registry *Registry, client client.Client) *Runner {
	return &Runner{
		registry: registry,
		client:   client,
	}
}

// RunAll executes all registered validators.
func (r *Runner) RunAll(ctx context.Context, profile profiles.Profile) ([]assessmentv1alpha1.Finding, error) {
	return r.Run(ctx, profile, nil)
}

// Run executes the specified validators (or all if validatorNames is empty).
func (r *Runner) Run(ctx context.Context, profile profiles.Profile, validatorNames []string) ([]assessmentv1alpha1.Finding, error) {
	logger := log.FromContext(ctx)

	var validators []Validator
	if len(validatorNames) == 0 {
		validators = r.registry.List()
	} else {
		for _, name := range validatorNames {
			v, ok := r.registry.Get(name)
			if !ok {
				logger.Info("Validator not found, skipping", "validator", name)
				continue
			}
			validators = append(validators, v)
		}
	}

	var allFindings []assessmentv1alpha1.Finding

	for _, v := range validators {
		logger.Info("Running validator", "validator", v.Name(), "category", v.Category())

		findings, err := v.Validate(ctx, r.client, profile)
		if err != nil {
			// Log error but continue with other validators
			logger.Error(err, "Validator failed", "validator", v.Name())
			// Add a finding for the failed validator
			allFindings = append(allFindings, assessmentv1alpha1.Finding{
				ID:          fmt.Sprintf("%s-error", v.Name()),
				Validator:   v.Name(),
				Category:    v.Category(),
				Status:      assessmentv1alpha1.FindingStatusFail,
				Title:       fmt.Sprintf("Validator %s encountered an error", v.Name()),
				Description: fmt.Sprintf("The validator failed to complete: %v", err),
				Impact:      "Assessment results for this validator are incomplete.",
			})
			continue
		}

		allFindings = append(allFindings, findings...)
		logger.Info("Validator completed", "validator", v.Name(), "findings", len(findings))
	}

	return allFindings, nil
}

// defaultRegistry is the global validator registry.
var defaultRegistry = NewRegistry()

// DefaultRegistry returns the global validator registry.
func DefaultRegistry() *Registry {
	return defaultRegistry
}

// Register adds a validator to the default registry.
func Register(v Validator) error {
	return defaultRegistry.Register(v)
}

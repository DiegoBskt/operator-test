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

package etcdbackup

import (
	"context"
	"fmt"
	"time"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"sigs.k8s.io/controller-runtime/pkg/client"

	assessmentv1alpha1 "github.com/openshift-assessment/cluster-assessment-operator/api/v1alpha1"
	"github.com/openshift-assessment/cluster-assessment-operator/pkg/profiles"
	"github.com/openshift-assessment/cluster-assessment-operator/pkg/validator"
)

const (
	validatorName        = "etcdbackup"
	validatorDescription = "Validates etcd backup configuration and status"
	validatorCategory    = "Platform"
)

func init() {
	validator.Register(&EtcdBackupValidator{})
}

// EtcdBackupValidator checks etcd backup configuration.
type EtcdBackupValidator struct{}

// Name returns the validator name.
func (v *EtcdBackupValidator) Name() string {
	return validatorName
}

// Description returns the validator description.
func (v *EtcdBackupValidator) Description() string {
	return validatorDescription
}

// Category returns the finding category.
func (v *EtcdBackupValidator) Category() string {
	return validatorCategory
}

// Validate performs etcd backup configuration checks.
func (v *EtcdBackupValidator) Validate(ctx context.Context, c client.Client, profile profiles.Profile) ([]assessmentv1alpha1.Finding, error) {
	var findings []assessmentv1alpha1.Finding

	// Check for OADP (OpenShift API for Data Protection)
	findings = append(findings, v.checkOADP(ctx, c)...)

	// Check for etcd backup CronJobs
	findings = append(findings, v.checkBackupCronJobs(ctx, c)...)

	// Check for Velero configuration
	findings = append(findings, v.checkVelero(ctx, c)...)

	// If no backup mechanism found, warn
	if len(findings) == 0 {
		findings = append(findings, assessmentv1alpha1.Finding{
			ID:             "etcdbackup-not-configured",
			Validator:      validatorName,
			Category:       validatorCategory,
			Status:         assessmentv1alpha1.FindingStatusWarn,
			Title:          "No Backup Solution Detected",
			Description:    "No etcd backup configuration was detected. Consider implementing a backup strategy.",
			Impact:         "Without backups, cluster recovery after data loss may not be possible.",
			Recommendation: "Configure OADP (OpenShift API for Data Protection) or a custom etcd backup solution.",
			References: []string{
				"https://docs.openshift.com/container-platform/latest/backup_and_restore/control_plane_backup_and_restore/backing-up-etcd.html",
			},
		})
	}

	return findings, nil
}

// checkOADP checks for OpenShift API for Data Protection installation.
func (v *EtcdBackupValidator) checkOADP(ctx context.Context, c client.Client) []assessmentv1alpha1.Finding {
	var findings []assessmentv1alpha1.Finding

	// Check for OADP DataProtectionApplication CR
	dpaGVK := schema.GroupVersionKind{
		Group:   "oadp.openshift.io",
		Version: "v1alpha1",
		Kind:    "DataProtectionApplicationList",
	}

	dpaList := &unstructured.UnstructuredList{}
	dpaList.SetGroupVersionKind(dpaGVK)

	if err := c.List(ctx, dpaList); err == nil && len(dpaList.Items) > 0 {
		for _, dpa := range dpaList.Items {
			name, _, _ := unstructured.NestedString(dpa.Object, "metadata", "name")
			namespace, _, _ := unstructured.NestedString(dpa.Object, "metadata", "namespace")

			// Check status
			phase, _, _ := unstructured.NestedString(dpa.Object, "status", "phase")

			if phase == "Reconciled" || phase == "" {
				findings = append(findings, assessmentv1alpha1.Finding{
					ID:          fmt.Sprintf("etcdbackup-oadp-%s", name),
					Validator:   validatorName,
					Category:    validatorCategory,
					Status:      assessmentv1alpha1.FindingStatusPass,
					Title:       "OADP Configured",
					Description: fmt.Sprintf("OpenShift API for Data Protection is configured: %s/%s", namespace, name),
				})
			} else {
				findings = append(findings, assessmentv1alpha1.Finding{
					ID:             fmt.Sprintf("etcdbackup-oadp-issue-%s", name),
					Validator:      validatorName,
					Category:       validatorCategory,
					Status:         assessmentv1alpha1.FindingStatusWarn,
					Title:          "OADP Configuration Issue",
					Description:    fmt.Sprintf("OADP %s/%s is in phase: %s", namespace, name, phase),
					Recommendation: "Check the OADP operator logs and DataProtectionApplication status.",
				})
			}
		}
	}

	return findings
}

// checkBackupCronJobs checks for etcd backup CronJobs.
func (v *EtcdBackupValidator) checkBackupCronJobs(ctx context.Context, c client.Client) []assessmentv1alpha1.Finding {
	var findings []assessmentv1alpha1.Finding

	// Check for backup-related ConfigMaps or CronJobs in openshift-etcd namespace
	cmList := &corev1.ConfigMapList{}
	if err := c.List(ctx, cmList, client.InNamespace("openshift-etcd")); err == nil {
		for _, cm := range cmList.Items {
			if cm.Name == "etcd-backup-config" || cm.Name == "cluster-backup-config" {
				findings = append(findings, assessmentv1alpha1.Finding{
					ID:          "etcdbackup-config-found",
					Validator:   validatorName,
					Category:    validatorCategory,
					Status:      assessmentv1alpha1.FindingStatusPass,
					Title:       "Etcd Backup Configuration Found",
					Description: fmt.Sprintf("Found backup configuration: %s", cm.Name),
				})
			}
		}
	}

	// Check for backup CronJobs in any namespace
	cronJobGVK := schema.GroupVersionKind{
		Group:   "batch",
		Version: "v1",
		Kind:    "CronJobList",
	}

	cronJobList := &unstructured.UnstructuredList{}
	cronJobList.SetGroupVersionKind(cronJobGVK)

	if err := c.List(ctx, cronJobList); err == nil {
		for _, cj := range cronJobList.Items {
			name, _, _ := unstructured.NestedString(cj.Object, "metadata", "name")
			namespace, _, _ := unstructured.NestedString(cj.Object, "metadata", "namespace")

			// Check for backup-related CronJobs
			if containsBackupKeyword(name) {
				lastSchedule, found, _ := unstructured.NestedString(cj.Object, "status", "lastScheduleTime")

				status := assessmentv1alpha1.FindingStatusPass
				desc := fmt.Sprintf("Backup CronJob found: %s/%s", namespace, name)

				if found && lastSchedule != "" {
					// Parse last schedule time to check if it's recent
					if t, err := time.Parse(time.RFC3339, lastSchedule); err == nil {
						if time.Since(t) > 7*24*time.Hour {
							status = assessmentv1alpha1.FindingStatusWarn
							desc = fmt.Sprintf("Backup CronJob %s/%s hasn't run in over 7 days", namespace, name)
						}
					}
				}

				findings = append(findings, assessmentv1alpha1.Finding{
					ID:          fmt.Sprintf("etcdbackup-cronjob-%s-%s", namespace, name),
					Validator:   validatorName,
					Category:    validatorCategory,
					Status:      status,
					Title:       "Backup CronJob Detected",
					Description: desc,
				})
			}
		}
	}

	return findings
}

// checkVelero checks for Velero installation.
func (v *EtcdBackupValidator) checkVelero(ctx context.Context, c client.Client) []assessmentv1alpha1.Finding {
	var findings []assessmentv1alpha1.Finding

	// Check for Velero namespace
	ns := &corev1.Namespace{}
	err := c.Get(ctx, client.ObjectKey{Name: "velero"}, ns)
	if err == nil {
		findings = append(findings, assessmentv1alpha1.Finding{
			ID:          "etcdbackup-velero",
			Validator:   validatorName,
			Category:    validatorCategory,
			Status:      assessmentv1alpha1.FindingStatusPass,
			Title:       "Velero Namespace Found",
			Description: "Velero backup solution appears to be installed.",
		})
	}

	// Also check openshift-adp namespace (OADP uses this)
	err = c.Get(ctx, client.ObjectKey{Name: "openshift-adp"}, ns)
	if err == nil {
		findings = append(findings, assessmentv1alpha1.Finding{
			ID:          "etcdbackup-oadp-namespace",
			Validator:   validatorName,
			Category:    validatorCategory,
			Status:      assessmentv1alpha1.FindingStatusPass,
			Title:       "OADP Namespace Present",
			Description: "OpenShift API for Data Protection namespace exists.",
		})
	}

	return findings
}

func containsBackupKeyword(name string) bool {
	keywords := []string{"backup", "etcd-backup", "cluster-backup", "velero", "oadp"}
	for _, kw := range keywords {
		if contains(name, kw) {
			return true
		}
	}
	return false
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsAt(s, substr))
}

func containsAt(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

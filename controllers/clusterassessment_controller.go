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

package controllers

import (
	"context"
	"fmt"
	"time"

	"github.com/robfig/cron/v3"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/util/retry"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

	configv1 "github.com/openshift/api/config/v1"

	assessmentv1alpha1 "github.com/openshift-assessment/cluster-assessment-operator/api/v1alpha1"
	"github.com/openshift-assessment/cluster-assessment-operator/pkg/profiles"
	"github.com/openshift-assessment/cluster-assessment-operator/pkg/report"
	"github.com/openshift-assessment/cluster-assessment-operator/pkg/validator"
)

// ClusterAssessmentReconciler reconciles a ClusterAssessment object
type ClusterAssessmentReconciler struct {
	client.Client
	Scheme   *runtime.Scheme
	Registry *validator.Registry
}

// +kubebuilder:rbac:groups=assessment.openshift.io,resources=clusterassessments,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=assessment.openshift.io,resources=clusterassessments/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=assessment.openshift.io,resources=clusterassessments/finalizers,verbs=update
// +kubebuilder:rbac:groups="",resources=nodes;namespaces;pods;services;configmaps;secrets;persistentvolumes;persistentvolumeclaims;serviceaccounts,verbs=get;list;watch
// +kubebuilder:rbac:groups="",resources=configmaps,verbs=create;update;patch;delete
// +kubebuilder:rbac:groups=config.openshift.io,resources=*,verbs=get;list;watch
// +kubebuilder:rbac:groups=machineconfiguration.openshift.io,resources=*,verbs=get;list;watch
// +kubebuilder:rbac:groups=security.openshift.io,resources=securitycontextconstraints,verbs=get;list;watch
// +kubebuilder:rbac:groups=storage.k8s.io,resources=storageclasses;csidrivers;csinodes,verbs=get;list;watch
// +kubebuilder:rbac:groups=networking.k8s.io,resources=networkpolicies;ingresses,verbs=get;list;watch
// +kubebuilder:rbac:groups=rbac.authorization.k8s.io,resources=clusterroles;clusterrolebindings;roles;rolebindings,verbs=get;list;watch
// +kubebuilder:rbac:groups=operator.openshift.io,resources=*,verbs=get;list;watch
// +kubebuilder:rbac:groups=monitoring.coreos.com,resources=*,verbs=get;list;watch
// +kubebuilder:rbac:groups=apps,resources=deployments;daemonsets;statefulsets;replicasets,verbs=get;list;watch

// Reconcile handles ClusterAssessment reconciliation.
func (r *ClusterAssessmentReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := log.FromContext(ctx)

	// Fetch the ClusterAssessment instance
	assessment := &assessmentv1alpha1.ClusterAssessment{}
	if err := r.Get(ctx, req.NamespacedName, assessment); err != nil {
		if errors.IsNotFound(err) {
			logger.Info("ClusterAssessment resource not found, ignoring")
			return ctrl.Result{}, nil
		}
		logger.Error(err, "Failed to get ClusterAssessment")
		return ctrl.Result{}, err
	}

	// Check if this is a scheduled assessment
	if assessment.Spec.Schedule != "" {
		return r.reconcileScheduled(ctx, assessment)
	}

	// One-time assessment
	return r.reconcileOneTime(ctx, assessment)
}

// reconcileOneTime handles one-time assessments.
func (r *ClusterAssessmentReconciler) reconcileOneTime(ctx context.Context, assessment *assessmentv1alpha1.ClusterAssessment) (ctrl.Result, error) {
	logger := log.FromContext(ctx)

	// Skip if already completed
	if assessment.Status.Phase == assessmentv1alpha1.PhaseCompleted {
		return ctrl.Result{}, nil
	}

	// Skip if suspended
	if assessment.Spec.Suspend {
		logger.Info("Assessment is suspended")
		return ctrl.Result{}, nil
	}

	// Run the assessment
	return r.runAssessment(ctx, assessment)
}

// reconcileScheduled handles scheduled assessments.
func (r *ClusterAssessmentReconciler) reconcileScheduled(ctx context.Context, assessment *assessmentv1alpha1.ClusterAssessment) (ctrl.Result, error) {
	logger := log.FromContext(ctx)

	// Skip if suspended
	if assessment.Spec.Suspend {
		logger.Info("Scheduled assessment is suspended")
		return ctrl.Result{}, nil
	}

	// Parse the cron schedule
	schedule, err := cron.ParseStandard(assessment.Spec.Schedule)
	if err != nil {
		logger.Error(err, "Invalid cron schedule")
		return r.updateStatus(ctx, assessment, assessmentv1alpha1.PhaseFailed,
			fmt.Sprintf("Invalid cron schedule: %v", err))
	}

	now := time.Now()

	// Calculate next run time
	var nextRun time.Time
	if assessment.Status.LastRunTime != nil {
		nextRun = schedule.Next(assessment.Status.LastRunTime.Time)
	} else {
		// First run - schedule for now
		nextRun = now
	}

	// Update next run time in status
	assessment.Status.NextRunTime = &metav1.Time{Time: nextRun}

	// Check if it's time to run
	if now.Before(nextRun) {
		// Not time yet, requeue for next run
		requeueAfter := nextRun.Sub(now)
		logger.Info("Scheduled assessment not due yet", "nextRun", nextRun, "requeueAfter", requeueAfter)
		if err := r.Status().Update(ctx, assessment); err != nil {
			logger.Error(err, "Failed to update status")
			return ctrl.Result{}, err
		}
		return ctrl.Result{RequeueAfter: requeueAfter}, nil
	}

	// Time to run!
	logger.Info("Running scheduled assessment")
	return r.runAssessment(ctx, assessment)
}

// runAssessment executes the assessment.
func (r *ClusterAssessmentReconciler) runAssessment(ctx context.Context, assessment *assessmentv1alpha1.ClusterAssessment) (ctrl.Result, error) {
	logger := log.FromContext(ctx)

	// Update status to Running
	if _, err := r.updateStatus(ctx, assessment, assessmentv1alpha1.PhaseRunning, "Assessment in progress"); err != nil {
		return ctrl.Result{}, err
	}

	// Get the profile
	profile := profiles.GetProfile(assessment.Spec.Profile)
	logger.Info("Using profile", "profile", profile.Name)

	// Collect cluster info
	clusterInfo, err := r.collectClusterInfo(ctx)
	if err != nil {
		logger.Error(err, "Failed to collect cluster info")
		// Continue anyway, cluster info is optional
	}
	assessment.Status.ClusterInfo = clusterInfo

	// Create validator runner
	runner := validator.NewRunner(r.Registry, r.Client)

	// Run validators
	findings, err := runner.Run(ctx, profile, assessment.Spec.Validators)
	if err != nil {
		logger.Error(err, "Assessment failed")
		return r.updateStatus(ctx, assessment, assessmentv1alpha1.PhaseFailed,
			fmt.Sprintf("Assessment failed: %v", err))
	}

	// Update findings
	assessment.Status.Findings = findings

	// Calculate summary
	assessment.Status.Summary = r.calculateSummary(findings, string(profile.Name))

	// Generate and store report
	if assessment.Spec.ReportStorage.ConfigMap != nil && assessment.Spec.ReportStorage.ConfigMap.Enabled {
		if err := r.storeReportInConfigMap(ctx, assessment); err != nil {
			logger.Error(err, "Failed to store report in ConfigMap")
		}
	}

	// Export to Git if configured
	if assessment.Spec.ReportStorage.Git != nil && assessment.Spec.ReportStorage.Git.Enabled {
		if err := r.exportToGit(ctx, assessment); err != nil {
			logger.Error(err, "Failed to export report to Git")
		}
	}

	// Update status to Completed with retry on conflict
	err = retry.RetryOnConflict(retry.DefaultRetry, func() error {
		// Re-fetch the latest version
		latest := &assessmentv1alpha1.ClusterAssessment{}
		if err := r.Get(ctx, client.ObjectKeyFromObject(assessment), latest); err != nil {
			return err
		}

		// Update status fields
		now := metav1.Now()
		latest.Status.LastRunTime = &now
		latest.Status.Phase = assessmentv1alpha1.PhaseCompleted
		latest.Status.Message = fmt.Sprintf("Assessment completed with %d findings", len(findings))
		latest.Status.ClusterInfo = clusterInfo
		latest.Status.Findings = findings
		latest.Status.Summary = r.calculateSummary(findings, string(profile.Name))
		latest.Status.ReportConfigMap = assessment.Status.ReportConfigMap

		// Update conditions
		latest.Status.Conditions = []metav1.Condition{
			{
				Type:               "Ready",
				Status:             metav1.ConditionTrue,
				LastTransitionTime: now,
				Reason:             "AssessmentCompleted",
				Message:            latest.Status.Message,
			},
		}

		return r.Status().Update(ctx, latest)
	})
	if err != nil {
		logger.Error(err, "Failed to update status")
		return ctrl.Result{}, err
	}

	logger.Info("Assessment completed", "findings", len(findings))

	// If scheduled, requeue for next run
	if assessment.Spec.Schedule != "" {
		schedule, _ := cron.ParseStandard(assessment.Spec.Schedule)
		now := time.Now()
		nextRun := schedule.Next(now)
		requeueAfter := nextRun.Sub(now)
		return ctrl.Result{RequeueAfter: requeueAfter}, nil
	}

	return ctrl.Result{}, nil
}

// collectClusterInfo gathers metadata about the cluster.
func (r *ClusterAssessmentReconciler) collectClusterInfo(ctx context.Context) (assessmentv1alpha1.ClusterInfo, error) {
	info := assessmentv1alpha1.ClusterInfo{}

	// Get ClusterVersion
	cv := &configv1.ClusterVersion{}
	if err := r.Get(ctx, client.ObjectKey{Name: "version"}, cv); err == nil {
		info.ClusterID = string(cv.Spec.ClusterID)
		if len(cv.Status.History) > 0 {
			info.ClusterVersion = cv.Status.History[0].Version
		}
		info.Channel = cv.Spec.Channel
	}

	// Get Infrastructure
	infra := &configv1.Infrastructure{}
	if err := r.Get(ctx, client.ObjectKey{Name: "cluster"}, infra); err == nil {
		info.Platform = string(infra.Status.PlatformStatus.Type)
	}

	// Count nodes
	nodes := &corev1.NodeList{}
	if err := r.List(ctx, nodes); err == nil {
		info.NodeCount = len(nodes.Items)
		for _, node := range nodes.Items {
			if _, ok := node.Labels["node-role.kubernetes.io/master"]; ok {
				info.ControlPlaneNodes++
			}
			if _, ok := node.Labels["node-role.kubernetes.io/control-plane"]; ok {
				info.ControlPlaneNodes++
			}
			if _, ok := node.Labels["node-role.kubernetes.io/worker"]; ok {
				info.WorkerNodes++
			}
		}
	}

	return info, nil
}

// calculateSummary computes the assessment summary from findings.
func (r *ClusterAssessmentReconciler) calculateSummary(findings []assessmentv1alpha1.Finding, profileName string) assessmentv1alpha1.AssessmentSummary {
	summary := assessmentv1alpha1.AssessmentSummary{
		TotalChecks: len(findings),
		ProfileUsed: profileName,
	}

	for _, f := range findings {
		switch f.Status {
		case assessmentv1alpha1.FindingStatusPass:
			summary.PassCount++
		case assessmentv1alpha1.FindingStatusWarn:
			summary.WarnCount++
		case assessmentv1alpha1.FindingStatusFail:
			summary.FailCount++
		case assessmentv1alpha1.FindingStatusInfo:
			summary.InfoCount++
		}
	}

	// Calculate a simple score (0-100)
	if summary.TotalChecks > 0 {
		// Weight: Pass=100, Info=80, Warn=50, Fail=0
		score := (summary.PassCount*100 + summary.InfoCount*80 + summary.WarnCount*50) / summary.TotalChecks
		summary.Score = &score
	}

	return summary
}

// storeReportInConfigMap creates a ConfigMap with the full report.
func (r *ClusterAssessmentReconciler) storeReportInConfigMap(ctx context.Context, assessment *assessmentv1alpha1.ClusterAssessment) error {
	logger := log.FromContext(ctx)

	// Generate report
	reportData, err := report.GenerateJSON(assessment)
	if err != nil {
		return fmt.Errorf("failed to generate report: %w", err)
	}

	// Determine ConfigMap name
	cmName := assessment.Spec.ReportStorage.ConfigMap.Name
	if cmName == "" {
		cmName = fmt.Sprintf("%s-report", assessment.Name)
	}

	// Create or update ConfigMap
	cm := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      cmName,
			Namespace: "openshift-cluster-assessment",
			Labels: map[string]string{
				"app.kubernetes.io/name":       "cluster-assessment-operator",
				"app.kubernetes.io/managed-by": "cluster-assessment-operator",
				"assessment.openshift.io/name": assessment.Name,
			},
		},
		Data: map[string]string{
			"report.json": string(reportData),
		},
	}

	// Set owner reference
	if err := ctrl.SetControllerReference(assessment, cm, r.Scheme); err != nil {
		logger.Error(err, "Failed to set owner reference on ConfigMap")
	}

	// Create or update
	existingCM := &corev1.ConfigMap{}
	err = r.Get(ctx, client.ObjectKey{Name: cm.Name, Namespace: cm.Namespace}, existingCM)
	if errors.IsNotFound(err) {
		if err := r.Create(ctx, cm); err != nil {
			return fmt.Errorf("failed to create ConfigMap: %w", err)
		}
	} else if err != nil {
		return fmt.Errorf("failed to get ConfigMap: %w", err)
	} else {
		existingCM.Data = cm.Data
		existingCM.Labels = cm.Labels
		if err := r.Update(ctx, existingCM); err != nil {
			return fmt.Errorf("failed to update ConfigMap: %w", err)
		}
	}

	assessment.Status.ReportConfigMap = cmName
	logger.Info("Report stored in ConfigMap", "configMap", cmName)
	return nil
}

// exportToGit exports the report to a Git repository.
func (r *ClusterAssessmentReconciler) exportToGit(ctx context.Context, assessment *assessmentv1alpha1.ClusterAssessment) error {
	// Git export will be implemented using go-git
	// For now, log that it would export
	logger := log.FromContext(ctx)
	logger.Info("Git export requested",
		"url", assessment.Spec.ReportStorage.Git.URL,
		"branch", assessment.Spec.ReportStorage.Git.Branch,
		"path", assessment.Spec.ReportStorage.Git.Path)

	// TODO: Implement Git export using go-git
	return nil
}

// updateStatus updates the assessment status.
func (r *ClusterAssessmentReconciler) updateStatus(ctx context.Context, assessment *assessmentv1alpha1.ClusterAssessment, phase, message string) (ctrl.Result, error) {
	assessment.Status.Phase = phase
	assessment.Status.Message = message

	if err := r.Status().Update(ctx, assessment); err != nil {
		return ctrl.Result{}, err
	}
	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *ClusterAssessmentReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&assessmentv1alpha1.ClusterAssessment{}).
		Owns(&corev1.ConfigMap{}).
		Complete(r)
}

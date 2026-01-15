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

package main

import (
	"flag"
	"os"

	"k8s.io/apimachinery/pkg/runtime"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/healthz"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
	metricsserver "sigs.k8s.io/controller-runtime/pkg/metrics/server"

	configv1 "github.com/openshift/api/config/v1"

	assessmentv1alpha1 "github.com/openshift-assessment/cluster-assessment-operator/api/v1alpha1"
	"github.com/openshift-assessment/cluster-assessment-operator/controllers"
	"github.com/openshift-assessment/cluster-assessment-operator/pkg/machineconfig"
	"github.com/openshift-assessment/cluster-assessment-operator/pkg/validator"

	// Import validators to register them
	_ "github.com/openshift-assessment/cluster-assessment-operator/pkg/validators/apiserver"
	_ "github.com/openshift-assessment/cluster-assessment-operator/pkg/validators/certificates"
	_ "github.com/openshift-assessment/cluster-assessment-operator/pkg/validators/compliance"
	_ "github.com/openshift-assessment/cluster-assessment-operator/pkg/validators/costoptimization"
	_ "github.com/openshift-assessment/cluster-assessment-operator/pkg/validators/deprecation"
	_ "github.com/openshift-assessment/cluster-assessment-operator/pkg/validators/etcdbackup"
	_ "github.com/openshift-assessment/cluster-assessment-operator/pkg/validators/imageregistry"
	_ "github.com/openshift-assessment/cluster-assessment-operator/pkg/validators/logging"
	_ "github.com/openshift-assessment/cluster-assessment-operator/pkg/validators/machineconfig"
	_ "github.com/openshift-assessment/cluster-assessment-operator/pkg/validators/monitoring"
	_ "github.com/openshift-assessment/cluster-assessment-operator/pkg/validators/networking"
	_ "github.com/openshift-assessment/cluster-assessment-operator/pkg/validators/networkpolicyaudit"
	_ "github.com/openshift-assessment/cluster-assessment-operator/pkg/validators/nodes"
	_ "github.com/openshift-assessment/cluster-assessment-operator/pkg/validators/operators"
	_ "github.com/openshift-assessment/cluster-assessment-operator/pkg/validators/resourcequotas"
	_ "github.com/openshift-assessment/cluster-assessment-operator/pkg/validators/security"
	_ "github.com/openshift-assessment/cluster-assessment-operator/pkg/validators/storage"
	_ "github.com/openshift-assessment/cluster-assessment-operator/pkg/validators/version"
)

var (
	scheme   = runtime.NewScheme()
	setupLog = ctrl.Log.WithName("setup")
)

func init() {
	utilruntime.Must(clientgoscheme.AddToScheme(scheme))
	utilruntime.Must(assessmentv1alpha1.AddToScheme(scheme))
	utilruntime.Must(configv1.AddToScheme(scheme))
	utilruntime.Must(machineconfig.AddToScheme(scheme))
}

func main() {
	var metricsAddr string
	var enableLeaderElection bool
	var probeAddr string

	flag.StringVar(&metricsAddr, "metrics-bind-address", ":8080", "The address the metric endpoint binds to.")
	flag.StringVar(&probeAddr, "health-probe-bind-address", ":8081", "The address the probe endpoint binds to.")
	flag.BoolVar(&enableLeaderElection, "leader-elect", false,
		"Enable leader election for controller manager. "+
			"Enabling this will ensure there is only one active controller manager.")

	opts := zap.Options{
		Development: true,
	}
	opts.BindFlags(flag.CommandLine)
	flag.Parse()

	ctrl.SetLogger(zap.New(zap.UseFlagOptions(&opts)))

	setupLog.Info("Starting Cluster Assessment Operator")

	mgr, err := ctrl.NewManager(ctrl.GetConfigOrDie(), ctrl.Options{
		Scheme: scheme,
		Metrics: metricsserver.Options{
			BindAddress: metricsAddr,
		},
		HealthProbeBindAddress: probeAddr,
		LeaderElection:         enableLeaderElection,
		LeaderElectionID:       "cluster-assessment-operator.openshift.io",
	})
	if err != nil {
		setupLog.Error(err, "unable to start manager")
		os.Exit(1)
	}

	// Log registered validators
	registry := validator.DefaultRegistry()
	setupLog.Info("Registered validators", "count", len(registry.Names()), "validators", registry.Names())

	if err = (&controllers.ClusterAssessmentReconciler{
		Client:   mgr.GetClient(),
		Scheme:   mgr.GetScheme(),
		Registry: registry,
	}).SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "ClusterAssessment")
		os.Exit(1)
	}

	if err := mgr.AddHealthzCheck("healthz", healthz.Ping); err != nil {
		setupLog.Error(err, "unable to set up health check")
		os.Exit(1)
	}
	if err := mgr.AddReadyzCheck("readyz", healthz.Ping); err != nil {
		setupLog.Error(err, "unable to set up ready check")
		os.Exit(1)
	}

	setupLog.Info("starting manager")
	if err := mgr.Start(ctrl.SetupSignalHandler()); err != nil {
		setupLog.Error(err, "problem running manager")
		os.Exit(1)
	}
}

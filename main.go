/*
Copyright 2022.

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

	"github.com/konflux-ci/internal-services/metadata"
	"github.com/konflux-ci/internal-services/tekton"
	"github.com/konflux-ci/operator-toolkit/controller"
	tektonv1 "github.com/tektoncd/pipeline/pkg/apis/pipeline/v1"
	"go.uber.org/zap/zapcore"

	"k8s.io/apimachinery/pkg/labels"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"sigs.k8s.io/controller-runtime/pkg/cache"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/cluster"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/metrics/server"
	"sigs.k8s.io/controller-runtime/pkg/webhook"

	"github.com/konflux-ci/internal-services/controllers"

	// Import all Kubernetes client auth plugins (e.g. Azure, GCP, OIDC, etc.)
	// to ensure that exec-entrypoint and run can make use of them.
	_ "k8s.io/client-go/plugin/pkg/client/auth"

	"k8s.io/apimachinery/pkg/runtime"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/healthz"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"

	konfluxciv1alpha1 "github.com/konflux-ci/internal-services/api/v1alpha1"
	//+kubebuilder:scaffold:imports
)

var (
	scheme   = runtime.NewScheme()
	setupLog = ctrl.Log.WithName("setup")
)

func init() {
	utilruntime.Must(clientgoscheme.AddToScheme(scheme))
	utilruntime.Must(tektonv1.AddToScheme(scheme))
	utilruntime.Must(konfluxciv1alpha1.AddToScheme(scheme))
	//+kubebuilder:scaffold:scheme
}

func main() {
	var metricsAddr string
	var enableLeaderElection bool
	var enableLeaderElectionId string
	var probeAddr string
	var remoteClusterConfigFile string
	flag.StringVar(&metricsAddr, "metrics-bind-address", ":8080", "The address the metric endpoint binds to.")
	flag.StringVar(&probeAddr, "health-probe-bind-address", ":8081", "The address the probe endpoint binds to.")
	flag.BoolVar(&enableLeaderElection, "leader-elect", false,
		"Enable leader election for controller manager. "+
			"Enabling this will ensure there is only one active controller manager.")
	flag.StringVar(&enableLeaderElectionId, "leader-election-id",
		"b548bb9d.redhat.com",
		"Enable overriding leader election ID for controller manager. ")
	flag.StringVar(&remoteClusterConfigFile, "remote-cluster-config-file", "",
		"The remote client will load its initial configuration from this file. "+
			"Omit this flag to use the default configuration values. "+
			"Command-line flags override configuration from this file.")
	opts := zap.Options{
		Development: true,
		TimeEncoder: zapcore.ISO8601TimeEncoder,
	}
	opts.BindFlags(flag.CommandLine)
	flag.Parse()

	ctrl.SetLogger(zap.New(zap.UseFlagOptions(&opts)))

	mgr, err := ctrl.NewManager(ctrl.GetConfigOrDie(), ctrl.Options{
		Scheme: scheme,
		Metrics: server.Options{
			BindAddress: metricsAddr,
		},
		WebhookServer: webhook.NewServer(webhook.Options{
			Port: 9443,
		}),
		HealthProbeBindAddress: probeAddr,
		LeaderElection:         enableLeaderElection,
		LeaderElectionID:       enableLeaderElectionId,
		Cache: cache.Options{
			ByObject: map[client.Object]cache.ByObject{
				&tektonv1.PipelineRun{}: {
					Label: labels.SelectorFromSet(labels.Set{metadata.PipelinesTypeLabel: tekton.PipelineTypeRelease}),
				},
			},
		},
	})
	if err != nil {
		setupLog.Error(err, "unable to start manager")
		os.Exit(1)
	}

	remoteCluster, err := setupRemoteCluster(mgr, remoteClusterConfigFile)
	if err != nil {
		setupLog.Error(err, "unable to register remote cluster")
		os.Exit(1)
	}

	setupControllers(mgr, remoteCluster)

	//+kubebuilder:scaffold:builder

	addHealthAndReadyChecks(mgr)

	setupLog.Info("starting manager")
	if err := mgr.Start(ctrl.SetupSignalHandler()); err != nil {
		setupLog.Error(err, "problem running manager")
		os.Exit(1)
	}
}

// addHealthAndReadyChecks adds health and ready checks to the manager.
func addHealthAndReadyChecks(mgr manager.Manager) {
	if err := mgr.AddHealthzCheck("healthz", healthz.Ping); err != nil {
		setupLog.Error(err, "unable to set up health check")
		os.Exit(1)
	}
	if err := mgr.AddReadyzCheck("readyz", healthz.Ping); err != nil {
		setupLog.Error(err, "unable to set up ready check")
		os.Exit(1)
	}
}

// getRemoteClusterClientConfig reads a config file with a remote cluster configuration and returns
// the pointer to a rest.Config object.
func getRemoteClusterClientConfig(configFile string) (*rest.Config, error) {
	/* #nosec G304 */
	data, err := os.ReadFile(configFile)
	if err != nil {
		return nil, err
	}

	clientConfig, err := clientcmd.NewClientConfigFromBytes(data)
	if err != nil {
		return nil, err
	}

	return clientConfig.ClientConfig()
}

// setupControllers setups all the operator controllers.
func setupControllers(mgr manager.Manager, remoteCluster cluster.Cluster) {
	err := controller.SetupControllers(mgr, remoteCluster, controllers.EnabledControllers...)
	if err != nil {
		setupLog.Error(err, "unable to setup controllers")
		os.Exit(1)
	}
}

// setupRemoteCluster register a remote cluster in the given manager.
func setupRemoteCluster(mgr manager.Manager, configFile string) (cluster.Cluster, error) {
	config, err := getRemoteClusterClientConfig(configFile)
	if err != nil {
		return nil, err
	}

	remoteCluster, err := cluster.New(config, func(options *cluster.Options) {
		options.Scheme = scheme
	})

	if err != nil {
		return nil, err
	}

	return remoteCluster, mgr.Add(remoteCluster)
}

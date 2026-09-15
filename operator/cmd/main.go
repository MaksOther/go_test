package main

import (
	"flag"
	"os"

	"k8s.io/apimachinery/pkg/runtime"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
	metricsserver "sigs.k8s.io/controller-runtime/pkg/metrics/server"

	"github.com/MaksOther/go_test/operator/api/v1alpha1"
	"github.com/MaksOther/go_test/operator/internal/controller"
	"github.com/MaksOther/go_test/operator/internal/provisioner"
)

func main() {
	var provisionerURL string
	flag.StringVar(&provisionerURL, "provisioner-url", "http://localhost:8080", "base URL of the provisioning API")
	zapOptions := zap.Options{Development: true}
	zapOptions.BindFlags(flag.CommandLine)
	flag.Parse()

	ctrl.SetLogger(zap.New(zap.UseFlagOptions(&zapOptions)))
	setupLog := ctrl.Log.WithName("setup")

	scheme := runtime.NewScheme()
	utilruntime.Must(clientgoscheme.AddToScheme(scheme))
	utilruntime.Must(v1alpha1.AddToScheme(scheme))

	mgr, err := ctrl.NewManager(ctrl.GetConfigOrDie(), ctrl.Options{
		Scheme:  scheme,
		Metrics: metricsserver.Options{BindAddress: "0"},
	})
	if err != nil {
		setupLog.Error(err, "unable to create manager")
		os.Exit(1)
	}

	reconciler := &controller.ManagedDatabaseReconciler{
		Client:      mgr.GetClient(),
		Provisioner: provisioner.NewClient(provisionerURL),
	}
	if err := reconciler.SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to set up controller")
		os.Exit(1)
	}

	setupLog.Info("starting manager", "provisionerURL", provisionerURL)
	if err := mgr.Start(ctrl.SetupSignalHandler()); err != nil {
		setupLog.Error(err, "manager stopped with error")
		os.Exit(1)
	}
}

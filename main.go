/*
Copyright 2021.

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
	"fmt"
	"os"

	"github.com/go-logr/zapr"
	"github.com/win5do/go-lib/logx"
	"k8s.io/apimachinery/pkg/types"

	// Import all Kubernetes client auth plugins (e.g. Azure, GCP, OIDC, etc.)
	// to ensure that exec-entrypoint and run can make use of them.
	_ "k8s.io/client-go/plugin/pkg/client/auth"

	"github.com/open-policy-agent/cert-controller/pkg/rotator"

	"k8s.io/apimachinery/pkg/runtime"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/healthz"
	czap "sigs.k8s.io/controller-runtime/pkg/log/zap"

	dbv1 "github.com/win5do/etcd-operator/api/v1"
	"github.com/win5do/etcd-operator/controllers"
	"github.com/win5do/etcd-operator/pkg/k8s"
	// +kubebuilder:scaffold:imports
)

var (
	scheme   = runtime.NewScheme()
	setupLog = ctrl.Log.WithName("setup")

	// DNSName is <service name>.<namespace>.svc
	dnsName  = fmt.Sprintf("%s.%s.svc", serviceName, k8s.GetOperatorNamespace())
	webhooks = []rotator.WebhookInfo{
		{
			Name: "etcd-operator-validating-webhook-config",
			Type: rotator.Validating,
		},
		{
			Name: "etcd-operator-mutating-webhook-config",
			Type: rotator.Mutating,
		},
	}
)

const (
	secretName     = "etcd-operator-webhook-cert"
	serviceName    = "etcd-operator-webhook"
	caName         = "etcd-operator-ca"
	caOrganization = "etcd-operator"
)

func init() {
	utilruntime.Must(clientgoscheme.AddToScheme(scheme))

	utilruntime.Must(dbv1.AddToScheme(scheme))
	// +kubebuilder:scaffold:scheme
}

func main() {
	var metricsAddr string
	var enableLeaderElection bool
	var probeAddr string
	var certDir string
	var disableCertRotation bool

	flag.StringVar(&metricsAddr, "metrics-bind-address", ":8080", "The address the metric endpoint binds to.")
	flag.StringVar(&probeAddr, "health-probe-bind-address", ":8081", "The address the probe endpoint binds to.")
	flag.BoolVar(&enableLeaderElection, "leader-elect", false,
		"Enable leader election for controller manager. "+
			"Enabling this will ensure there is only one active controller manager.")
	flag.StringVar(&certDir, "cert-dir", "/certs", "The directory where certs are stored, defaults to /certs")
	flag.BoolVar(&disableCertRotation, "disable-cert-rotation", false, "disable automatic generation and rotation of webhook TLS certificates/keys")

	opts := czap.Options{
		Development: true,
	}
	opts.BindFlags(flag.CommandLine)
	flag.Parse()

	zaplog := czap.NewRaw(czap.UseFlagOptions(&opts))
	ctrl.SetLogger(zapr.NewLogger(zaplog))
	logx.SetLogger(zaplog)

	mgr, err := ctrl.NewManager(ctrl.GetConfigOrDie(), ctrl.Options{
		Scheme:                 scheme,
		MetricsBindAddress:     metricsAddr,
		Port:                   9443,
		CertDir:                certDir,
		HealthProbeBindAddress: probeAddr,
		LeaderElection:         enableLeaderElection,
		LeaderElectionID:       "a73bd0c8.gogo.io",
	})
	if err != nil {
		setupLog.Error(err, "unable to start manager")
		os.Exit(1)
	}

	// Make sure certs are generated and valid if cert rotation is enabled.
	setupFinished := make(chan struct{})
	if !disableCertRotation {
		setupLog.Info("setting up cert rotation")
		err := rotator.AddRotator(mgr, &rotator.CertRotator{
			SecretKey: types.NamespacedName{
				Namespace: k8s.GetOperatorNamespace(),
				Name:      secretName,
			},
			CertDir:        certDir,
			CAName:         caName,
			CAOrganization: caOrganization,
			DNSName:        dnsName,
			IsReady:        setupFinished,
			Webhooks:       webhooks,
		})
		if err != nil {
			setupLog.Error(err, "unable to set up cert rotation")
			os.Exit(1)
		}
	} else {
		close(setupFinished)
	}

	go func() {
		<-setupFinished

		err := (&controllers.EtcdReconciler{
			Client: mgr.GetClient(),
			Log:    zaplog.Sugar().Named("controllers").Named("Etcd"),
			Scheme: mgr.GetScheme(),
		}).SetupWithManager(mgr)
		if err != nil {
			setupLog.Error(err, "unable to create controller", "controller", "Etcd")
			os.Exit(1)
		}

		if certDir != "" {
			err = (&dbv1.Etcd{}).SetupWebhookWithManager(mgr)
			if err != nil {
				setupLog.Error(err, "unable to SetupWebhookWithManager")
				os.Exit(1)
			}
		}
	}()

	// +kubebuilder:scaffold:builder

	if err := mgr.AddHealthzCheck("health", healthz.Ping); err != nil {
		setupLog.Error(err, "unable to set up health check")
		os.Exit(1)
	}
	if err := mgr.AddReadyzCheck("check", healthz.Ping); err != nil {
		setupLog.Error(err, "unable to set up ready check")
		os.Exit(1)
	}

	setupLog.Info("starting manager")
	if err := mgr.Start(ctrl.SetupSignalHandler()); err != nil {
		setupLog.Error(err, "problem running manager")
		os.Exit(1)
	}
}

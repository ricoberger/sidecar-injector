package main

import (
	goflag "flag"
	"fmt"
	"net/http"
	"os"

	"github.com/ricoberger/sidecar-injector/pkg/sidecar"
	"github.com/ricoberger/sidecar-injector/pkg/version"

	flag "github.com/spf13/pflag"
	_ "k8s.io/client-go/plugin/pkg/client/auth/gcp"
	"sigs.k8s.io/controller-runtime/pkg/client/config"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/manager/signals"
	"sigs.k8s.io/controller-runtime/pkg/webhook"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
)

var (
	certDir     string
	configFile  string
	showVersion bool
	log         = logf.Log.WithName("webhook")
)

// init is used to define all flags for external-authz.
func init() {
	defaultCertDir := "certs"
	if os.Getenv("WEBHOOK_CERTS") != "" {
		defaultCertDir = os.Getenv("WEBHOOK_CERTS")
	}

	defaultConfigFile := "config.yaml"
	if os.Getenv("WEBHOOK_CONFIG") != "" {
		defaultConfigFile = os.Getenv("WEBHOOK_CONFIG")
	}

	flag.StringVar(&certDir, "certs", defaultCertDir, "Folder containing the x509 certificate and key file.")
	flag.StringVar(&configFile, "config", defaultConfigFile, "Name of the configuration file.")
	flag.BoolVar(&showVersion, "version", false, "Print version information.")
}

func main() {
	opts := zap.Options{}
	opts.BindFlags(goflag.CommandLine)

	flag.CommandLine.AddGoFlagSet(goflag.CommandLine)
	flag.Parse()

	logger := zap.New(zap.UseFlagOptions(&opts))
	logf.SetLogger(logger)

	// When the version value is set to "true" (--version) we will print the version information for external-authz.
	// After we printed the version information the service is stopped.
	// The short form of the version information is also printed in two lines, when the version option is set to
	// "false".
	if showVersion {
		v, err := version.Print("sidecar-injector")
		if err != nil {
			log.Error(err, "Failed to print version information")
			os.Exit(1)
		}

		fmt.Fprintln(os.Stdout, v)
		return
	}

	log.Info("Version information", version.Info()...)
	log.Info("Build context", version.BuildContext()...)

	c, err := sidecar.LoadConfig(configFile)
	if err != nil {
		log.Error(err, "Could not load configuration file.")
		os.Exit(1)
	}

	// Setup a Manager
	log.Info("Settings up manager.")
	mgr, err := manager.New(config.GetConfigOrDie(), manager.Options{
		Port:                   8443,
		CertDir:                certDir,
		MetricsBindAddress:     ":8081",
		HealthProbeBindAddress: ":8080",
		ReadinessEndpointName:  "/readyz",
		LivenessEndpointName:   "/healthz",
	})
	if err != nil {
		log.Error(err, "Unable to set up overall controller manager.")
		os.Exit(1)
	}

	mgr.AddReadyzCheck("readyz", func(req *http.Request) error {
		return nil
	})
	mgr.AddHealthzCheck("healthz", func(req *http.Request) error {
		return nil
	})

	// Setup Webhooks
	log.Info("Setting up webhook server.")
	hookServer := mgr.GetWebhookServer()

	log.Info("Registering webhooks to the webhook server.")
	hookServer.Register("/mutate", &webhook.Admission{
		Handler: &sidecar.Injector{
			Client:  mgr.GetClient(),
			Config:  c,
			Decoder: admission.NewDecoder(mgr.GetScheme()),
		},
	})

	log.Info("Starting manager.")
	if err := mgr.Start(signals.SetupSignalHandler()); err != nil {
		log.Error(err, "Unable to run manager.")
		os.Exit(1)
	}
}

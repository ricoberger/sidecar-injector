package main

import (
	"fmt"
	"net/http"
	"os"

	"github.com/ricoberger/sidecar-injector/pkg/log"
	"github.com/ricoberger/sidecar-injector/pkg/sidecar"
	"github.com/ricoberger/sidecar-injector/pkg/version"

	flag "github.com/spf13/pflag"
	"go.uber.org/zap"
	_ "k8s.io/client-go/plugin/pkg/client/auth/gcp"
	"sigs.k8s.io/controller-runtime/pkg/client/config"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/manager/signals"
	"sigs.k8s.io/controller-runtime/pkg/webhook"
)

var (
	certDir     string
	configFile  string
	logFormat   string
	logLevel    string
	showVersion bool
)

// init is used to define all flags for external-authz.
func init() {
	defaultCertDir := "cert"
	if os.Getenv("WEBHOOK_CERTS") != "" {
		defaultCertDir = os.Getenv("WEBHOOK_CERTS")
	}

	defaultConfigFile := "config.yaml"
	if os.Getenv("WEBHOOK_CONFIG") != "" {
		defaultConfigFile = os.Getenv("WEBHOOK_CONFIG")
	}

	defaultLogFormat := "console"
	if os.Getenv("WEBHOOK_LOG_FORMAT") != "" {
		defaultLogFormat = os.Getenv("WEBHOOK_LOG_FORMAT")
	}

	defaultLogLevel := "info"
	if os.Getenv("WEBHOOK_LOG_LEVEL") != "" {
		defaultLogLevel = os.Getenv("WEBHOOK_LOG_LEVEL")
	}

	flag.StringVar(&certDir, "certs", defaultCertDir, "Folder containing the x509 certificate and key file.")
	flag.StringVar(&configFile, "config", defaultConfigFile, "Name of the configuration file.")
	flag.StringVar(&logFormat, "log.format", defaultLogFormat, "Set the output format of the logs. Must be \"console\" or \"json\".")
	flag.StringVar(&logLevel, "log.level", defaultLogLevel, "Set the log level. Must be \"debug\", \"info\", \"warn\", \"error\", \"fatal\" or \"panic\".")
	flag.BoolVar(&showVersion, "version", false, "Print version information.")
}

func main() {
	flag.Parse()
	log.Setup(logLevel, logFormat)

	// When the version value is set to "true" (--version) we will print the version information for external-authz.
	// After we printed the version information the service is stopped.
	// The short form of the version information is also printed in two lines, when the version option is set to
	// "false".
	if showVersion {
		v, err := version.Print("sidecar-injector")
		if err != nil {
			log.Fatal("Failed to print version information", zap.Error(err))
		}

		fmt.Fprintln(os.Stdout, v)
		return
	}

	log.Info("Version information", version.Info()...)
	log.Info("Build context", version.BuildContext()...)

	c, err := sidecar.LoadConfig(configFile)
	if err != nil {
		log.Fatal("Could not load configuration file.", zap.Error(err))
	}

	// Setup a Manager
	log.Info("Setting up manager.")
	mgr, err := manager.New(config.GetConfigOrDie(), manager.Options{
		Port:                   8443,
		CertDir:                certDir,
		MetricsBindAddress:     ":8081",
		HealthProbeBindAddress: ":8080",
		ReadinessEndpointName:  "/readyz",
		LivenessEndpointName:   "/healthz",
	})
	if err != nil {
		log.Fatal("Unable to set up overall controller manager.", zap.Error(err))
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
			Client: mgr.GetClient(),
			Config: c,
		},
	})

	log.Info("Starting manager.")
	if err := mgr.Start(signals.SetupSignalHandler()); err != nil {
		log.Fatal("Unable to run manager.", zap.Error(err))
	}
}

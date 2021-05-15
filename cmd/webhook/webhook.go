package main

import (
	"fmt"
	"net/http"
	"os"

	"github.com/ricoberger/sidecar-injector/pkg/sidecar"
	"github.com/ricoberger/sidecar-injector/pkg/version"

	"github.com/sirupsen/logrus"
	flag "github.com/spf13/pflag"
	_ "k8s.io/client-go/plugin/pkg/client/auth/gcp"
	"sigs.k8s.io/controller-runtime/pkg/client/config"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/manager/signals"
	"sigs.k8s.io/controller-runtime/pkg/webhook"
)

var (
	log         = logrus.WithFields(logrus.Fields{"package": "webhook"})
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

	defaultLogFormat := "plain"
	if os.Getenv("WEBHOOK_LOG_FORMAT") != "" {
		defaultLogFormat = os.Getenv("WEBHOOK_LOG_FORMAT")
	}

	defaultLogLevel := "info"
	if os.Getenv("WEBHOOK_LOG_LEVEL") != "" {
		defaultLogLevel = os.Getenv("WEBHOOK_LOG_LEVEL")
	}

	flag.StringVar(&certDir, "certs", defaultCertDir, "Folder containing the x509 certificate and key file.")
	flag.StringVar(&configFile, "config", defaultConfigFile, "Name of the configuration file.")
	flag.StringVar(&logFormat, "log.format", defaultLogFormat, "Set the output format of the logs. Must be \"plain\" or \"json\".")
	flag.StringVar(&logLevel, "log.level", defaultLogLevel, "Set the log level. Must be \"trace\", \"debug\", \"info\", \"warn\", \"error\", \"fatal\" or \"panic\".")
	flag.BoolVar(&showVersion, "version", false, "Print version information.")
}

func main() {
	flag.Parse()

	// Configure our logging library. The logs can be written in plain format (the plain format is compatible with
	// logfmt) or in json format. The default is plain, because it is better to read during development. In a production
	// environment you should consider to use json, so that the logs can be parsed by a logging system like
	// Elasticsearch.
	// Next to the log format it is also possible to configure the log level. The accepted values are "trace", "debug",
	// "info", "warn", "error", "fatal" and "panic". The default log level is "info". When the log level is set to
	// "trace" or "debug" we will also print the caller in the logs.
	if logFormat == "json" {
		logrus.SetFormatter(&logrus.JSONFormatter{})
	} else {
		logrus.SetFormatter(&logrus.TextFormatter{
			FullTimestamp: true,
		})
	}

	lvl, err := logrus.ParseLevel(logLevel)
	if err != nil {
		log.WithError(err).WithFields(logrus.Fields{"log.level": logLevel}).Fatal("Could not set log level")
	}
	logrus.SetLevel(lvl)

	if lvl == logrus.TraceLevel || lvl == logrus.DebugLevel {
		logrus.SetReportCaller(true)
	}

	// When the version value is set to "true" (--version) we will print the version information for external-authz.
	// After we printed the version information the service is stopped.
	// The short form of the version information is also printed in two lines, when the version option is set to
	// "false".
	if showVersion {
		v, err := version.Print("sidecar-injector")
		if err != nil {
			log.WithError(err).Fatalf("Failed to print version information")
		}

		fmt.Fprintln(os.Stdout, v)
		return
	}

	log.WithFields(version.Info()).Infof("Version information")
	log.WithFields(version.BuildContext()).Infof("Build context")

	c, err := sidecar.LoadConfig(configFile)
	if err != nil {
		log.WithError(err).Fatalf("Could not load configuration file.")
	}

	// Setup a Manager
	log.Infof("Setting up manager.")
	mgr, err := manager.New(config.GetConfigOrDie(), manager.Options{
		Port:                   8443,
		CertDir:                certDir,
		MetricsBindAddress:     ":8081",
		HealthProbeBindAddress: ":8080",
		ReadinessEndpointName:  "/readyz",
		LivenessEndpointName:   "/healthz",
	})
	if err != nil {
		log.WithError(err).Fatalf("Unable to set up overall controller manager.")
	}

	mgr.AddReadyzCheck("readyz", func(req *http.Request) error {
		return nil
	})
	mgr.AddHealthzCheck("healthz", func(req *http.Request) error {
		return nil
	})

	// Setup Webhooks
	log.Infof("Setting up webhook server.")
	hookServer := mgr.GetWebhookServer()

	log.Infof("Registering webhooks to the webhook server.")
	hookServer.Register("/mutate", &webhook.Admission{
		Handler: &sidecar.Injector{
			Client: mgr.GetClient(),
			Config: c,
		},
	})

	log.Infof("Starting manager.")
	if err := mgr.Start(signals.SetupSignalHandler()); err != nil {
		log.WithError(err).Fatalf("Unable to run manager.")
	}
}

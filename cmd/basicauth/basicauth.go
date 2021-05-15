package main

import (
	"encoding/base64"
	"fmt"
	"net/http"
	"os"
	"strings"

	"github.com/ricoberger/sidecar-injector/pkg/version"

	"github.com/sirupsen/logrus"
	flag "github.com/spf13/pflag"
)

var (
	log               = logrus.WithFields(logrus.Fields{"package": "main"})
	address           string
	basicAuthPassword string
	basicAuthUsername string
	logFormat         string
	logLevel          string
	showVersion       bool
)

// init is used to define all flags for external-authz.
func init() {
	defaultAddress := ":4180"
	if os.Getenv("BASIC_AUTH_ADDRESS") != "" {
		defaultAddress = os.Getenv("BASIC_AUTH_ADDRESS")
	}

	defaultLogFormat := "plain"
	if os.Getenv("BASIC_AUTH_LOG_FORMAT") != "" {
		defaultLogFormat = os.Getenv("BASIC_AUTH_LOG_FORMAT")
	}

	defaultLogLevel := "info"
	if os.Getenv("BASIC_AUTH_LOG_LEVEL") != "" {
		defaultLogLevel = os.Getenv("BASIC_AUTH_LOG_LEVEL")
	}

	basicAuthPassword = os.Getenv("BASIC_AUTH_PASSWORD")
	basicAuthUsername = os.Getenv("BASIC_AUTH_USERNAME")

	flag.StringVar(&address, "address", defaultAddress, "The address, where the server is listen on.")
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
		v, err := version.Print("basic-auth")
		if err != nil {
			log.WithError(err).Fatalf("Failed to print version information")
		}

		fmt.Fprintln(os.Stdout, v)
		return
	}

	log.WithFields(version.Info()).Infof("Version information")
	log.WithFields(version.BuildContext()).Infof("Build context")

	// Create and start the http server. The server has just two routes, one which can be used for the Kubernetes health
	// check and another one to handle verify credentials for basic authentication.
	router := http.NewServeMux()
	router.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	router.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		log.WithFields(logrus.Fields{"host": r.Host, "address": r.RemoteAddr, "method": r.Method, "requestURI": r.RequestURI, "proto": r.Proto, "useragent": r.UserAgent()}).Infof("Received request.")

		auth := strings.SplitN(r.Header.Get("Authorization"), " ", 2)
		if len(auth) != 2 || auth[0] != "Basic" {
			http.Error(w, http.StatusText(http.StatusUnauthorized), http.StatusUnauthorized)
			return
		}

		payload, err := base64.StdEncoding.DecodeString(auth[1])
		if err != nil {
			http.Error(w, http.StatusText(http.StatusUnauthorized), http.StatusUnauthorized)
			return
		}

		pair := strings.SplitN(string(payload), ":", 2)
		if len(pair) != 2 || pair[0] != basicAuthUsername || pair[1] != basicAuthPassword {
			http.Error(w, http.StatusText(http.StatusUnauthorized), http.StatusUnauthorized)
			return
		}

		w.WriteHeader(http.StatusOK)
	})

	server := &http.Server{
		Addr:    address,
		Handler: router,
	}

	if err := server.ListenAndServe(); err != http.ErrServerClosed {
		log.WithError(err).Fatalf("Server died unexpected.")
	}
}

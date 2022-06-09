package main

import (
	"encoding/base64"
	"fmt"
	"net/http"
	"os"
	"strings"

	"github.com/ricoberger/sidecar-injector/pkg/log"
	"github.com/ricoberger/sidecar-injector/pkg/version"

	flag "github.com/spf13/pflag"
	"go.uber.org/zap"
)

var (
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

	defaultLogFormat := "console"
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
		v, err := version.Print("basic-auth")
		if err != nil {
			log.Fatal("Failed to print version information", zap.Error(err))
		}

		fmt.Fprintln(os.Stdout, v)
		return
	}

	log.Info("Version information", version.Info()...)
	log.Info("Build context", version.BuildContext()...)

	// Create and start the http server. The server has just two routes, one which can be used for the Kubernetes health
	// check and another one to handle verify credentials for basic authentication.
	router := http.NewServeMux()
	router.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	router.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		log.Info("Received request", zap.String("host", r.Host), zap.String("address", r.RemoteAddr), zap.String("method", r.Method), zap.String("requestURI", r.RequestURI), zap.String("proto", r.Proto), zap.String("useragent", r.UserAgent()))

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
		log.Fatal("Server died unexpected.", zap.Error(err))
	}
}

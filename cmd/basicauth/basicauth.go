package main

import (
	"encoding/base64"
	goflag "flag"
	"fmt"
	"net/http"
	"os"
	"strings"

	"github.com/ricoberger/sidecar-injector/pkg/version"

	flag "github.com/spf13/pflag"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
)

var (
	address           string
	basicAuthPassword string
	basicAuthUsername string
	basicAuthRealm    string
	showVersion       bool
	log               = logf.Log.WithName("basicauth")
)

// init is used to define all flags for external-authz.
func init() {
	defaultAddress := ":4180"
	if os.Getenv("BASIC_AUTH_ADDRESS") != "" {
		defaultAddress = os.Getenv("BASIC_AUTH_ADDRESS")
	}

	defaultRealm := "Restricted Access"
	if os.Getenv("BASIC_AUTH_REALM") != "" {
		defaultRealm = os.Getenv("BASIC_AUTH_REALM")
	}

	basicAuthPassword = os.Getenv("BASIC_AUTH_PASSWORD")
	basicAuthUsername = os.Getenv("BASIC_AUTH_USERNAME")

	flag.StringVar(&address, "address", defaultAddress, "The address, where the server is listen on.")
	flag.StringVar(&basicAuthRealm, "realm", defaultRealm, "The realm for the basic authentication.")
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
		v, err := version.Print("basic-auth")
		if err != nil {
			log.Error(err, "Failed to print version information")
			os.Exit(1)
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
		log.Info("Received request", "host", r.Host, "address", r.RemoteAddr, "method", r.Method, "requestURI", r.RequestURI, "proto", r.Proto, "useragent", r.UserAgent())

		auth := strings.SplitN(r.Header.Get("Authorization"), " ", 2)
		if len(auth) != 2 || auth[0] != "Basic" {
			handleFailedAuth(w)
			return
		}

		payload, err := base64.StdEncoding.DecodeString(auth[1])
		if err != nil {
			handleFailedAuth(w)
			return
		}

		pair := strings.SplitN(string(payload), ":", 2)
		if len(pair) != 2 || pair[0] != basicAuthUsername || pair[1] != basicAuthPassword {
			handleFailedAuth(w)
			return
		}

		w.WriteHeader(http.StatusOK)
	})

	server := &http.Server{
		Addr:    address,
		Handler: router,
	}

	if err := server.ListenAndServe(); err != http.ErrServerClosed {
		log.Error(err, "Server died unexpected.")
		os.Exit(1)
	}
}

func handleFailedAuth(w http.ResponseWriter) {
	w.Header().Set("WWW-Authenticate", fmt.Sprintf(`Basic realm="%s", charset="UTF-8"`, basicAuthRealm))
	http.Error(w, http.StatusText(http.StatusUnauthorized), http.StatusUnauthorized)
}

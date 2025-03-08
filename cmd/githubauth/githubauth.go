package main

import (
	"encoding/base64"
	goflag "flag"
	"fmt"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/ricoberger/sidecar-injector/pkg/version"

	"github.com/google/go-github/v65/github"
	"github.com/hashicorp/golang-lru/v2/expirable"
	flag "github.com/spf13/pflag"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
)

var (
	address        string
	basicAuthRealm string
	organization   string
	cacheDuration  int
	showVersion    bool
	log            = logf.Log.WithName("githubauth")
)

// init is used to define all flags for external-authz.
func init() {
	defaultAddress := ":4180"
	if os.Getenv("GITHUB_AUTH_ADDRESS") != "" {
		defaultAddress = os.Getenv("GITHUB_AUTH_ADDRESS")
	}

	defaultRealm := "Restricted Access"
	if os.Getenv("GITHUB_AUTH_REALM") != "" {
		defaultRealm = os.Getenv("GITHUB_AUTH_REALM")
	}

	defaultOrganization := ""
	if os.Getenv("GITHUB_ORGANIZATION") != "" {
		defaultOrganization = os.Getenv("GITHUB_ORGANIZATION")
	}

	defaultCacheDuration := 86400
	if os.Getenv("GITHUB_CACHE_DURATION") != "" {
		defaultCacheDurationTmp := os.Getenv("GITHUB_CACHE_DURATION")
		parsedCacheDuration, err := strconv.Atoi(defaultCacheDurationTmp)
		if err == nil {
			defaultCacheDuration = parsedCacheDuration
		}
	}

	flag.StringVar(&address, "address", defaultAddress, "The address, where the server is listen on.")
	flag.StringVar(&basicAuthRealm, "realm", defaultRealm, "The realm for the basic authentication.")
	flag.StringVar(&organization, "organization", defaultOrganization, "The GitHub organization to check if the user is a member of.")
	flag.IntVar(&cacheDuration, "cache-time", defaultCacheDuration, "The time to cache authenticated users in seconds. This is used to reduce the number of requests to GitHub.")
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
		v, err := version.Print("githubauth")
		if err != nil {
			log.Error(err, "Failed to print version information")
			os.Exit(1)
		}

		fmt.Fprintln(os.Stdout, v)
		return
	}

	log.Info("Version information", version.Info()...)
	log.Info("Build context", version.BuildContext()...)

	log.Info("Create cache", "seconds", cacheDuration)
	cache := expirable.NewLRU[string, string](1000, nil, time.Second*time.Duration(cacheDuration))

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
		if len(pair) != 2 {
			handleFailedAuth(w)
			return
		}

		if val, ok := cache.Get(strings.ToLower(pair[0])); ok {
			if val == pair[1] {
				log.Info("User is already authenticated", "username", pair[0])
				w.WriteHeader(http.StatusOK)
				return
			}
		}

		client := github.NewClient(nil).WithAuthToken(pair[1])
		user, _, err := client.Users.Get(r.Context(), "")
		if err != nil {
			log.Error(err, "Failed to get user information")
			handleFailedAuth(w)
			return
		}

		if !strings.EqualFold(user.GetLogin(), pair[0]) {
			log.Error(fmt.Errorf("login does not match"), "Provided username does not match GitHub login", "organization", organization, "username", pair[0], "login", user.GetLogin())
			handleFailedAuth(w)
			return
		}

		isMember, _, err := client.Organizations.IsMember(r.Context(), organization, user.GetLogin())
		if err != nil {
			log.Error(err, "Failed to check if user is a member of the organization", "organization", organization, "login", user.GetLogin())
			handleFailedAuth(w)
			return
		}

		if !isMember {
			log.Error(fmt.Errorf("isMember is false"), "User is not a member of the organization", "organization", organization, "login", user.GetLogin())
			handleFailedAuth(w)
			return
		}

		cache.Add(strings.ToLower(pair[0]), pair[1])
		w.WriteHeader(http.StatusOK)
	})

	server := &http.Server{
		Addr:              address,
		Handler:           router,
		ReadHeaderTimeout: 5 * time.Second,
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

package main

import (
	"net"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/sirupsen/logrus"
	"hedisam/kubemock/api"
)

func main() {
	logger := logrus.New()

	if strings.ToLower(os.Getenv("VERBOSE")) == "enabled" {
		logger.SetLevel(logrus.DebugLevel)
	}

	kubeHandler := api.NewKubeHandler(logger)
	mux := http.NewServeMux()
	// this is an actual kube endpoint that is called by HC Vault
	mux.Handle("/apis/authentication.k8s.io/v1/tokenreviews", http.HandlerFunc(kubeHandler.LoginHandler))
	// this is a custom endpoint that will be called directly by our unit tests to register a fake service account
	// and generate a valid jwt token for it so that the jwt can later be validated by Vault via the login endpoint above.
	mux.Handle("/api/v1/testing/serviceaccounts", http.HandlerFunc(kubeHandler.RegisterServiceAccountHandler))
	mux.Handle("/api/v1/testing/health", http.HandlerFunc(kubeHandler.HealthHandler))
	// reset endpoint to clean up service account registry before running a test
	mux.Handle("/api/v1/testing/reset", http.HandlerFunc(kubeHandler.ResetHandler))
	// handle the root endpoint for any unexpected request
	mux.Handle("/", http.HandlerFunc(kubeHandler.UnimplementedHandler))

	netAddr := "0.0.0.0:6443"
	ln, err := net.Listen("tcp", netAddr)
	if err != nil {
		logger.WithField("net_addr", netAddr).WithError(err).Fatal("Could not start tcp listener")
	}
	addr := "http://" + netAddr
	s := http.Server{
		Handler:     mux,
		ReadTimeout: 5 * time.Second,
		Addr:        addr,
	}
	defer func() {
		err := s.Close()
		if err != nil {
			logger.WithError(err).Error("Failed to close kube http server")
		}
	}()

	logger.WithField("addr", s.Addr).Info("Starting kube http server")
	err = s.Serve(ln)
	if err != nil {
		logger.WithError(err).Fatal("Kube http server closed with unexpected error")
	}

	logger.Info("Kube server stopped")
}

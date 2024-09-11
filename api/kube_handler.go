package api

import (
	"crypto/rand"
	"crypto/rsa"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/golang-jwt/jwt/v5"
	"github.com/sirupsen/logrus"
)

type serviceAccountInfo struct {
	UID       string `json:"uid"`
	Name      string `json:"name"`
	Namespace string `json:"namespace"`
}

type loginRequest struct {
	Spec struct {
		Token string `json:"token"`
	} `json:"spec"`
}

// KubeHandler handles kube auth requests.
type KubeHandler struct {
	logger                  *logrus.Logger
	jwtToServiceAccountInfo map[string]serviceAccountInfo
}

// NewKubeHandler creates and returns a new kube handler.
func NewKubeHandler(logger *logrus.Logger) *KubeHandler {
	return &KubeHandler{
		logger:                  logger,
		jwtToServiceAccountInfo: make(map[string]serviceAccountInfo),
	}
}

// RootHandler handles any unimplemented request.
func (s *KubeHandler) RootHandler(w http.ResponseWriter, r *http.Request) {
	s.logger.WithField("request_url", r.URL).Debug("Kube auth server received unknown request")
	s.writeResponse(w, http.StatusNotImplemented, map[string]any{
		"success": false,
		"error":   fmt.Sprintf("unknown request: %s", r.URL),
	})
}

// HealthHandler is a health endpoint that can be called by unit tests to make sure the server is functioning.
func (s *KubeHandler) HealthHandler(w http.ResponseWriter, r *http.Request) {
	s.logger.WithField("request_url", r.URL).Debug("Kube auth server received health probe")
	s.writeResponse(w, http.StatusOK, nil)
}

// RegisterServiceAccountHandler handles service account registration requests made directly by unit tests.
func (s *KubeHandler) RegisterServiceAccountHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		s.writeResponse(w, http.StatusNotImplemented, map[string]any{
			"success": false,
			"error":   fmt.Sprintf("service account registration handler expects POST but got %q", r.Method),
		})
		return
	}

	var sa serviceAccountInfo
	err := json.NewDecoder(r.Body).Decode(&sa)
	if err != nil {
		s.logger.WithError(err).Error("Could not decode service account registration request")
		s.writeResponse(w, http.StatusBadRequest, map[string]any{
			"success": false,
			"error":   fmt.Sprintf("invalid service account registration request: %v", err),
		})
		return
	}

	jwtToken, err := generateKubeJWT(sa.Name, sa.Namespace, sa.UID)
	if err != nil {
		s.logger.WithError(err).WithField("service_account", sa).Error("Could not generate jwt token")
		return
	}

	s.jwtToServiceAccountInfo[jwtToken] = sa
	s.writeResponse(w, http.StatusOK, map[string]any{
		"success": true,
		"jwt":     jwtToken,
	})
}

// LoginHandler handles kube auth login requests made by HC Vault possibly with a valid jwt token generated
// by RegisterServiceAccountHandler.
func (s *KubeHandler) LoginHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost && r.Method != http.MethodPut {
		s.writeResponse(w, http.StatusNotImplemented, map[string]any{
			"success": false,
			"error":   fmt.Sprintf("login request must be either PUT or POST but got %q", r.Method),
		})
		return
	}

	var req loginRequest
	err := json.NewDecoder(r.Body).Decode(&req)
	if err != nil {
		s.logger.WithError(err).Error("Could not decode kube login request")
		s.writeResponse(w, http.StatusBadRequest, map[string]any{
			"success": false,
			"error":   fmt.Sprintf("invalid login request: %v", err),
		})
		return
	}

	sa, jwtValid := s.jwtToServiceAccountInfo[req.Spec.Token]
	if !jwtValid {
		s.logger.Debug("Received kube login request with unknown token")
		s.writeResponse(w, http.StatusOK, map[string]any{
			"status": map[string]any{
				"authenticated": false,
			},
		})
		return
	}

	s.writeResponse(w, http.StatusOK, map[string]any{
		"status": map[string]any{
			"authenticated": true,
			"user": map[string]any{
				"username": fmt.Sprintf("system:serviceaccount:%s:%s", sa.Namespace, sa.Name),
				"uid":      sa.UID,
			},
		},
	})

	s.logger.Debug("Successfully handled kube login request")
}

func (s *KubeHandler) writeResponse(w http.ResponseWriter, statusCode int, resp any) {
	w.WriteHeader(statusCode)
	err := json.NewEncoder(w).Encode(resp)
	if err != nil {
		s.logger.WithFields(logrus.Fields{
			"status_code": statusCode,
			"response":    resp,
		}).Error("Could not write response to http writer")
	}
}

// generateKubeJWT generates a valid k8s jwt token that the vault testing instance can accept and validate.
func generateKubeJWT(service, namespace, uid string) (string, error) {
	secretKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return "", fmt.Errorf("generate secret key: %w", err)
	}

	claims := jwt.MapClaims{
		"kubernetes.io/serviceaccount/service-account.uid":  uid,
		"kubernetes.io/serviceaccount/service-account.name": service,
		"kubernetes.io/serviceaccount/namespace":            namespace,
	}

	token := jwt.NewWithClaims(jwt.SigningMethodRS256, claims)
	signedJWT, err := token.SignedString(secretKey)
	if err != nil {
		return "", fmt.Errorf("sign token: %w", err)
	}

	return signedJWT, nil
}

package api

import (
	"fmt"
	"net/http"
	"strings"

	logf "sigs.k8s.io/controller-runtime/pkg/log"
)

func (api *Api) getBearerToken(r *http.Request) (string, error) {
	authHeader := r.Header.Get("Authorization")
	if authHeader == "" {
		return "", fmt.Errorf("missing Authorization header")
	}

	parts := strings.SplitN(authHeader, " ", 2)
	if len(parts) != 2 || strings.ToLower(parts[0]) != "bearer" {
		return "", fmt.Errorf("invalid Authorization header format")
	}

	return parts[1], nil
}

func (api *Api) authenticateRequest(r *http.Request) (*jwt, error) {
	logger := logf.FromContext(r.Context())

	tokenString, err := api.getBearerToken(r)
	if err != nil {
		return nil, err
	}

	token := newJwt(api.jwtConfig)

	err = token.ParseToken(tokenString)
	if err != nil {
		logger.Error(err, "Failed to parse JWT")
		return nil, fmt.Errorf("failed to parse JWT")
	}

	if err := token.IsValid(); err != nil {
		logger.Error(err, "JWT is not valid")
		return nil, fmt.Errorf("invalid token")
	}

	return token, nil
}

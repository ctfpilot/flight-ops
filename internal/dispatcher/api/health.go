package api

import (
	"fmt"
	"log"
	"net/http"

	logf "sigs.k8s.io/controller-runtime/pkg/log"
)

// func (api *Api) setUnhealthy(reason string) {
// 	api.health.Store(false)
// 	log.Println("Service marked as unhealthy: " + reason)
// }

func (api *Api) writeErrorHandler(r *http.Request, err error) {
	logger := logf.FromContext(r.Context())
	logger.Error(err, "Error writing response")
}

func (api *Api) IndexHandler(w http.ResponseWriter, r *http.Request) {
	// Index page, that just lists the name of the service and status
	w.Header().Set("Content-Type", "application/json")

	healthcheck := "ok"
	if !api.healthy() {
		healthcheck = "error"
		w.WriteHeader(http.StatusInternalServerError)
	}
	_, err := fmt.Fprintf(w, "{\"name\":\"CTFd manager\",\"status\":\"%s\"}\n", healthcheck)
	if err != nil {
		api.writeErrorHandler(r, err)
	}
}

func (api *Api) StatusHandler(w http.ResponseWriter, r *http.Request) {
	// Print status in json
	w.Header().Set("Content-Type", "application/json")

	healthcheck := "ok"
	if !api.healthy() {
		healthcheck = "error"
		w.WriteHeader(http.StatusInternalServerError)
	}
	_, err := fmt.Fprintf(w, "{\"status\":\"%s\"}\n", healthcheck)
	if err != nil {
		api.writeErrorHandler(r, err)
	}
}

func (api *Api) VersionHandler(w http.ResponseWriter, r *http.Request) {
	// Print version in json
	w.Header().Set("Content-Type", "application/json")
	_, err := fmt.Fprintf(w, "{\"version\":\"%s\"}\n", api.version)
	if err != nil {
		api.writeErrorHandler(r, err)
	}
}

func (api *Api) healthy() bool {
	// Check if the service is healthy
	if api.checkAccess() != nil {
		return false
	}

	if api.rateLimiter == nil {
		log.Println("Service deemed unhealthy, due to missing rate limiter")
		return false
	}

	if err := api.rateLimiter.Healthy(); err != nil {
		log.Println("Service deemed unhealthy, due to Redis health check failure:", err)
		return false
	}

	if !api.health.Load() {
		log.Println("Service deemed unhealthy, due to unhealthy status")
		return false
	}

	return true
}

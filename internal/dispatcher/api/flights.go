package api

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/ctfpilot/flight-ops/internal/dispatcher/lib"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
)

func (api *Api) flightsErrorHandler(w http.ResponseWriter, r *http.Request, err error) {
	logger := logf.FromContext(r.Context())
	logger.Error(err, "Error getting flight plans")
	http.Error(w, "Error getting flight plans - Please contact an admin for more information", http.StatusInternalServerError)
}

func (api *Api) flightErrorHandler(w http.ResponseWriter, r *http.Request, err error) {
	logger := logf.FromContext(r.Context())
	logger.Error(err, "Error getting flight")
	http.Error(w, "Error getting flight plan - Please contact an admin for more information", http.StatusInternalServerError)
}

func (api *Api) GetFlightsHandler(w http.ResponseWriter, r *http.Request) {
	recorder := newStatusRecorder(w)
	w = recorder
	action := unauthenticatedAction
	start := time.Now()
	defer func() {
		api.metrics.ObserveRequest("/api/flights", action, recorder.Status(), time.Since(start))
	}()

	ctx := r.Context()

	token, err := api.authenticateRequest(r)
	if err != nil {
		http.Error(w, "Unauthorized: "+err.Error(), http.StatusUnauthorized)
		return
	}
	action = token.Flight.Action
	if limitErr, err := api.CheckRateLimit(ctx, token); err != nil {
		http.Error(w, "Service unavailable: rate limiter temporarily unavailable", http.StatusServiceUnavailable)
		return
	} else if limitErr != nil {
		w.Header().Set("Retry-After", strconv.FormatInt(int64(limitErr.RetryAfter.Seconds()), 10))
		http.Error(w, "Too Many Requests", http.StatusTooManyRequests)
		return
	}
	if !strings.EqualFold(token.Flight.Action, JwtActionList) {
		http.Error(w, "Forbidden: You do not have permission to list flights", http.StatusForbidden)
		return
	}
	if token.Flight.Flight != "" && token.Flight.FlightPlan != "" {
		http.Error(w, "Forbidden: You do not have permission to read the flight", http.StatusForbidden)
		return
	}

	dispatcher := lib.NewDispatcher(
		api.serverSecret,
		api.flightDomain,
		api.k8sClient,
		api.namespace,
		token.Flight.Admin,
		token.Flight.Owner,
		token.Flight.Terminal,
		api.maxConcurrentFlights,
	)

	flight, err := dispatcher.GetFlights(ctx, nil)
	if err != nil {
		api.flightsErrorHandler(w, r, err)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	err = json.NewEncoder(w).Encode(flight)
	if err != nil {
		api.flightsErrorHandler(w, r, err)
		return
	}
}

func (api *Api) GetFlightHandler(w http.ResponseWriter, r *http.Request) {
	recorder := newStatusRecorder(w)
	w = recorder
	action := unauthenticatedAction
	start := time.Now()
	defer func() {
		api.metrics.ObserveRequest("/api/flights/{name}", action, recorder.Status(), time.Since(start))
	}()

	ctx := r.Context()

	token, err := api.authenticateRequest(r)
	if err != nil {
		http.Error(w, "Unauthorized: "+err.Error(), http.StatusUnauthorized)
		return
	}
	action = token.Flight.Action
	if limitErr, err := api.CheckRateLimit(ctx, token); err != nil {
		http.Error(w, "Service unavailable: rate limiter temporarily unavailable", http.StatusServiceUnavailable)
		return
	} else if limitErr != nil {
		w.Header().Set("Retry-After", strconv.FormatInt(int64(limitErr.RetryAfter.Seconds()), 10))
		http.Error(w, "Too Many Requests", http.StatusTooManyRequests)
		return
	}
	if !strings.EqualFold(token.Flight.Action, JwtActionRead) {
		http.Error(w, "Forbidden: You do not have permission to read the flight", http.StatusForbidden)
		return
	}

	dispatcher := lib.NewDispatcher(
		api.serverSecret,
		api.flightDomain,
		api.k8sClient,
		api.namespace,
		token.Flight.Admin,
		token.Flight.Owner,
		token.Flight.Terminal,
		api.maxConcurrentFlights,
	)

	name := r.PathValue("name")
	if !lib.ValidK8sName(name) {
		http.Error(w, "Invalid flight plan name", http.StatusBadRequest)
		return
	}
	if !strings.EqualFold(token.Flight.Flight, name) {
		http.Error(w, "Forbidden: You do not have permission to read the flight", http.StatusForbidden)
		return
	}

	flight, err := dispatcher.GetFlight(ctx, name)
	if err != nil {
		api.flightErrorHandler(w, r, err)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	err = json.NewEncoder(w).Encode(flight)
	if err != nil {
		api.flightErrorHandler(w, r, err)
		return
	}
}

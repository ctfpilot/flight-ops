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

func (api *Api) departErrorHandler(w http.ResponseWriter, r *http.Request, err error) {
	logger := logf.FromContext(r.Context())
	logger.Error(err, "Error departing flight plan")

	// "maximum number of flights reached for owner in terminal" -> Max limit reached
	if strings.Contains(err.Error(), "maximum number of flights reached for owner in terminal") {
		http.Error(w, "Error departing flight plan - Maximum number of flights reached for owner in terminal", http.StatusConflict)
		return
	}

	http.Error(w, "Error departing flight plan - Please contact an admin for more information", http.StatusInternalServerError)
}

func (api *Api) DepartHandler(w http.ResponseWriter, r *http.Request) {
	recorder := newStatusRecorder(w)
	w = recorder
	action := unauthenticatedAction
	start := time.Now()
	defer func() {
		api.metrics.ObserveRequest("/api/flightplans/{name}/depart", action, recorder.Status(), time.Since(start))
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
	if !strings.EqualFold(token.Flight.Action, JwtActionDepart) {
		http.Error(w, "Forbidden: You do not have permission to depart flights", http.StatusForbidden)
		return
	}

	dispatcherHandler := lib.NewDispatcher(
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
	if !strings.EqualFold(token.Flight.FlightPlan, name) {
		http.Error(w, "Forbidden: You do not have permission to depart the flight", http.StatusForbidden)
		return
	}

	flightName, err := dispatcherHandler.DepartFlight(ctx, name)
	if err != nil {
		api.departErrorHandler(w, r, err)
		return
	}

	flight, err := dispatcherHandler.GetFlight(ctx, flightName)
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

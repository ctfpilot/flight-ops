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

func (api *Api) landErrorHandler(w http.ResponseWriter, r *http.Request, err error) {
	logger := logf.FromContext(r.Context())
	logger.Error(err, "Error landing flight plan")
	http.Error(w, "Error landing flight plan - Please contact an admin for more information", http.StatusInternalServerError)
}

func (api *Api) LandFlightPlanHandler(w http.ResponseWriter, r *http.Request) {
	recorder := newStatusRecorder(w)
	w = recorder
	action := unauthenticatedAction
	start := time.Now()
	defer func() {
		api.metrics.ObserveRequest("/api/flightplans/{name}/land", action, recorder.Status(), time.Since(start))
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
	if !strings.EqualFold(token.Flight.Action, JwtActionLand) {
		http.Error(w, "Forbidden: You do not have permission to land flights", http.StatusForbidden)
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

	flightPlan := r.PathValue("name")
	if !lib.ValidK8sName(flightPlan) {
		http.Error(w, "Invalid flight plan name", http.StatusBadRequest)
		return
	}
	if !strings.EqualFold(token.Flight.FlightPlan, flightPlan) {
		http.Error(w, "Forbidden: You do not have permission to land the flight", http.StatusForbidden)
		return
	}

	filter := &lib.FlightFilter{
		Owner:       token.Flight.Owner,
		Terminal:    token.Flight.Terminal,
		FlightPlans: []string{flightPlan},
	}
	flights, err := dispatcher.GetFlights(ctx, filter)
	if err != nil {
		api.flightErrorHandler(w, r, err)
		return
	}

	for _, flight := range flights.Items {
		err = dispatcher.LandFlight(ctx, flight.Name)
		if err != nil {
			api.landErrorHandler(w, r, err)
			return
		}
	}

	w.Header().Set("Content-Type", "application/json")
	err = json.NewEncoder(w).Encode(map[string]string{"message": "Flight landed successfully"})
	if err != nil {
		api.landErrorHandler(w, r, err)
		return
	}
}

func (api *Api) LandFlightHandler(w http.ResponseWriter, r *http.Request) {
	recorder := newStatusRecorder(w)
	w = recorder
	action := unauthenticatedAction
	start := time.Now()
	defer func() {
		api.metrics.ObserveRequest("/api/flights/{name}/land", action, recorder.Status(), time.Since(start))
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
	if !strings.EqualFold(token.Flight.Action, JwtActionLand) {
		http.Error(w, "Forbidden: You do not have permission to land flights", http.StatusForbidden)
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

	flightName := r.PathValue("name")
	if !lib.ValidK8sName(flightName) {
		http.Error(w, "Invalid flight name", http.StatusBadRequest)
		return
	}
	if !strings.EqualFold(token.Flight.Flight, flightName) {
		http.Error(w, "Forbidden: You do not have permission to land flight", http.StatusForbidden)
		return
	}

	err = dispatcher.LandFlight(ctx, flightName)
	if err != nil {
		api.landErrorHandler(w, r, err)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	err = json.NewEncoder(w).Encode(map[string]string{"message": "Flight landed successfully"})
	if err != nil {
		api.landErrorHandler(w, r, err)
		return
	}
}

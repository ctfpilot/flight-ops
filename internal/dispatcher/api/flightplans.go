package api

import (
	"encoding/json"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/ctfpilot/flight-ops/internal/dispatcher/lib"
)

func (api *Api) flightPlansErrorHandler(w http.ResponseWriter, err error) {
	log.Printf("Error getting flight plans: %s", err)
	http.Error(w, "Error getting flight plans - Please contact an admin for more information", http.StatusInternalServerError)
}

func (api *Api) flightPlanErrorHandler(w http.ResponseWriter, err error) {
	log.Printf("Error getting flight plan: %s", err)
	http.Error(w, "Error getting flight plan - Please contact an admin for more information", http.StatusInternalServerError)
}

func (api *Api) GetFlightPlansHandler(w http.ResponseWriter, r *http.Request) {
	recorder := newStatusRecorder(w)
	w = recorder
	action := unauthenticatedAction
	start := time.Now()
	defer func() {
		api.metrics.ObserveRequest("/api/flightplans", action, recorder.Status(), time.Since(start))
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
	if !token.Flight.Admin {
		http.Error(w, "Forbidden: You do not have permission to access flight plans", http.StatusForbidden)
		return
	}
	if !strings.EqualFold(token.Flight.Action, JwtActionList) {
		http.Error(w, "Forbidden: You do not have permission to list flight plans", http.StatusForbidden)
		return
	}
	if token.Flight.Flight != "" && token.Flight.FlightPlan != "" {
		http.Error(w, "Forbidden: You do not have permission to list flight plans", http.StatusForbidden)
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

	flightPlans, err := dispatcher.GetFlightPlans(ctx)
	if err != nil {
		api.flightPlansErrorHandler(w, err)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	err = json.NewEncoder(w).Encode(flightPlans)
	if err != nil {
		api.flightPlansErrorHandler(w, err)
		return
	}
}

func (api *Api) GetFlightPlanHandler(w http.ResponseWriter, r *http.Request) {
	recorder := newStatusRecorder(w)
	w = recorder
	action := unauthenticatedAction
	start := time.Now()
	defer func() {
		api.metrics.ObserveRequest("/api/flightplans/{name}", action, recorder.Status(), time.Since(start))
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
	if !token.Flight.Admin {
		http.Error(w, "Forbidden: You do not have permission to access the flight plan", http.StatusForbidden)
		return
	}
	if !strings.EqualFold(token.Flight.Action, JwtActionRead) {
		http.Error(w, "Forbidden: You do not have permission to read the flight plan", http.StatusForbidden)
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
	if !strings.EqualFold(token.Flight.FlightPlan, name) {
		http.Error(w, "Forbidden: You do not have permission to read the flight plan", http.StatusForbidden)
		return
	}

	flightPlan, err := dispatcher.GetFlightPlan(ctx, name)
	if err != nil {
		api.flightPlanErrorHandler(w, err)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	err = json.NewEncoder(w).Encode(flightPlan)
	if err != nil {
		api.flightPlanErrorHandler(w, err)
		return
	}
}

func (api *Api) GetFlightPlanFlightsHandler(w http.ResponseWriter, r *http.Request) {
	recorder := newStatusRecorder(w)
	w = recorder
	action := unauthenticatedAction
	start := time.Now()
	defer func() {
		api.metrics.ObserveRequest("/api/flightplans/{name}/flights", action, recorder.Status(), time.Since(start))
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
	// Non-admins are allowed to view flights for a flight plan, but they can only view flights that they own and are associated with their terminal.
	if !strings.EqualFold(token.Flight.Action, JwtActionList) {
		http.Error(w, "Forbidden: You do not have permission to list flights", http.StatusForbidden)
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
		http.Error(w, "Forbidden: You do not have permission to list the flights", http.StatusForbidden)
		return
	}

	filter := &lib.FlightFilter{
		FlightPlans: []string{flightPlan},
	}
	if !token.Flight.Admin {
		filter.Owner = token.Flight.Owner
		filter.Terminal = token.Flight.Terminal
	}
	flights, err := dispatcher.GetFlights(ctx, filter)
	if err != nil {
		api.flightErrorHandler(w, r, err)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	err = json.NewEncoder(w).Encode(flights)
	if err != nil {
		api.flightErrorHandler(w, r, err)
		return
	}
}

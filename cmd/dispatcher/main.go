/*
Copyright 2026.

Licensed under the European Union Public Licence (the ‘EUPL’), Version 1.2 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at: https://eupl.eu/1.2/en/
*/

package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/ctfpilot/flight-ops/internal/dispatcher/api"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"go.uber.org/zap/zapcore"
	ctrl "sigs.k8s.io/controller-runtime"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintf(os.Stderr, "application error: %v\n", err)
		os.Exit(1)
	}
}

func run() error {
	// Init logger from controller-runtime
	opts := zap.Options{
		Development: false,
		TimeEncoder: zapcore.ISO8601TimeEncoder,
	}
	logger := zap.New(zap.UseFlagOptions(&opts))
	ctrl.SetLogger(logger)

	setupLog := logger.WithName("setup")
	setupLog.Info("Starting Flight Ops Dispatcher...")

	rateLimitConfig, err := api.LoadRateLimitConfigFromEnv()
	if err != nil {
		return fmt.Errorf("error loading rate limit config: %w", err)
	}
	rateLimiter, err := api.NewRateLimiter(rateLimitConfig)
	if err != nil {
		return fmt.Errorf("error initializing rate limiter: %w", err)
	}
	defer func() { _ = rateLimiter.Close() }()

	metrics := api.NewDispatcherMetrics(prometheus.DefaultRegisterer)
	mux := http.NewServeMux()

	apiHandler, err := api.NewApi(
		getNamespace(),
		getVersion(),
		&api.JwtConfig{
			Secret:     getJwtSecret(),
			Issuers:    getJwtIssuers(),
			Audience:   getJwtAudience(),
			Expiration: getJwtExpiration(),
		},
		getMaxConcurrentFlights(),
		getServerSecret(),
		getFlightDomain(),
		rateLimiter,
		metrics,
	)
	if err != nil {
		return fmt.Errorf("error creating API handler: %w", err)
	}

	err = apiHandler.Init()
	if err != nil {
		return fmt.Errorf("error initializing API handler: %w", err)
	}

	mux.HandleFunc("/{$}", apiHandler.IndexHandler)
	mux.HandleFunc("/api/version", apiHandler.VersionHandler)
	mux.HandleFunc("/api/status", apiHandler.StatusHandler)
	mux.HandleFunc("/status", apiHandler.StatusHandler)

	mux.HandleFunc("/api/flightplans", apiHandler.GetFlightPlansHandler)
	mux.HandleFunc("/api/flightplans/{name}", apiHandler.GetFlightPlanHandler)
	mux.HandleFunc("/api/flightplans/{name}/flights", apiHandler.GetFlightPlanFlightsHandler)
	mux.HandleFunc("/api/flightplans/{name}/depart", apiHandler.DepartHandler)
	mux.HandleFunc("/api/flightplans/{name}/land", apiHandler.LandFlightPlanHandler)
	mux.HandleFunc("/api/flightplans/{name}/refresh", apiHandler.RefreshFlightPlanHandler)
	mux.HandleFunc("/api/flights", apiHandler.GetFlightsHandler)
	mux.HandleFunc("/api/flights/{name}", apiHandler.GetFlightHandler)
	mux.HandleFunc("/api/flights/{name}/refresh", apiHandler.RefreshFlightHandler)
	mux.HandleFunc("/api/flights/{name}/land", apiHandler.LandFlightHandler)

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, pattern := mux.Handler(r)

		// 404 if no pattern matches
		if pattern == "" {
			w.WriteHeader(http.StatusNotFound)
			if _, err := w.Write([]byte("404 - Not Found")); err != nil {
				logger := logf.FromContext(r.Context())
				logger.Error(err, "Error writing 404 response")
			}
			return
		}

		mux.ServeHTTP(w, r)
	})

	metricsMux := http.NewServeMux()
	metricsMux.Handle("/metrics", promhttp.Handler())

	// Start webserver
	server := &http.Server{
		Addr:              ":" + getPort(),
		Handler:           handler,
		ReadHeaderTimeout: 3 * time.Second,
		ReadTimeout:       10 * time.Second,
		WriteTimeout:      15 * time.Second,
		IdleTimeout:       120 * time.Second,
	}
	metricsServer := &http.Server{
		Addr:              getMetricsBindAddress(),
		Handler:           metricsMux,
		ReadHeaderTimeout: 3 * time.Second,
		ReadTimeout:       5 * time.Second,
		WriteTimeout:      5 * time.Second,
		IdleTimeout:       30 * time.Second,
	}

	serverErrors := make(chan error, 2)

	go func() {
		setupLog.Info("server listening", "port", getPort())
		if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			serverErrors <- fmt.Errorf("http server failed: %w", err)
		}
	}()

	go func() {
		setupLog.Info("metrics server listening", "port", strings.TrimPrefix(getMetricsBindAddress(), ":"))
		if err := metricsServer.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			serverErrors <- fmt.Errorf("metrics server failed: %w", err)
		}
	}()

	// Handle graceful shutdown
	shutdownSignal := make(chan os.Signal, 1)
	signal.Notify(shutdownSignal, syscall.SIGINT, syscall.SIGTERM)

	select {
	case err := <-serverErrors:
		return err
	case sig := <-shutdownSignal:
		setupLog.Info("received shutdown signal", "signal", sig.String())

		// Allow 15 seconds for active HTTP requests to complete
		ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer cancel()

		if err := server.Shutdown(ctx); err != nil {
			_ = server.Close()
			return fmt.Errorf("could not stop server gracefully: %w", err)
		}
		if err := metricsServer.Shutdown(ctx); err != nil {
			_ = metricsServer.Close()
			return fmt.Errorf("could not stop metrics server gracefully: %w", err)
		}
	}

	setupLog.Info("server stopped gracefully")
	return nil

}

func getMetricsBindAddress() string {
	metricsBindAddress := strings.TrimSpace(os.Getenv("METRICS_BIND_ADDRESS"))
	if metricsBindAddress == "" {
		return ":9090"
	}
	return metricsBindAddress
}

func getPort() string {
	// Load data from env
	port := strings.TrimSpace(os.Getenv("PORT"))
	if port == "" {
		return "8080"
	}
	return port
}

func getJwtSecret() string {
	// Load data from env
	password := strings.TrimSpace(os.Getenv("JWT_SECRET"))
	if password == "" {
		log.Fatal("JWT_SECRET environment variable is not set")
	}
	return password
}

func getJwtIssuers() []string {
	// Load data from env
	issuers := strings.TrimSpace(os.Getenv("JWT_ISSUERS"))
	if issuers == "" {
		log.Fatal("JWT_ISSUERS environment variable is not set")
	}
	return strings.Split(issuers, ",")
}

func getJwtAudience() string {
	// Load data from env
	audience := strings.TrimSpace(os.Getenv("JWT_AUDIENCE"))
	if audience == "" {
		return "ground-control-dispatcher"
	}
	return audience
}

func getJwtExpiration() int64 {
	// Load data from env
	expiration := strings.TrimSpace(os.Getenv("JWT_EXPIRATION"))
	if expiration == "" {
		return 3600 // Default to 1 hour
	}
	exp, err := strconv.ParseInt(expiration, 10, 64)
	if err != nil {
		log.Fatalf("Invalid JWT_EXPIRATION value: %s", expiration)
	}
	return exp
}

func getMaxConcurrentFlights() int {
	// Load data from env
	maxConcurrentFlights := strings.TrimSpace(os.Getenv("MAX_CONCURRENT_FLIGHTS"))
	if maxConcurrentFlights == "" {
		return 5 // Default to 5 concurrent flights
	}
	max, err := strconv.Atoi(maxConcurrentFlights)
	if err != nil || max <= 0 {
		log.Fatalf("Invalid MAX_CONCURRENT_FLIGHTS value: %s", maxConcurrentFlights)
	}
	return max
}

func getServerSecret() string {
	// Load data from env
	serverSecret := strings.TrimSpace(os.Getenv("SERVER_SECRET"))
	if serverSecret == "" {
		log.Fatal("SERVER_SECRET environment variable is not set")
	}
	return serverSecret
}

func getFlightDomain() string {
	// Load data from env
	flightDomain := strings.TrimSpace(os.Getenv("FLIGHT_DOMAIN"))
	if flightDomain == "" {
		log.Fatal("FLIGHT_DOMAIN environment variable is not set")
	}
	return flightDomain
}

func getVersion() string {
	// Load data from env
	version := strings.TrimSpace(os.Getenv("VERSION"))
	if version == "" {
		return "0.0.0"
	}
	return version
}

func getNamespace() string {
	// Load data from env
	namespace := strings.TrimSpace(os.Getenv("NAMESPACE"))
	if namespace == "" {
		return "flights-instanced"
	}
	return namespace
}

/*
Copyright 2026.

Licensed under the European Union Public Licence (the ‘EUPL’), Version 1.2 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at: https://eupl.eu/1.2/en/
*/

package controller

import (
	"context"
	"time"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	logf "sigs.k8s.io/controller-runtime/pkg/log"

	challengesv1alpha1 "github.com/ctfpilot/flight-ops/api/v1alpha1"
)

// FlightPlanReconciler reconciles a FlightPlan object
type FlightPlanReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

// +kubebuilder:rbac:groups=challenges.ctfpilot.com,resources=flightplans,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=challenges.ctfpilot.com,resources=flightplans/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=challenges.ctfpilot.com,resources=flightplans/finalizers,verbs=update

// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.24.1/pkg/reconcile
func (r *FlightPlanReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := logf.FromContext(ctx)
	logger.Info("Handling finalizer")

	// Get the FlightPlan object
	var flightPlan challengesv1alpha1.FlightPlan
	if err := r.Get(ctx, req.NamespacedName, &flightPlan); err != nil {
		// If the FlightPlan object is not found, it might have been deleted after the reconcile request.
		// In this case, we don't need to requeue the request.
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	// Handle finalizer logic
	result, err, deleting := r.finalizer(flightPlan, ctx)
	if err != nil {
		logger.Error(err, "Failed to handle finalizer for FlightPlan")
		return result, err
	}
	if result.RequeueAfter > 0 {
		logger.Info("Requeuing reconciliation for FlightPlan due to finalizer logic")
		return result, nil
	}
	if deleting {
		logger.Info("FlightPlan is being deleted. Stopping reconciliation.")
		return ctrl.Result{}, nil
	}

	logger.Info("Successfully reconciled FlightPlan resource")

	return ctrl.Result{}, nil
}

func (r *FlightPlanReconciler) finalizer(flightPlan challengesv1alpha1.FlightPlan, ctx context.Context) (ctrl.Result, error, bool) {
	logger := logf.FromContext(ctx)
	// name of our custom finalizer
	finalizerName := challengesv1alpha1.FinalizerFlightPlan

	// examine DeletionTimestamp to determine if object is under deletion
	if !flightPlan.DeletionTimestamp.IsZero() {
		// The object is being deleted
		if controllerutil.ContainsFinalizer(&flightPlan, finalizerName) {
			// our finalizer is present, so let's handle any external dependency
			logger.Info("Cleaning up active Flights for deleted FlightPlan", "slug", flightPlan.Name)

			// List all Flights tied to this FlightPlan
			var flightList challengesv1alpha1.FlightList
			if err := r.List(ctx, &flightList, client.MatchingFields{"spec.flightPlanRef": flightPlan.Name}); err != nil {
				return ctrl.Result{}, err, false
			}

			// Delete active Flights
			for _, flight := range flightList.Items {
				if err := r.Delete(ctx, &flight); err != nil && !apierrors.IsNotFound(err) {
					return ctrl.Result{}, err, false
				}
			}

			// If active Flights still exist, requeue and wait for them to finish landing
			if len(flightList.Items) > 0 {
				logger.Info("Waiting for active Flights to land before deleting FlightPlan", "count", len(flightList.Items))
				return ctrl.Result{RequeueAfter: 5 * time.Second}, nil, false
			}

			// remove our finalizer from the list and update it.
			controllerutil.RemoveFinalizer(&flightPlan, finalizerName)
			if err := r.Update(ctx, &flightPlan); err != nil {
				return ctrl.Result{}, err, false
			}
		}

		// Stop reconciliation as the item is being deleted
		logger.Info("Stopping FlightPlan reconciliation as the object is being deleted")
		return ctrl.Result{}, nil, true
	}

	// The object is not being deleted, so if it does not have our finalizer,
	// then let's add the finalizer and update the object. This is equivalent
	// to registering our finalizer.
	if !controllerutil.ContainsFinalizer(&flightPlan, finalizerName) {
		controllerutil.AddFinalizer(&flightPlan, finalizerName)
		if err := r.Update(ctx, &flightPlan); err != nil {
			return ctrl.Result{}, err, false
		}
	}

	return ctrl.Result{}, nil, false
}

// SetupWithManager sets up the controller with the Manager.
func (r *FlightPlanReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&challengesv1alpha1.FlightPlan{}).
		Named("flightplan").
		Complete(r)
}

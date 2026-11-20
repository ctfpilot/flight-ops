package lib

import (
	"context"
	"errors"

	challengesv1alpha1 "github.com/ctfpilot/flight-ops/api/v1alpha1"
	dispatcherv1alpha1 "github.com/ctfpilot/flight-ops/api/v1alpha1/dispatcher"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/selection"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	FlightOwnerLabel    = challengesv1alpha1.LabelFlightOwner
	FlightTerminalLabel = challengesv1alpha1.LabelFlightTerminal
	FlightPlanLabel     = challengesv1alpha1.LabelFlightPlan
)

type FlightFilter struct {
	Owner       string
	Terminal    string
	FlightPlans []string
}

func (d *Dispatcher) GetFlights(ctx context.Context, filter *FlightFilter) (*dispatcherv1alpha1.ApiFlightList, error) {
	if d == nil {
		return nil, errors.New("dispatcher is nil")
	}
	if filter == nil {
		filter = &FlightFilter{}
	}

	// Filter by label
	selector := labels.NewSelector()

	// Force overwrite for non-admin users to ensure they can only see their own flights
	if !d.Admin {
		filter.Owner = d.Owner
		filter.Terminal = d.Terminal
	}

	// Set up filter
	if filter.Owner != "" {
		req, err := labels.NewRequirement(FlightOwnerLabel, selection.Equals, []string{filter.Owner})
		if err != nil {
			return nil, err
		}
		selector = selector.Add(*req)
	}
	if filter.Terminal != "" {
		req, err := labels.NewRequirement(FlightTerminalLabel, selection.Equals, []string{filter.Terminal})
		if err != nil {
			return nil, err
		}
		selector = selector.Add(*req)
	}
	if len(filter.FlightPlans) > 0 {
		req, err := labels.NewRequirement(challengesv1alpha1.LabelFlightPlan, selection.In, filter.FlightPlans)
		if err != nil {
			return nil, err
		}
		selector = selector.Add(*req)
	}

	// Get flights from Kubernetes
	var flightList challengesv1alpha1.FlightList
	err := d.K8sClient.List(ctx, &flightList, &client.ListOptions{
		Namespace:     d.Namespace,
		LabelSelector: selector,
	})
	if err != nil {
		return nil, err
	}

	// Convert to ApiFlightList for API response
	apiFlightList := dispatcherv1alpha1.ApiFlightListFromFlightList(&flightList)

	return apiFlightList, nil
}

func (d *Dispatcher) GetFlight(ctx context.Context, name string) (*dispatcherv1alpha1.ApiFlight, error) {
	if d == nil {
		return nil, errors.New("dispatcher is nil")
	}

	var flight challengesv1alpha1.Flight
	err := d.K8sClient.Get(ctx, client.ObjectKey{Name: name, Namespace: d.Namespace}, &flight)
	if err != nil {
		return nil, err
	}

	// Validate user has permission to access this flight
	if !d.Admin {
		ErrForbidden := errors.New("user does not have permission to access this flight")
		if d.Owner != "" && flight.Labels[FlightOwnerLabel] != d.Owner {
			return nil, ErrForbidden
		}
		if d.Terminal != "" && flight.Labels[FlightTerminalLabel] != d.Terminal {
			return nil, ErrForbidden
		}
	}

	// Convert to ApiFlight for API response
	apiFlight := dispatcherv1alpha1.ApiFlightFromFlight(&flight)

	return apiFlight, nil
}

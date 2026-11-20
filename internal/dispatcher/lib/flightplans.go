package lib

import (
	"context"

	challengesv1alpha1 "github.com/ctfpilot/flight-ops/api/v1alpha1"
	dispatcherv1alpha1 "github.com/ctfpilot/flight-ops/api/v1alpha1/dispatcher"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func (d *Dispatcher) GetFlightPlans(ctx context.Context) (*dispatcherv1alpha1.ApiFlightPlanList, error) {
	var flightPlanList challengesv1alpha1.FlightPlanList
	err := d.K8sClient.List(ctx, &flightPlanList, &client.ListOptions{})
	if err != nil {
		return nil, err
	}

	// Convert to ApiFlightPlanList for API response
	apiFlightPlanList := dispatcherv1alpha1.ApiFlightPlanListFromFlightPlanList(&flightPlanList)

	return apiFlightPlanList, nil
}

func (d *Dispatcher) GetFlightPlan(ctx context.Context, name string) (*dispatcherv1alpha1.ApiFlightPlan, error) {
	var flightPlan challengesv1alpha1.FlightPlan
	err := d.K8sClient.Get(ctx, client.ObjectKey{Name: name, Namespace: d.Namespace}, &flightPlan)
	if err != nil {
		return nil, err
	}

	// Convert to ApiFlightPlan for API response
	apiFlightPlan := dispatcherv1alpha1.ApiFlightPlanFromFlightPlan(&flightPlan)

	return apiFlightPlan, nil
}

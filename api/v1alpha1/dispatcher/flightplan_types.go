/*
Copyright 2026.

Licensed under the European Union Public Licence (the ‘EUPL’), Version 1.2 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at: https://eupl.eu/1.2/en/
*/

package dispatcher

import (
	challengesv1alpha1 "github.com/ctfpilot/flight-ops/api/v1alpha1"
)

// ApiFlightPlan is a simplified representation of a FlightPlan for API responses.
type ApiFlightPlan struct {
	Name string                            `json:"name"`
	Spec challengesv1alpha1.FlightPlanSpec `json:"spec"`
}

func ApiFlightPlanFromFlightPlan(flightPlan *challengesv1alpha1.FlightPlan) *ApiFlightPlan {
	// Copy to new object to avoid modifying the original FlightPlan
	flightPlanCopy := flightPlan.DeepCopy()
	flightPlanCopy.Spec.Resources = nil // Exclude resources from API response to minimize data leakage

	return &ApiFlightPlan{
		Name: flightPlanCopy.Name,
		Spec: flightPlanCopy.Spec,
	}
}

// ApiFlightPlanList is a collection of ApiFlightPlan objects for API responses.
type ApiFlightPlanList struct {
	Items []ApiFlightPlan `json:"items"`
}

func ApiFlightPlanListFromFlightPlanList(flightPlanList *challengesv1alpha1.FlightPlanList) *ApiFlightPlanList {
	apiFlightPlanList := ApiFlightPlanList{
		Items: make([]ApiFlightPlan, len(flightPlanList.Items)),
	}

	for i, flightPlan := range flightPlanList.Items {
		apiFlightPlanList.Items[i] = *ApiFlightPlanFromFlightPlan(&flightPlan)
	}

	return &apiFlightPlanList
}

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

// ApiFlightOwner represents the owner and terminal information of a flight for API responses.
type ApiFlightOwner struct {
	Owner    string `json:"owner"`
	Terminal string `json:"terminal"`
}

// ApiFlight is a simplified representation of a Flight for API responses.
type ApiFlight struct {
	Name   string                          `json:"name"`
	Owner  ApiFlightOwner                  `json:"owner"`
	Spec   challengesv1alpha1.FlightSpec   `json:"spec"`
	Status challengesv1alpha1.FlightStatus `json:"status"`
}

func ApiFlightFromFlight(flight *challengesv1alpha1.Flight) *ApiFlight {
	// Extract owner and terminal from labels
	owner := flight.Labels[challengesv1alpha1.LabelFlightOwner]
	terminal := flight.Labels[challengesv1alpha1.LabelFlightTerminal]

	return &ApiFlight{
		Name:   flight.Name,
		Owner:  ApiFlightOwner{Owner: owner, Terminal: terminal},
		Spec:   flight.Spec,
		Status: flight.Status,
	}
}

// ApiFlightList is a collection of ApiFlight objects for API responses.
type ApiFlightList struct {
	Items []ApiFlight `json:"items"`
}

func ApiFlightListFromFlightList(flightList *challengesv1alpha1.FlightList) *ApiFlightList {
	apiFlightList := ApiFlightList{
		Items: make([]ApiFlight, len(flightList.Items)),
	}

	for i, flight := range flightList.Items {
		apiFlightList.Items[i] = *ApiFlightFromFlight(&flight)
	}

	return &apiFlightList
}

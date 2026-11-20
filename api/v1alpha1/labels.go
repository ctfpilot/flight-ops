/*
Copyright 2026.

Licensed under the European Union Public Licence (the ‘EUPL’), Version 1.2 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at: https://eupl.eu/1.2/en/
*/

package v1alpha1

const (

	// Flight Modes
	FlightModeStatic    = "static"
	FlightModeShared    = "shared"
	FlightModeInstanced = "instanced"

	// Label Keys
	LabelFlightMode     = "flight.ctfpilot.com/mode"
	LabelFlightCategory = "flight.ctfpilot.com/category"
	LabelFlightPlan     = "flight.ctfpilot.com/plan"

	// Refresh TTL
	LabelFlightRefresh = "flight.ctfpilot.com/refresh" // e.g., "true" or "false"

	// Origin & Multi-Tenancy Labels
	LabelFlightID       = "flight.ctfpilot.com/id"
	LabelFlightOwner    = "flight.ctfpilot.com/owner"
	LabelFlightTerminal = "flight.ctfpilot.com/terminal" // e.g., "terminal-alpha"
	LabelFlightAirfield = "flight.ctfpilot.com/airfield" // e.g., "airfield-main"

	LabelComponent = "ctfpilot.com/component"

	// Owner Special Values
	OwnerGlobal = "global"

	// Finalizers
	FinalizerFlightPlan = "groundcontrol.ctfpilot.com/flightplan-cleanup"
	FinalizerFlight     = "groundcontrol.ctfpilot.com/flight-cleanup"

	// Component Values
	ComponentInstancedChallenge = "flight"
	ComponentGroundControl      = "ground-control"
	ComponentDispatcher         = "dispatcher"
)

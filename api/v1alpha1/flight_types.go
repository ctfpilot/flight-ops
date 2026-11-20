/*
Copyright 2026.

Licensed under the European Union Public Licence (the ‘EUPL’), Version 1.2 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at: https://eupl.eu/1.2/en/
*/

package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

// FlightPhase represents the current lifecycle state of a flight instance
// +kubebuilder:validation:Enum=Pending;InFlight;Landed;Failed
type FlightPhase string

const (
	PhasePending  FlightPhase = "Pending"  // K8s resources are being created
	PhaseInFlight FlightPhase = "InFlight" // Pods are healthy and accessible
	PhaseLanded   FlightPhase = "Landed"   // Instance has expired or been stopped
	PhaseFailed   FlightPhase = "Failed"   // Deployment or pod creation failed
)

// FlightSpec defines the desired state of an active player mission
type FlightSpec struct {
	// FlightPlanRef is the metadata name/slug of the parent FlightPlan
	// +kubebuilder:validation:Immutable
	FlightPlanRef string `json:"flightPlanRef"`

	// OwnerID is the player or team ID requesting the instance
	// +kubebuilder:validation:Immutable
	OwnerID string `json:"ownerID"`

	// TerminalID is the Terminal ID (e.g. "ctfd-primary", "ctfd-university")
	// +kubebuilder:validation:Immutable
	TerminalID string `json:"terminalID"`

	// DeploymentID is the unique identifier (UUID) for this specific execution
	// +kubebuilder:validation:Immutable
	DeploymentID string `json:"deploymentID"`

	// TTL specifies how long (in seconds) this instance stays alive before auto-cleanup.
	// Defaults to FlightPlan.spec.defaultTTL if omitted.
	// +kubebuilder:validation:Immutable
	TTL int32 `json:"ttl,omitempty"`

	// Domain is the base domain used to construct instanced subdomains (e.g. "inst.ctfpilot.com")
	// +kubebuilder:validation:Immutable
	Domain string `json:"domain"`
}

// FlightStatus defines the observed state of an active mission
type FlightStatus struct {
	// Phase is the current execution state (Pending, InFlight, Landed, Failed)
	Phase FlightPhase `json:"phase,omitempty"`

	// ConnectionStrings contains the rendered connection endpoints returned to Terminal
	// e.g. ["https://usr1042-web.inst.ctfpilot.com", "nc usr1042-pwn.inst.ctfpilot.com 443"]
	ConnectionStrings []string `json:"connectionStrings,omitempty"`

	// LaunchedAt is the timestamp when reconciliation completed and pods went ready
	LaunchedAt *metav1.Time `json:"launchedAt,omitempty"`

	// ExpiresAt is the timestamp when GroundControl will clean up this instance
	ExpiresAt *metav1.Time `json:"expiresAt,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// Flight is the dynamic, per-player instance managed by GroundControl
type Flight struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   FlightSpec   `json:"spec,omitempty"`
	Status FlightStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// FlightList contains a list of flights
type FlightList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitzero"`
	Items           []Flight `json:"items"`
}

func init() {
	SchemeBuilder.Register(func(s *runtime.Scheme) error {
		s.AddKnownTypes(SchemeGroupVersion, &Flight{}, &FlightList{})
		return nil
	})
}

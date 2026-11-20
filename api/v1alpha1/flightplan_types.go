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

// EndpointProtocol defines the routing protocol for Traefik
// +kubebuilder:validation:Enum=web;tcp
type EndpointProtocol string

const (
	ProtocolWeb EndpointProtocol = "web"
	ProtocolTCP EndpointProtocol = "tcp"
)

// SubdomainSpec defines a single entry in instanced_subdomains
type SubdomainSpec struct {
	// Protocol dictates how Traefik routes traffic (web = HTTP Ingress, tcp = IngressRouteTCP)
	// +kubebuilder:default=web
	Protocol EndpointProtocol `json:"protocol"`

	// Name is the subdomain prefix (e.g. "api", "admin", "pwn")
	Name string `json:"name"`

	// ConnectionTemplate is a raw string format shown in CTFd for static/shared challenges.
	// Supports variable substitution like "{{ .host }}" or "nc {{ .host }} 1337"
	ConnectionTemplate string `json:"connectionTemplate,omitempty"`
}

// EndpointsSpec handles routing declarations for both static and instanced flights
type EndpointsSpec struct {
	// Subdomains defines the instanced subdomains for dynamic routing via Traefik (over TLS 443)
	// Corresponds to CTF Pilot instanced_subdomains
	Subdomains []SubdomainSpec `json:"subdomains,omitempty"`
}

// FlightPlanSpec defines the desired state of FlightPlan
type FlightPlanSpec struct {
	// Mode dictates where this flight lands ("shared" or "instanced")
	// +kubebuilder:validation:Enum=shared;instanced
	Mode string `json:"mode"`

	// Category is the type of challenge (web, pwn, crypto, etc.)
	Category string `json:"category"`

	// DefaultTTL is the default Time-To-Live in seconds for instanced flights (e.g., 3600)
	DefaultTTL int32 `json:"defaultTTL,omitempty"`

	// FirstDepartureAt is a Unix timestamp indicating when this flight is allowed to first be launched.
	// 0 means immediately available.
	FirstDepartureAt int64 `json:"firstDepartureAt,omitempty"`

	// Endpoints defines domain routing rules for Traefik and connection strings
	Endpoints EndpointsSpec `json:"endpoints,omitempty"`

	// Resources contains raw Kubernetes manifests (Deployments, Services, Traefik Ingresses)
	// +kubebuilder:pruning:PreserveUnknownFields
	Resources []runtime.RawExtension `json:"resources"`
}

// +kubebuilder:object:root=true
// +kubebuilder:resource:scope=Cluster
// FlightPlan is the static definition/blueprint of a challenge
type FlightPlan struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec FlightPlanSpec `json:"spec,omitempty"`
}

// +kubebuilder:object:root=true

// FlightPlanList contains a list of FlightPlan
type FlightPlanList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitzero"`
	Items           []FlightPlan `json:"items"`
}

func init() {
	SchemeBuilder.Register(func(s *runtime.Scheme) error {
		s.AddKnownTypes(SchemeGroupVersion, &FlightPlan{}, &FlightPlanList{})
		return nil
	})
}

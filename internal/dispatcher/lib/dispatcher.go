package lib

import (
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type Dispatcher struct {
	// KubernetesClient to access the Kubernetes API
	K8sClient client.Client
	// Namespace to operate in, which is used to filter the data that is returned to the user.
	Namespace string
	// FlightDomain is the domain used for the challenge, which is passed to the flight
	FlightDomain string
	// Admin indicates if the user accessing the dispatcher has admin privileges
	Admin bool
	// Owner is the owner, which is used to filter the data that is returned to the user.
	// This is used to filter the data that is returned to the user.
	Owner string
	// Terminal indicates from what terminal instance the user is accessing the dispatcher.
	// This is used to filter the data that is returned to the user.
	Terminal string
	// MaxConcurrentFlights indicates the maximum number of concurrent flights that can be running at the same time.
	// This is used to limit the number of flights that can be started at the same time.
	MaxConcurrentFlights int
	// ServerSecret used to make random re-producible values for the same dispatcher instance.
	serverSecret string
}

func NewDispatcher(serverSecret string, flightDomain string, k8sClient client.Client, namespace string, admin bool, owner string, terminal string, maxConcurrentFlights int) *Dispatcher {
	return &Dispatcher{
		K8sClient:            k8sClient,
		FlightDomain:         flightDomain,
		Namespace:            namespace,
		Admin:                admin,
		Owner:                owner,
		Terminal:             terminal,
		MaxConcurrentFlights: maxConcurrentFlights,
		serverSecret:         serverSecret,
	}
}

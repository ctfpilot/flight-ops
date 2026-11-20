package lib

import (
	"context"
	"errors"
	"fmt"

	challengesv1alpha1 "github.com/ctfpilot/flight-ops/api/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
)

// DepartFlight is responsible for creating a new flight based on the given flight plan name.
// It returns the flight name (deployment ID) if successful, or an error if the operation fails.
func (d *Dispatcher) DepartFlight(ctx context.Context, flightPlanName string) (string, error) {
	logger := logf.FromContext(ctx)

	// Validate how many of the flights the owner in terminal has, and if it is more than the max, return an error
	flights, err := d.GetFlights(ctx, nil)
	if err != nil {
		return "", err
	}

	// Count how many flights the owner has in the terminal
	count := 0
	for _, flight := range flights.Items {
		if flight.Owner.Owner == d.Owner && flight.Owner.Terminal == d.Terminal {
			count++
		}
	}

	if count >= d.MaxConcurrentFlights {
		return "", errors.New("maximum number of flights reached for owner in terminal")
	}

	// Validate that the flight plan exists
	flightPlan, err := d.GetFlightPlan(ctx, flightPlanName)
	if err != nil {
		logger.Error(err, "Failed to get flight plan", "flightPlanName", flightPlanName)
		return "", fmt.Errorf("failed to get flight plan")
	}
	if flightPlan == nil {
		return "", fmt.Errorf("flight plan %s does not exist", flightPlanName)
	}

	// Generate a unique deployment ID for the new flight
	deploymentID, err := d.generateDeploymentID(flightPlanName)
	if err != nil {
		return "", err
	}

	// Validate input values for the flight
	if !ValidK8sName(flightPlanName) {
		return "", fmt.Errorf("flight plan name '%s' is not a valid Kubernetes resource name", flightPlanName)
	}
	if !ValidK8sName(deploymentID) {
		return "", fmt.Errorf("generated deployment ID '%s' is not a valid Kubernetes resource name", deploymentID)
	}
	if !ValidK8sName(d.Owner) {
		return "", fmt.Errorf("owner '%s' is not a valid Kubernetes resource name", d.Owner)
	}
	if !ValidK8sName(d.Terminal) {
		return "", fmt.Errorf("terminal '%s' is not a valid Kubernetes resource name", d.Terminal)
	}

	// Create a new flight with the given flight plan name
	flight := &challengesv1alpha1.Flight{
		ObjectMeta: metav1.ObjectMeta{
			Name:      deploymentID,
			Namespace: d.Namespace,
		},
	}

	// Apply the flight to the Kubernetes cluster
	op, err := controllerutil.CreateOrUpdate(ctx, d.K8sClient, flight, func() error {
		// Set or update Labels
		if flight.Labels == nil {
			flight.Labels = make(map[string]string)
		}
		flight.Labels[challengesv1alpha1.LabelFlightID] = deploymentID
		flight.Labels[challengesv1alpha1.LabelFlightOwner] = d.Owner
		flight.Labels[challengesv1alpha1.LabelFlightTerminal] = d.Terminal
		flight.Labels[challengesv1alpha1.LabelFlightPlan] = flightPlanName

		// Set or update Spec
		flight.Spec = challengesv1alpha1.FlightSpec{
			FlightPlanRef: flightPlanName,
			OwnerID:       d.Owner,
			TerminalID:    d.Terminal,
			Domain:        d.FlightDomain,
			DeploymentID:  deploymentID,
		}

		return nil
	})
	if err != nil {
		logger.Error(err, "Failed to create or update flight resource", "flightPlanName", flightPlanName, "deploymentID", deploymentID)
		return "", fmt.Errorf("failed to create or update flight")
	}

	logger.Info("Flight resource departed", "deploymentID", deploymentID, "op", op) // op will be "created", "updated", or "unchanged"
	return deploymentID, nil
}

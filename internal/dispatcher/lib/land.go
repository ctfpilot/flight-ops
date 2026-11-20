package lib

import (
	"context"
	"fmt"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	logf "sigs.k8s.io/controller-runtime/pkg/log"

	challengesv1alpha1 "github.com/ctfpilot/flight-ops/api/v1alpha1"
)

func (d *Dispatcher) LandFlight(ctx context.Context, flightName string) error {
	logger := logf.FromContext(ctx)

	// Validate that the flight exists
	flight, err := d.GetFlight(ctx, flightName)
	if err != nil {
		logger.Error(err, "Failed to get flight", "flightName", flightName)
		return fmt.Errorf("failed to get flight")
	}
	if flight == nil {
		return fmt.Errorf("flight %s does not exist", flightName)
	}

	if !d.Admin {
		if flight.Owner.Owner != d.Owner || flight.Owner.Terminal != d.Terminal {
			return fmt.Errorf("flight %s does not match owner %s and terminal %s", flight.Name, d.Owner, d.Terminal)
		}
	}

	flightToDelete := &challengesv1alpha1.Flight{
		ObjectMeta: metav1.ObjectMeta{
			Name:      flight.Name,
			Namespace: d.Namespace,
		},
	}
	err = d.K8sClient.Delete(ctx, flightToDelete, &client.DeleteOptions{})
	// Verify that the error is not a NotFound error, if it is, we can ignore it
	if err != nil && !apierrors.IsNotFound(err) {
		logger.Error(err, "Failed to delete flight", "flight", flight.Name)
		return fmt.Errorf("failed to delete flight %s", flight.Name)
	} else {
		logger.Info("Successfully deleted flight", "flight", flight.Name)
	}

	return nil
}

package lib

import (
	"context"
	"errors"
	"fmt"

	challengesv1alpha1 "github.com/ctfpilot/flight-ops/api/v1alpha1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/client-go/util/retry"
	"sigs.k8s.io/controller-runtime/pkg/client"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
)

func (d *Dispatcher) RefreshFlight(ctx context.Context, flightName string) error {
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
		if d.Owner != "" && flight.Owner.Owner != d.Owner {
			return fmt.Errorf("flight %s does not exist", flightName)
		}
		if d.Terminal != "" && flight.Owner.Terminal != d.Terminal {
			return fmt.Errorf("flight %s does not exist", flightName)
		}
	}

	err = retry.RetryOnConflict(retry.DefaultRetry, func() error {
		var retrievedFlight challengesv1alpha1.Flight
		err := d.K8sClient.Get(ctx, client.ObjectKey{Name: flight.Name, Namespace: d.Namespace}, &retrievedFlight)
		if err != nil {
			return err
		}

		// Validate user has permission to access this flight
		if !d.Admin {
			ErrForbidden := errors.New("user does not have permission to access this flight")
			if d.Owner != "" && retrievedFlight.Labels[FlightOwnerLabel] != d.Owner {
				return ErrForbidden
			}
			if d.Terminal != "" && retrievedFlight.Labels[FlightTerminalLabel] != d.Terminal {
				return ErrForbidden
			}
		}

		// Add refresh label to the flight
		if retrievedFlight.Labels == nil {
			retrievedFlight.Labels = make(map[string]string)
		}
		retrievedFlight.Labels[challengesv1alpha1.LabelFlightRefresh] = "true"

		// Update the flight in Kubernetes
		return d.K8sClient.Update(ctx, &retrievedFlight)
	})

	if apierrors.IsNotFound(err) {
		return fmt.Errorf("flight %s does not exist", flightName)
	} else if err != nil {
		logger.Error(err, "Failed to refresh flight", "flight", flight.Name)
		return fmt.Errorf("failed to refresh flight %s", flight.Name)
	}
	logger.Info("Successfully refreshed flight", "flight", flight.Name)

	return nil
}

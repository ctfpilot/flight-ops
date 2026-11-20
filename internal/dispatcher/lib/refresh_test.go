package lib

import (
	"context"
	"testing"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	challengesv1alpha1 "github.com/ctfpilot/flight-ops/api/v1alpha1"
)

func TestRefreshFlight(t *testing.T) {
	scheme := runtime.NewScheme()
	utilruntime.Must(challengesv1alpha1.AddToScheme(scheme))

	t.Run("Refresh flight successfully", func(t *testing.T) {
		// Create a fake client with a flight and flight plan
		fakeClient := fake.NewClientBuilder().WithScheme(scheme).WithObjects(
			&challengesv1alpha1.Flight{
				ObjectMeta: metav1.ObjectMeta{
					Name:      defaultFlight1,
					Namespace: defaultNamespace,
					Labels: map[string]string{
						FlightOwnerLabel:    defaultOwner1,
						FlightTerminalLabel: defaultTerminal1,
						FlightPlanLabel:     defaultPlan1,
					},
				},
			},
			&challengesv1alpha1.FlightPlan{
				ObjectMeta: metav1.ObjectMeta{
					Name:      defaultPlan1,
					Namespace: defaultNamespace,
				},
			},
		).Build()

		dispatcher := &Dispatcher{
			K8sClient: fakeClient,
			Owner:     defaultOwner1,
			Terminal:  defaultTerminal1,
			Namespace: defaultNamespace,
		}

		err := dispatcher.RefreshFlight(context.Background(), defaultFlight1)
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}

		// Verify that the flight has been refreshed (label updated)
		var flight challengesv1alpha1.Flight
		err = fakeClient.Get(context.Background(), client.ObjectKey{Name: defaultFlight1, Namespace: defaultNamespace}, &flight)
		if err != nil {
			t.Fatalf("expected flight to exist, got error %v", err)
		}
		if flight.Labels[challengesv1alpha1.LabelFlightRefresh] != "true" {
			t.Fatalf("expected flight to be refreshed, but label is %v", flight.Labels[challengesv1alpha1.LabelFlightRefresh])
		}
	})

	t.Run("Refresh flight with no existing flight", func(t *testing.T) {
		fakeClient := fake.NewClientBuilder().WithScheme(scheme).Build()
		dispatcher := &Dispatcher{
			K8sClient: fakeClient,
			Owner:     defaultOwner1,
			Terminal:  defaultTerminal1,
			Namespace: defaultNamespace,
		}

		err := dispatcher.RefreshFlight(context.Background(), "nonexistent-flight")
		if err == nil {
			t.Fatalf("expected error for nonexistent flight plan, got nil")
		}
	})

	t.Run("Refresh flight with no existing flight but with flight plan", func(t *testing.T) {
		// Create a fake client with a flight plan but no flight
		fakeClient := fake.NewClientBuilder().WithScheme(scheme).WithObjects(
			&challengesv1alpha1.FlightPlan{
				ObjectMeta: metav1.ObjectMeta{
					Name:      defaultPlan1,
					Namespace: defaultNamespace,
				},
			},
		).Build()

		dispatcher := &Dispatcher{
			K8sClient: fakeClient,
			Owner:     defaultOwner1,
			Terminal:  defaultTerminal1,
			Namespace: defaultNamespace,
		}

		err := dispatcher.RefreshFlight(context.Background(), "nonexistent-flight")
		if err == nil {
			t.Fatalf("expected error for no existing flight, got nil")
		}
	})

	t.Run("Refresh flight with mismatched owner", func(t *testing.T) {
		fakeClient := fake.NewClientBuilder().WithScheme(scheme).WithObjects(
			&challengesv1alpha1.Flight{
				ObjectMeta: metav1.ObjectMeta{
					Name:      defaultFlight1,
					Namespace: defaultNamespace,
					Labels: map[string]string{
						FlightOwnerLabel:    defaultOwner2, // Mismatched owner
						FlightTerminalLabel: defaultTerminal1,
						FlightPlanLabel:     defaultPlan1,
					},
				},
			},
			&challengesv1alpha1.FlightPlan{
				ObjectMeta: metav1.ObjectMeta{
					Name:      defaultPlan1,
					Namespace: defaultNamespace,
				},
			},
		).Build()

		dispatcher := &Dispatcher{
			K8sClient: fakeClient,
			Owner:     defaultOwner1,
			Terminal:  defaultTerminal1,
			Namespace: defaultNamespace,
		}

		err := dispatcher.RefreshFlight(context.Background(), defaultFlight1)
		if err == nil {
			t.Fatalf("expected error for mismatched owner, got nil")
		}
	})

	t.Run("Refresh flight with mismatched owner and terminal", func(t *testing.T) {
		fakeClient := fake.NewClientBuilder().WithScheme(scheme).WithObjects(
			&challengesv1alpha1.Flight{
				ObjectMeta: metav1.ObjectMeta{
					Name:      defaultFlight1,
					Namespace: defaultNamespace,
					Labels: map[string]string{
						FlightOwnerLabel:    defaultOwner2,    // Mismatched owner
						FlightTerminalLabel: defaultTerminal2, // Mismatched terminal
						FlightPlanLabel:     defaultPlan1,
					},
				},
			},
			&challengesv1alpha1.FlightPlan{
				ObjectMeta: metav1.ObjectMeta{
					Name:      defaultPlan1,
					Namespace: defaultNamespace,
				},
			},
		).Build()

		dispatcher := &Dispatcher{
			K8sClient: fakeClient,
			Owner:     defaultOwner1,
			Terminal:  defaultTerminal1,
			Namespace: defaultNamespace,
		}

		err := dispatcher.RefreshFlight(context.Background(), defaultFlight1)
		if err == nil {
			t.Fatalf("expected error for mismatched owner and terminal, got nil")
		}
	})

	t.Run("Refresh flight with admin privileges", func(t *testing.T) {
		fakeClient := fake.NewClientBuilder().WithScheme(scheme).WithObjects(
			&challengesv1alpha1.Flight{
				ObjectMeta: metav1.ObjectMeta{
					Name:      defaultFlight1,
					Namespace: defaultNamespace,
					Labels: map[string]string{
						FlightOwnerLabel:    defaultOwner2,    // Mismatched owner
						FlightTerminalLabel: defaultTerminal2, // Mismatched terminal
						FlightPlanLabel:     defaultPlan1,
					},
				},
			},
			&challengesv1alpha1.FlightPlan{
				ObjectMeta: metav1.ObjectMeta{
					Name:      defaultPlan1,
					Namespace: defaultNamespace,
				},
			},
		).Build()

		dispatcher := &Dispatcher{
			K8sClient: fakeClient,
			Admin:     true, // Admin privileges
			Owner:     defaultOwner1,
			Terminal:  defaultTerminal1,
			Namespace: defaultNamespace,
		}

		err := dispatcher.RefreshFlight(context.Background(), defaultFlight1)
		if err != nil {
			t.Fatalf("expected no error for admin user, got %v", err)
		}
	})
}

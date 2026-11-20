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

func TestLandFlight(t *testing.T) {
	scheme := runtime.NewScheme()
	utilruntime.Must(challengesv1alpha1.AddToScheme(scheme))

	t.Run("Land flight successfully", func(t *testing.T) {
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

		err := dispatcher.LandFlight(context.Background(), defaultFlight1)
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}

		// Verify that the flight has been deleted
		var flight challengesv1alpha1.Flight
		err = fakeClient.Get(context.Background(), client.ObjectKey{Name: defaultFlight1, Namespace: defaultNamespace}, &flight)
		if err == nil {
			t.Fatalf("expected flight to be deleted, but it still exists")
		}
	})

	t.Run("Land flight with no existing flight plans", func(t *testing.T) {
		// Create a fake client with no flight plan
		fakeClient := fake.NewClientBuilder().WithScheme(scheme).Build()

		dispatcher := &Dispatcher{
			K8sClient: fakeClient,
			Owner:     defaultOwner1,
			Terminal:  defaultTerminal1,
			Namespace: defaultNamespace,
		}

		err := dispatcher.LandFlight(context.Background(), "nonexistent-flight")
		if err == nil {
			t.Fatalf("expected error for nonexistent flight, got nil")
		}
	})

	t.Run("Land flight with no existing flight", func(t *testing.T) {
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

		err := dispatcher.LandFlight(context.Background(), "nonexistent-flight")
		if err == nil {
			t.Fatalf("expected error for no existing flight, got nil")
		}
	})

	t.Run("Land flight with mismatched owner", func(t *testing.T) {
		// Create a fake client with a flight that has a different owner and terminal
		fakeClient := fake.NewClientBuilder().WithScheme(scheme).WithObjects(
			&challengesv1alpha1.Flight{
				ObjectMeta: metav1.ObjectMeta{
					Name:      defaultFlight1,
					Namespace: defaultNamespace,
					Labels: map[string]string{
						FlightOwnerLabel:    defaultOwner2,
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

		err := dispatcher.LandFlight(context.Background(), defaultFlight1)
		if err == nil {
			t.Fatalf("expected error for mismatched owner, got nil")
		}
	})

	t.Run("Land flight with mismatched owner and terminal", func(t *testing.T) {
		// Create a fake client with a flight that has a different owner and terminal
		fakeClient := fake.NewClientBuilder().WithScheme(scheme).WithObjects(
			&challengesv1alpha1.Flight{
				ObjectMeta: metav1.ObjectMeta{
					Name:      defaultFlight1,
					Namespace: defaultNamespace,
					Labels: map[string]string{
						FlightOwnerLabel:    defaultOwner2,
						FlightTerminalLabel: defaultTerminal2,
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

		err := dispatcher.LandFlight(context.Background(), defaultFlight1)
		if err == nil {
			t.Fatalf("expected error for mismatched owner or terminal, got nil")
		}
	})

	t.Run("Land flight with admin privileges", func(t *testing.T) {
		// Create a fake client with a flight that has a different owner and terminal
		fakeClient := fake.NewClientBuilder().WithScheme(scheme).WithObjects(
			&challengesv1alpha1.Flight{
				ObjectMeta: metav1.ObjectMeta{
					Name:      defaultFlight1,
					Namespace: defaultNamespace,
					Labels: map[string]string{
						FlightOwnerLabel:    defaultOwner2,
						FlightTerminalLabel: defaultTerminal2,
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
			Admin:     true,
		}

		err := dispatcher.LandFlight(context.Background(), defaultFlight1)
		if err != nil {
			t.Fatalf("expected no error for admin, got %v", err)
		}

		// Verify that the flight has been deleted
		var flight challengesv1alpha1.Flight
		err = fakeClient.Get(context.Background(), client.ObjectKey{Name: defaultFlight1, Namespace: defaultNamespace}, &flight)
		if err == nil {
			t.Fatalf("expected flight to be deleted, but it still exists")
		}
	})
}

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

func TestDepartFlight(t *testing.T) {

	scheme := runtime.NewScheme()
	utilruntime.Must(challengesv1alpha1.AddToScheme(scheme))

	t.Run("Owner has reached max concurrent flights", func(t *testing.T) {
		flights := challengesv1alpha1.FlightList{
			Items: []challengesv1alpha1.Flight{
				{
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
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:      defaultFlight2,
						Namespace: defaultNamespace,
						Labels: map[string]string{
							FlightOwnerLabel:    defaultOwner1,
							FlightTerminalLabel: defaultTerminal1,
							FlightPlanLabel:     defaultPlan2,
						},
					},
				},
			},
		}

		flightPlans := challengesv1alpha1.FlightPlanList{
			Items: []challengesv1alpha1.FlightPlan{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:      defaultPlan1,
						Namespace: defaultNamespace,
					},
				},
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:      defaultPlan2,
						Namespace: defaultNamespace,
					},
				},
			},
		}

		k8sClient := fake.NewClientBuilder().WithScheme(scheme).WithLists(&flights, &flightPlans).Build()
		dispatcher := NewDispatcher("secret", "domain", k8sClient, defaultNamespace, false, defaultOwner1, defaultTerminal1, 2)

		_, err := dispatcher.DepartFlight(context.Background(), "plan3")
		if err == nil || err.Error() != "maximum number of flights reached for owner in terminal" {
			t.Errorf("Expected error for max concurrent flights, got: %v", err)
		}
	})

	t.Run("Flight plan does not exist", func(t *testing.T) {
		k8sClient := fake.NewClientBuilder().WithScheme(scheme).Build()
		dispatcher := NewDispatcher("secret", "domain", k8sClient, defaultNamespace, false, defaultOwner1, defaultTerminal1, 5)

		_, err := dispatcher.DepartFlight(context.Background(), "nonexistent-plan")
		if err == nil || err.Error() != "failed to get flight plan" {
			t.Errorf("Expected error for non-existent flight plan, got: %v", err)
		}
	})

	t.Run("Successful create flight departure", func(t *testing.T) {
		flightPlans := challengesv1alpha1.FlightPlanList{
			Items: []challengesv1alpha1.FlightPlan{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:      defaultPlan1,
						Namespace: defaultNamespace,
					},
				},
			},
		}

		k8sClient := fake.NewClientBuilder().WithScheme(scheme).WithLists(&flightPlans).Build()
		dispatcher := NewDispatcher("secret", "domain", k8sClient, defaultNamespace, false, defaultOwner1, defaultTerminal1, 5)

		deploymentID, err := dispatcher.DepartFlight(context.Background(), defaultPlan1)
		if err != nil {
			t.Fatalf("Expected no error, got %v", err)
		}

		if deploymentID == "" {
			t.Errorf("Expected a valid deployment ID, got empty string")
		}

		// Validate that the flight resource was created in the fake client
		flight := &challengesv1alpha1.Flight{}
		err = k8sClient.Get(context.Background(), client.ObjectKey{Name: deploymentID, Namespace: defaultNamespace}, flight)
		if err != nil {
			t.Fatalf("Expected flight resource to be created, got error: %v", err)
		}

		if flight.Labels[FlightPlanLabel] != defaultPlan1 {
			t.Errorf("Expected flight plan label to be 'plan1', got %s", flight.Labels[FlightPlanLabel])
		}
	})

	t.Run("Successful update flight departure", func(t *testing.T) {
		flightPlans := challengesv1alpha1.FlightPlanList{
			Items: []challengesv1alpha1.FlightPlan{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:      defaultPlan1,
						Namespace: defaultNamespace,
					},
				},
			},
		}

		k8sClient := fake.NewClientBuilder().WithScheme(scheme).WithLists(&flightPlans).Build()
		dispatcher := NewDispatcher("secret", "domain", k8sClient, defaultNamespace, false, defaultOwner1, defaultTerminal1, 5)

		deploymentID, err := dispatcher.DepartFlight(context.Background(), defaultPlan1)
		if err != nil {
			t.Fatalf("Expected no error, got %v", err)
		}

		if deploymentID == "" {
			t.Errorf("Expected a valid deployment ID, got empty string")
		}

		// Validate that the flight resource was created in the fake client
		flight := &challengesv1alpha1.Flight{}
		err = k8sClient.Get(context.Background(), client.ObjectKey{Name: deploymentID, Namespace: defaultNamespace}, flight)
		if err != nil {
			t.Fatalf("Expected flight resource to be created, got error: %v", err)
		}

		if flight.Labels[FlightPlanLabel] != defaultPlan1 {
			t.Errorf("Expected flight plan label to be 'plan1', got %s", flight.Labels[FlightPlanLabel])
		}

		// Now, depart the same flight plan again, which should update the existing flight
		deploymentID2, err := dispatcher.DepartFlight(context.Background(), defaultPlan1)
		if err != nil {
			t.Fatalf("Expected no error on update, got %v", err)
		}

		if deploymentID2 != deploymentID {
			t.Errorf("Expected deployment ID to remain the same on update, got %s and %s", deploymentID, deploymentID2)
		}
	})

	t.Run("Invalid owner name", func(t *testing.T) {
		flightPlans := challengesv1alpha1.FlightPlanList{
			Items: []challengesv1alpha1.FlightPlan{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:      defaultPlan1,
						Namespace: defaultNamespace,
					},
				},
			},
		}

		k8sClient := fake.NewClientBuilder().WithScheme(scheme).WithLists(&flightPlans).Build()
		dispatcher := NewDispatcher("secret", "domain", k8sClient, defaultNamespace, false, defaultOwner1, defaultTerminal1, 5)

		// Temporarily set an invalid owner name
		dispatcher.Owner = "Invalid Owner Name!"

		_, err := dispatcher.DepartFlight(context.Background(), defaultPlan1)
		if err == nil {
			t.Errorf("Expected error for invalid owner name, got: %v", err)
		}
	})

	t.Run("Invalid terninal name", func(t *testing.T) {
		flightPlans := challengesv1alpha1.FlightPlanList{
			Items: []challengesv1alpha1.FlightPlan{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:      defaultPlan1,
						Namespace: defaultNamespace,
					},
				},
			},
		}

		k8sClient := fake.NewClientBuilder().WithScheme(scheme).WithLists(&flightPlans).Build()
		dispatcher := NewDispatcher("secret", "domain", k8sClient, defaultNamespace, false, defaultOwner1, defaultTerminal1, 5)

		// Temporarily set an invalid owner name
		dispatcher.Terminal = "Invalid Terminal Name!"

		_, err := dispatcher.DepartFlight(context.Background(), defaultPlan1)
		if err == nil {
			t.Errorf("Expected error for invalid owner name, got: %v", err)
		}
	})
}

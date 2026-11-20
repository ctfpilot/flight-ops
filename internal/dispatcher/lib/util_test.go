package lib

import (
	"strings"
	"testing"
)

func TestGenerateDeploymentID(t *testing.T) {
	// Create a mock Dispatcher
	dispatcher := &Dispatcher{
		Owner:        defaultOwner1,
		Terminal:     defaultTerminal1,
		Namespace:    defaultNamespace,
		serverSecret: defaultSecret1,
	}

	t.Run("deployment id starts with flight plan name", func(t *testing.T) {
		flightPlan := defaultFlight1
		deploymentID, err := dispatcher.generateDeploymentID(flightPlan)
		if err != nil {
			t.Fatalf("Expected no error, got %v", err)
		}

		if !strings.HasPrefix(deploymentID, flightPlan) {
			t.Errorf("Expected deployment ID to start with %s, got %s", flightPlan, deploymentID)
		}
	})

	t.Run("deployment id is consistent for same inputs", func(t *testing.T) {
		flightPlan := defaultFlight1
		deploymentID1, err := dispatcher.generateDeploymentID(flightPlan)
		if err != nil {
			t.Fatalf("Expected no error, got %v", err)
		}
		deploymentID2, err := dispatcher.generateDeploymentID(flightPlan)
		if err != nil {
			t.Fatalf("Expected no error, got %v", err)
		}

		if deploymentID1 != deploymentID2 {
			t.Errorf("Expected consistent deployment IDs, got %s and %s", deploymentID1, deploymentID2)
		}
	})

	t.Run("deployment id is different for different flight plans", func(t *testing.T) {
		flightPlan1 := defaultPlan1
		flightPlan2 := defaultPlan2
		deploymentID1, err := dispatcher.generateDeploymentID(flightPlan1)
		if err != nil {
			t.Fatalf("Expected no error, got %v", err)
		}
		deploymentID2, err := dispatcher.generateDeploymentID(flightPlan2)
		if err != nil {
			t.Fatalf("Expected no error, got %v", err)
		}

		if deploymentID1 == deploymentID2 {
			t.Errorf("Expected different deployment IDs for different flight plans, got %s and %s", deploymentID1, deploymentID2)
		}
	})

	t.Run("deployment id is different for different owners", func(t *testing.T) {
		dispatcher1 := &Dispatcher{
			Owner:        defaultOwner1,
			Terminal:     defaultTerminal1,
			Namespace:    defaultNamespace,
			serverSecret: defaultSecret1,
		}
		dispatcher2 := &Dispatcher{
			Owner:        defaultOwner2,
			Terminal:     defaultTerminal1,
			Namespace:    defaultNamespace,
			serverSecret: defaultSecret1,
		}
		flightPlan := defaultFlight1
		deploymentID1, err := dispatcher1.generateDeploymentID(flightPlan)
		if err != nil {
			t.Fatalf("Expected no error, got %v", err)
		}
		deploymentID2, err := dispatcher2.generateDeploymentID(flightPlan)
		if err != nil {
			t.Fatalf("Expected no error, got %v", err)
		}

		if deploymentID1 == deploymentID2 {
			t.Errorf("Expected different deployment IDs for different owners, got %s and %s", deploymentID1, deploymentID2)
		}
	})

	t.Run("deployment id is different for different terminals", func(t *testing.T) {
		dispatcher1 := &Dispatcher{
			Owner:        defaultOwner1,
			Terminal:     defaultTerminal2,
			Namespace:    defaultNamespace,
			serverSecret: defaultSecret1,
		}
		dispatcher2 := &Dispatcher{
			Owner:        defaultOwner1,
			Terminal:     "test-terminal-2",
			Namespace:    defaultNamespace,
			serverSecret: defaultSecret1,
		}
		flightPlan := defaultFlight1
		deploymentID1, err := dispatcher1.generateDeploymentID(flightPlan)
		if err != nil {
			t.Fatalf("Expected no error, got %v", err)
		}
		deploymentID2, err := dispatcher2.generateDeploymentID(flightPlan)
		if err != nil {
			t.Fatalf("Expected no error, got %v", err)
		}

		if deploymentID1 == deploymentID2 {
			t.Errorf("Expected different deployment IDs for different terminals, got %s and %s", deploymentID1, deploymentID2)
		}
	})

	t.Run("deployment id is different for different namespaces", func(t *testing.T) {
		dispatcher1 := &Dispatcher{
			Owner:        defaultOwner1,
			Terminal:     defaultTerminal1,
			Namespace:    "test-namespace-1",
			serverSecret: defaultSecret1,
		}
		dispatcher2 := &Dispatcher{
			Owner:        defaultOwner1,
			Terminal:     defaultTerminal1,
			Namespace:    "test-namespace-2",
			serverSecret: defaultSecret1,
		}
		flightPlan := defaultFlight1
		deploymentID1, err := dispatcher1.generateDeploymentID(flightPlan)
		if err != nil {
			t.Fatalf("Expected no error, got %v", err)
		}
		deploymentID2, err := dispatcher2.generateDeploymentID(flightPlan)
		if err != nil {
			t.Fatalf("Expected no error, got %v", err)
		}

		if deploymentID1 == deploymentID2 {
			t.Errorf("Expected different deployment IDs for different namespaces, got %s and %s", deploymentID1, deploymentID2)
		}
	})

	t.Run("deployment id is different for different server secrets", func(t *testing.T) {
		dispatcher1 := &Dispatcher{
			Owner:        defaultOwner1,
			Terminal:     defaultTerminal1,
			Namespace:    defaultNamespace,
			serverSecret: defaultSecret1,
		}
		dispatcher2 := &Dispatcher{
			Owner:        defaultOwner1,
			Terminal:     defaultTerminal1,
			Namespace:    defaultNamespace,
			serverSecret: defaultSecret2,
		}
		flightPlan := defaultFlight1
		deploymentID1, err := dispatcher1.generateDeploymentID(flightPlan)
		if err != nil {
			t.Fatalf("Expected no error, got %v", err)
		}
		deploymentID2, err := dispatcher2.generateDeploymentID(flightPlan)
		if err != nil {
			t.Fatalf("Expected no error, got %v", err)
		}

		if deploymentID1 == deploymentID2 {
			t.Errorf("Expected different deployment IDs for different server secrets, got %s and %s", deploymentID1, deploymentID2)
		}
	})
}

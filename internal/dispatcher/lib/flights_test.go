package lib

import (
	"context"
	"slices"
	"testing"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	challengesv1alpha1 "github.com/ctfpilot/flight-ops/api/v1alpha1"
)

func TestGetFlights(t *testing.T) {
	scheme := runtime.NewScheme()
	utilruntime.Must(challengesv1alpha1.AddToScheme(scheme))

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
						FlightOwnerLabel:    defaultOwner2,
						FlightTerminalLabel: defaultTerminal2,
						FlightPlanLabel:     defaultPlan1,
					},
				},
			},
			{
				ObjectMeta: metav1.ObjectMeta{
					Name:      defaultFlight3,
					Namespace: defaultNamespace,
					Labels: map[string]string{
						FlightOwnerLabel:    defaultOwner3,
						FlightTerminalLabel: defaultTerminal1,
						FlightPlanLabel:     defaultPlan2,
					},
				},
			},
			{
				ObjectMeta: metav1.ObjectMeta{
					Name:      defaultFlight4,
					Namespace: defaultNamespace,
					Labels: map[string]string{
						FlightOwnerLabel:    defaultOwner3,
						FlightTerminalLabel: defaultTerminal1,
						FlightPlanLabel:     defaultPlan2,
					},
				},
			},
			{
				ObjectMeta: metav1.ObjectMeta{
					Name:      defaultFlight5,
					Namespace: defaultNamespace,
					Labels: map[string]string{
						FlightOwnerLabel:    defaultOwner3,
						FlightTerminalLabel: defaultTerminal1,
						FlightPlanLabel:     defaultPlan3,
					},
				},
			},
		},
	}

	fakeClient := fake.NewClientBuilder().WithScheme(scheme).WithLists(&flights).Build()

	dispatcher := &Dispatcher{
		K8sClient: fakeClient,
		Namespace: defaultNamespace,
	}

	tests := []struct {
		name        string
		owner       string
		terminal    string
		admin       bool
		filter      *FlightFilter
		wantErr     bool
		wantFlights []string
	}{
		{
			name:        "Get flights with correct owner and terminal",
			owner:       defaultOwner1,
			terminal:    defaultTerminal1,
			filter:      nil,
			admin:       false,
			wantErr:     false,
			wantFlights: []string{defaultFlight1},
		},
		{
			name:        "Get flights with incorrect owner",
			owner:       wrongOwner,
			terminal:    defaultTerminal1,
			filter:      nil,
			admin:       false,
			wantErr:     false,
			wantFlights: []string{},
		},
		{
			name:        "Get flights with incorrect terminal",
			owner:       defaultOwner1,
			terminal:    wrongTerminal,
			filter:      nil,
			admin:       false,
			wantErr:     false,
			wantFlights: []string{},
		},
		{
			name:        "Get flights with correct owner and terminal with multiple flights",
			owner:       defaultOwner3,
			terminal:    defaultTerminal1,
			filter:      nil,
			admin:       false,
			wantErr:     false,
			wantFlights: []string{defaultFlight3, defaultFlight4, defaultFlight5},
		},
		{
			name:     "Get flights with correct owner and terminal, with invalid filter",
			owner:    defaultOwner1,
			terminal: defaultTerminal1,
			filter: &FlightFilter{
				Owner:    defaultOwner2,
				Terminal: defaultTerminal2,
			},
			admin:       false,
			wantErr:     false,
			wantFlights: []string{defaultFlight1},
		},
		{
			name:        "Get flights as admin",
			owner:       "",
			terminal:    "",
			filter:      nil,
			admin:       true,
			wantErr:     false,
			wantFlights: []string{defaultFlight1, defaultFlight2, defaultFlight3, defaultFlight4, defaultFlight5},
		},
		{
			name:        "Get flights as admin with incorrect owner and terminal set",
			owner:       defaultOwner2,
			terminal:    defaultTerminal2,
			filter:      nil,
			admin:       true,
			wantErr:     false,
			wantFlights: []string{defaultFlight1, defaultFlight2, defaultFlight3, defaultFlight4, defaultFlight4},
		},
		{
			name:     "Get flights as admin with independent filter",
			owner:    defaultOwner2,
			terminal: defaultTerminal2,
			filter: &FlightFilter{
				Owner:    defaultOwner1,
				Terminal: defaultTerminal1,
			},
			admin:       true,
			wantErr:     false,
			wantFlights: []string{defaultFlight1},
		},
		{
			name:     "Get flights with plan filter - plan2",
			owner:    defaultOwner3,
			terminal: defaultTerminal1,
			filter: &FlightFilter{
				Owner:       defaultOwner1,
				Terminal:    defaultTerminal1,
				FlightPlans: []string{defaultPlan2},
			},
			admin:       false,
			wantErr:     false,
			wantFlights: []string{defaultFlight3, defaultFlight4},
		},
		{
			name:     "Get flights with plan filter - plan3",
			owner:    defaultOwner3,
			terminal: defaultTerminal1,
			filter: &FlightFilter{
				Owner:       defaultOwner1,
				Terminal:    defaultTerminal1,
				FlightPlans: []string{defaultPlan3},
			},
			admin:       false,
			wantErr:     false,
			wantFlights: []string{defaultFlight5},
		},
		{
			name:     "Get flights with plan filter - incorrect plan",
			owner:    defaultOwner1,
			terminal: defaultTerminal1,
			filter: &FlightFilter{
				FlightPlans: []string{defaultPlan2},
			},
			admin:       false,
			wantErr:     false,
			wantFlights: []string{},
		},
		{
			name:     "Get flights with plan filter as admin - cross users plans",
			owner:    defaultOwner1,
			terminal: defaultTerminal1,
			filter: &FlightFilter{
				FlightPlans: []string{defaultPlan1, defaultPlan2},
			},
			admin:       true,
			wantErr:     false,
			wantFlights: []string{defaultFlight1, defaultFlight2, defaultFlight3, defaultFlight4},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dispatcher.Owner = tt.owner
			dispatcher.Terminal = tt.terminal
			dispatcher.Admin = tt.admin

			gotFlights, err := dispatcher.GetFlights(context.Background(), tt.filter)
			if (err != nil) != tt.wantErr {
				t.Errorf("GetFlights() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr {
				gotFlightNames := make([]string, len(gotFlights.Items))
				for i, flight := range gotFlights.Items {
					gotFlightNames[i] = flight.Name
				}
				if len(gotFlightNames) != len(tt.wantFlights) {
					t.Errorf("GetFlights() got %v flights, want %v flights", len(gotFlightNames), len(tt.wantFlights))
				}
				for _, wantFlight := range tt.wantFlights {
					found := slices.Contains(gotFlightNames, wantFlight)
					if !found {
						t.Errorf("GetFlights() missing expected flight %v", wantFlight)
					}
				}
			}
		})
	}

	t.Run("nil dispatcher", func(t *testing.T) {
		var nilDispatcher *Dispatcher
		_, err := nilDispatcher.GetFlights(context.Background(), nil)
		if err == nil {
			t.Errorf("GetFlights() expected error for nil dispatcher, got nil")
		}
	})
}

func TestGetFlight(t *testing.T) {
	scheme := runtime.NewScheme()
	utilruntime.Must(challengesv1alpha1.AddToScheme(scheme))

	flights := challengesv1alpha1.FlightList{
		Items: []challengesv1alpha1.Flight{
			{
				ObjectMeta: metav1.ObjectMeta{
					Name:      defaultFlight1,
					Namespace: defaultNamespace,
					Labels: map[string]string{
						FlightOwnerLabel:    defaultOwner1,
						FlightTerminalLabel: defaultTerminal1,
					},
				},
			},
			{
				ObjectMeta: metav1.ObjectMeta{
					Name:      defaultFlight2,
					Namespace: defaultNamespace,
					Labels: map[string]string{
						FlightOwnerLabel:    defaultOwner2,
						FlightTerminalLabel: defaultTerminal2,
					},
				},
			},
		},
	}

	fakeClient := fake.NewClientBuilder().WithScheme(scheme).WithLists(&flights).Build()

	dispatcher := &Dispatcher{
		K8sClient: fakeClient,
		Namespace: defaultNamespace,
	}

	tests := []struct {
		name       string
		flightName string
		owner      string
		terminal   string
		admin      bool
		wantErr    bool
	}{
		{
			name:       "Get flight with correct owner and terminal",
			flightName: defaultFlight1,
			owner:      defaultOwner1,
			terminal:   defaultTerminal1,
			admin:      false,
			wantErr:    false,
		},
		{
			name:       "Get flight with incorrect owner",
			flightName: defaultFlight1,
			owner:      wrongOwner,
			terminal:   defaultTerminal1,
			admin:      false,
			wantErr:    true,
		},
		{
			name:       "Get flight with incorrect terminal",
			flightName: defaultFlight1,
			owner:      defaultOwner1,
			terminal:   wrongTerminal,
			admin:      false,
			wantErr:    true,
		},
		{
			name:       "Get flight as admin",
			flightName: defaultFlight1,
			owner:      "",
			terminal:   "",
			admin:      true,
			wantErr:    false,
		},
		{
			name:       "Get flight as admin with wrong owner and terminal set",
			flightName: defaultFlight1,
			owner:      defaultOwner2,
			terminal:   defaultTerminal2,
			admin:      true,
			wantErr:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dispatcher.Owner = tt.owner
			dispatcher.Terminal = tt.terminal
			dispatcher.Admin = tt.admin

			gotFlight, err := dispatcher.GetFlight(context.Background(), tt.flightName)
			if (err != nil) != tt.wantErr {
				t.Errorf("GetFlight() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr {
				if gotFlight.Name != tt.flightName {
					t.Errorf("GetFlight() got = %v, want %v", gotFlight.Name, tt.flightName)
				}
			}
		})
	}
}

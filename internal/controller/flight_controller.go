/*
Copyright 2026.

Licensed under the European Union Public Licence (the ‘EUPL’), Version 1.2 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at: https://eupl.eu/1.2/en/
*/

package controller

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"text/template"
	"time"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/validation"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	logf "sigs.k8s.io/controller-runtime/pkg/log"

	challengesv1alpha1 "github.com/ctfpilot/flight-ops/api/v1alpha1"
)

// FlightReconciler reconciles a Flight object
type FlightReconciler struct {
	client.Client
	Scheme             *runtime.Scheme
	NamespaceInstanced string          // Passed from main.go (e.g. "flights-instanced")
	RegistryPrefix     string          // Container registry base (e.g. "registry.ctfpilot.com")
	AllowedKinds       map[string]bool // Allowed K8s resource Kinds in FlightPlans
}

// TemplateContext is passed to text/template when parsing raw K8s manifests
type TemplateContext struct {
	DeploymentID   string
	OwnerID        string
	TerminalID     string
	Domain         string
	RegistryPrefix string
}

// +kubebuilder:rbac:groups=challenges.ctfpilot.com,resources=flights,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=challenges.ctfpilot.com,resources=flights/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=challenges.ctfpilot.com,resources=flights/finalizers,verbs=update

// Allow the controller to manage the resources it creates (Deployments, Services, etc.)
// +kubebuilder:rbac:groups="*",resources=ingresses;ingressroutes;ingressroutetcps;ingressrouteudps;pods;deployments;services;secrets;configmaps;persistentvolumeclaims;middlewares,verbs=get;list;watch;create;update;patch;delete

// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.24.1/pkg/reconcile
func (r *FlightReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := logf.FromContext(ctx)
	logger.Info("Handling finalizer")

	// Get the Flight object
	var flight challengesv1alpha1.Flight
	if err := r.Get(ctx, req.NamespacedName, &flight); err != nil {
		// If the Flight object is not found, it might have been deleted after the reconcile request.
		// In this case, we don't need to requeue the request.
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	// Handle finalizer logic
	err, deleting := r.finalizer(flight, ctx)
	if err != nil {
		logger.Error(err, "Failed to handle finalizer for Flight")
		return ctrl.Result{}, err
	}
	if deleting {
		logger.Info("FlightPlan is being deleted. Stopping reconciliation.")
		return ctrl.Result{}, nil
	}

	// Check if the Flight is in the namespace for instanced flights, if not, it should be failed and skipped
	if flight.Namespace != r.NamespaceInstanced {
		logger.Info("Flight is not in the namespace for instanced flights. Marking as failed.", "namespace", flight.Namespace)
		flight.Status.Phase = challengesv1alpha1.PhaseFailed
		if err := r.Status().Update(ctx, &flight); err != nil {
			return ctrl.Result{}, err
		}
		return ctrl.Result{}, nil
	}

	if err := validateFlightSpecNames(&flight); err != nil {
		logger.Error(err, "Flight contains invalid resource naming")
		flight.Status.Phase = challengesv1alpha1.PhaseFailed
		if statusErr := r.Status().Update(ctx, &flight); statusErr != nil {
			return ctrl.Result{}, statusErr
		}
		return ctrl.Result{}, nil
	}

	// Check TTL Expiration
	if flight.Status.ExpiresAt != nil && time.Now().After(flight.Status.ExpiresAt.Time) {
		logger.Info("Flight TTL expired. Terminating flight", "flight", flight.Name)
		if err := r.Delete(ctx, &flight); err != nil {
			return ctrl.Result{}, err
		}
		return ctrl.Result{}, nil
	}

	// Fetch the Parent FlightPlan Blueprint
	var flightPlan challengesv1alpha1.FlightPlan
	fpKey := types.NamespacedName{Name: flight.Spec.FlightPlanRef}
	if err := r.Get(ctx, fpKey, &flightPlan); err != nil {
		if apierrors.IsNotFound(err) {
			logger.Error(err, "Referenced FlightPlan does not exist", "flightPlan", flight.Spec.FlightPlanRef)
			flight.Status.Phase = challengesv1alpha1.PhaseFailed
			_ = r.Status().Update(ctx, &flight)
			return ctrl.Result{}, nil
		}
		return ctrl.Result{}, err
	}

	// Should ensure its own labels as well, in case they were removed or modified
	labels := flight.GetLabels()
	if labels == nil {
		labels = make(map[string]string)
	}
	labels[challengesv1alpha1.LabelFlightPlan] = flightPlan.Name
	labels[challengesv1alpha1.LabelFlightID] = flight.Spec.DeploymentID
	labels[challengesv1alpha1.LabelFlightOwner] = flight.Spec.OwnerID
	labels[challengesv1alpha1.LabelFlightTerminal] = flight.Spec.TerminalID
	flight.SetLabels(labels)
	if err := r.Update(ctx, &flight); err != nil {
		return ctrl.Result{}, err
	}

	// Template & Provision Workloads
	tmplCtx := TemplateContext{
		DeploymentID:   flight.Spec.DeploymentID,
		OwnerID:        flight.Spec.OwnerID,
		TerminalID:     flight.Spec.TerminalID,
		Domain:         flight.Spec.Domain,
		RegistryPrefix: r.RegistryPrefix,
	}

	if err := r.provisionWorkloads(ctx, &flightPlan, &flight, tmplCtx); err != nil {
		logger.Error(err, "Failed to provision workloads for Flight")
		flight.Status.Phase = challengesv1alpha1.PhaseFailed
		_ = r.Status().Update(ctx, &flight)
		return ctrl.Result{}, err
	}

	// Calculate Connection Strings & Expiration Time
	connectionStrings := r.generateConnectionStrings(&flightPlan, &flight)

	now := metav1.Now()

	shouldRefresh := flight.Labels[challengesv1alpha1.LabelFlightRefresh] == "true"
	expiresAt := flight.Status.ExpiresAt
	if expiresAt == nil || shouldRefresh {
		ttlDuration := time.Duration(flight.Spec.TTL) * time.Second
		if flight.Spec.TTL <= 0 {
			defaultTTL := time.Duration(flightPlan.Spec.DefaultTTL) * time.Second
			if defaultTTL <= 0 {
				defaultTTL = 3600 * time.Second // Default to 1 hour if not specified
			}
			ttlDuration = defaultTTL
		}
		expiresAtRefresh := metav1.NewTime(now.Add(ttlDuration))
		logger.Info("Flight TTL set/updated", "flight", flight.Name, "expiresAtRefresh", expiresAtRefresh, "shouldRefresh", shouldRefresh)

		// Remove the refresh label after processing
		if shouldRefresh {
			delete(flight.Labels, challengesv1alpha1.LabelFlightRefresh)
			if err := r.Update(ctx, &flight); err != nil {
				logger.Error(err, "Failed to remove refresh label from Flight")
				return ctrl.Result{}, err
			}
		}

		expiresAt = &expiresAtRefresh
	}
	flight.Status.ExpiresAt = expiresAt

	// Update Status
	flight.Status.Phase = challengesv1alpha1.PhaseInFlight
	flight.Status.ConnectionStrings = connectionStrings
	if flight.Status.LaunchedAt == nil {
		flight.Status.LaunchedAt = &now
	}

	if err := r.Status().Update(ctx, &flight); err != nil {
		return ctrl.Result{}, err
	}

	// Requeue automatically right when the TTL expires
	timeUntilExpiration := time.Until(expiresAt.Time)
	logger.Info("Flight successfully in flight", "flight", flight.Name, "expiresIn", timeUntilExpiration)
	return ctrl.Result{RequeueAfter: timeUntilExpiration}, nil
}

func validateFlightSpecNames(flight *challengesv1alpha1.Flight) error {
	if flight == nil {
		return fmt.Errorf("flight is nil")
	}

	for field, value := range map[string]string{
		"spec.flightPlanRef": flight.Spec.FlightPlanRef,
		"spec.ownerID":       flight.Spec.OwnerID,
		"spec.terminalID":    flight.Spec.TerminalID,
		"spec.deploymentID":  flight.Spec.DeploymentID,
	} {
		if errs := validation.NameIsDNSSubdomain(value, false); len(errs) > 0 {
			return fmt.Errorf("%s must be a valid Kubernetes resource name", field)
		}
	}

	return nil
}

func (r *FlightReconciler) finalizer(flight challengesv1alpha1.Flight, ctx context.Context) (error, bool) {
	logger := logf.FromContext(ctx)
	// name of our custom finalizer
	finalizerName := challengesv1alpha1.FinalizerFlight

	// examine DeletionTimestamp to determine if object is under deletion
	if !flight.DeletionTimestamp.IsZero() {
		// The object is being deleted
		if controllerutil.ContainsFinalizer(&flight, finalizerName) {
			// our finalizer is present, so let's handle any external dependency
			logger.Info("Flight is landing (deleting). Cleaning up resources", "flight", flight.Name)

			// Native K8s OwnerReferences will clean up child workloads automatically.
			// Remove our finalizer to allow etcd deletion.
			controllerutil.RemoveFinalizer(&flight, finalizerName)
			if err := r.Update(ctx, &flight); err != nil {
				return err, false
			}
		}

		// Stop reconciliation as the item is being deleted
		logger.Info("Stopping Flight reconciliation as the object is being deleted")
		return nil, true
	}

	// The object is not being deleted, so if it does not have our finalizer,
	// then let's add the finalizer and update the object. This is equivalent
	// to registering our finalizer.
	if !controllerutil.ContainsFinalizer(&flight, finalizerName) {
		controllerutil.AddFinalizer(&flight, finalizerName)
		if err := r.Update(ctx, &flight); err != nil {
			return err, false
		}
	}

	return nil, false
}

func (r *FlightReconciler) provisionWorkloads(
	ctx context.Context,
	flightPlan *challengesv1alpha1.FlightPlan,
	flight *challengesv1alpha1.Flight,
	tmplCtx TemplateContext,
) error {
	deploymentObjects := make([]*unstructured.Unstructured, 0, len(flightPlan.Spec.Resources))

	for i, rawRes := range flightPlan.Spec.Resources {
		// Execute Go Text Template on raw YAML
		tmpl, err := template.New("manifest").Parse(string(rawRes.Raw))
		if err != nil {
			return fmt.Errorf("invalid template at index %d: %w", i, err)
		}

		var buf bytes.Buffer
		if err := tmpl.Execute(&buf, tmplCtx); err != nil {
			return fmt.Errorf("failed executing template at index %d: %w", i, err)
		}

		// Unmarshal into Unstructured Object
		obj, err := r.validateAndUnmarshalResource(buf.Bytes())
		if err != nil {
			return fmt.Errorf("failed validating/unmarshalling resource at index %d: %w", i, err)
		}

		// Mutate Metadata (Force Namespace, Unique Name, and Tracking Labels)
		obj.SetNamespace(r.NamespaceInstanced)

		// Append DeploymentID to prevent resource name collisions
		uniqueName := flight.Spec.DeploymentID
		if obj.GetName() != "" {
			uniqueName = fmt.Sprintf("%s-%s", obj.GetName(), uniqueName)
		}
		obj.SetName(uniqueName)

		// Apply standards labels for tracking and filtering
		labels := obj.GetLabels()
		if labels == nil {
			labels = make(map[string]string)
		}
		labels[challengesv1alpha1.LabelFlightMode] = challengesv1alpha1.FlightModeInstanced
		labels[challengesv1alpha1.LabelFlightPlan] = flightPlan.Name
		labels[challengesv1alpha1.LabelFlightID] = flight.Spec.DeploymentID
		labels[challengesv1alpha1.LabelFlightOwner] = flight.Spec.OwnerID
		labels[challengesv1alpha1.LabelFlightTerminal] = flight.Spec.TerminalID
		labels[challengesv1alpha1.LabelComponent] = challengesv1alpha1.ComponentInstancedChallenge
		obj.SetLabels(labels)

		// Apply additional labels and configs, for deployments
		if obj.GetKind() == "Deployment" {
			if err = r.configureDeployments(obj); err != nil {
				return fmt.Errorf("failed configuring deployment at index %d: %w", i, err)
			}
		}

		// Set OwnerReference to the Flight (Enables automatic K8s Garbage Collection!)
		if err := controllerutil.SetControllerReference(flight, obj, r.Scheme); err != nil {
			return fmt.Errorf("failed setting owner reference on element %d: %w", i, err)
		}

		// Append to list of objects to deploy
		deploymentObjects = append(deploymentObjects, obj)
	}

	for _, obj := range deploymentObjects {
		// Server-Side Apply (SSA) to create/patch resource in K8s
		applyPayload, err := json.Marshal(obj.Object)
		if err != nil {
			return fmt.Errorf("failed marshalling apply payload for resource %s: %w", obj.GetName(), err)
		}

		if err := r.Patch(
			ctx,
			obj,
			client.RawPatch(types.ApplyPatchType, applyPayload),
			client.FieldOwner("ground-control"),
		); err != nil {
			return fmt.Errorf("failed applying resource %s: %w", obj.GetName(), err)
		}
	}

	return nil
}

func (r *FlightReconciler) validateAndUnmarshalResource(raw []byte) (*unstructured.Unstructured, error) {
	obj := &unstructured.Unstructured{}
	if err := json.Unmarshal(raw, obj); err != nil {
		return nil, fmt.Errorf("invalid YAML syntax: %w", err)
	}

	kind := obj.GetKind()

	// Kind Whitelist Check
	if !r.AllowedKinds[kind] {
		return nil, fmt.Errorf("resource kind '%s' is not in the allowed kinds whitelist", kind)
	}

	return obj, nil
}

func (r *FlightReconciler) configureDeployments(obj *unstructured.Unstructured) error {
	podSpecPath := []string{"spec", "template", "spec"}

	// Pod security hardening: Disable automountServiceAccountToken and enableServiceLinks
	if err := unstructured.SetNestedField(obj.Object, false, append(podSpecPath, "automountServiceAccountToken")...); err != nil {
		return err
	}
	if err := unstructured.SetNestedField(obj.Object, false, append(podSpecPath, "enableServiceLinks")...); err != nil {
		return err
	}

	// Set DNS Policy to "None" and configure custom DNS nameservers, to prevent internal K8s DNS resolution from leaking internal service names to the instanced flight.
	if err := unstructured.SetNestedField(obj.Object, "None", append(podSpecPath, "dnsPolicy")...); err != nil {
		return err
	}
	dnsConfig := map[string]any{
		"nameservers": []any{"1.1.1.1", "8.8.8.8"},
	}
	if err := unstructured.SetNestedMap(obj.Object, dnsConfig, append(podSpecPath, "dnsConfig")...); err != nil {
		return err
	}

	return nil
}

func (r *FlightReconciler) generateConnectionStrings(
	flightPlan *challengesv1alpha1.FlightPlan,
	flight *challengesv1alpha1.Flight,
) []string {
	var connStrings []string

	for _, sub := range flightPlan.Spec.Endpoints.Subdomains {
		// Subdomain format: <subdomain>-<deploymentID>.<terminalID>.<domain>
		host := ""
		if sub.Name == "" {
			host = fmt.Sprintf("%s.%s", flight.Spec.DeploymentID, flight.Spec.Domain)
		} else {
			host = fmt.Sprintf("%s-%s.%s", sub.Name, flight.Spec.DeploymentID, flight.Spec.Domain)
		}

		// Template based connection string if provided in the FlightPlan
		if sub.ConnectionTemplate != "" {
			// Use the provided connection template with variable substitution
			tmpl, err := template.New("connTemplate").Parse(sub.ConnectionTemplate)
			if err != nil {
				continue // Skip invalid templates
			}
			var buf bytes.Buffer
			if err := tmpl.Execute(&buf, map[string]string{"host": host}); err != nil {
				continue // Skip if template execution fails
			}
			connStrings = append(connStrings, buf.String())
			continue
		}

		// Default connection strings based on protocol

		switch sub.Protocol {
		case challengesv1alpha1.ProtocolWeb:
			connStrings = append(connStrings, fmt.Sprintf("https://%s", host))
		case challengesv1alpha1.ProtocolTCP:
			connStrings = append(connStrings, fmt.Sprintf("ncat --ssl %s 443", host))
		default:
			// Unknown protocol, skip
			continue
		}
	}

	return connStrings
}

// SetupWithManager sets up the controller with the Manager.
func (r *FlightReconciler) SetupWithManager(mgr ctrl.Manager) error {
	if err := mgr.GetFieldIndexer().IndexField(context.Background(), &challengesv1alpha1.Flight{}, "spec.flightPlanRef", func(rawObj client.Object) []string {
		flight, ok := rawObj.(*challengesv1alpha1.Flight)
		if !ok {
			return nil
		}
		if flight.Spec.FlightPlanRef == "" {
			return nil
		}
		return []string{flight.Spec.FlightPlanRef}
	}); err != nil {
		return err
	}

	return ctrl.NewControllerManagedBy(mgr).
		For(&challengesv1alpha1.Flight{}).
		// Watch child Deployments and Services owned by a Flight
		Owns(&appsv1.Deployment{}).
		Owns(&corev1.Service{}).
		Named("flight").
		Complete(r)
}

/*
Copyright 2026.

Licensed under the European Union Public Licence (the ‘EUPL’), Version 1.2 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at: https://eupl.eu/1.2/en/
*/

package v1alpha1

import (
	"context"
	"encoding/json"
	"fmt"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	ctrl "sigs.k8s.io/controller-runtime"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"

	challengesv1alpha1 "github.com/ctfpilot/flight-ops/api/v1alpha1"
)

// nolint:unused
// log is for logging in this package.
var flightplanlog = logf.Log.WithName("flightplan-resource")

// SetupFlightPlanWebhookWithManager registers the webhook for FlightPlan in the manager.
func SetupFlightPlanWebhookWithManager(mgr ctrl.Manager, allowedKinds map[string]bool) error {
	return ctrl.NewWebhookManagedBy(mgr, &challengesv1alpha1.FlightPlan{}).
		WithValidator(&FlightPlanCustomValidator{AllowedKinds: allowedKinds}).
		WithDefaulter(&FlightPlanCustomDefaulter{}).
		Complete()
}

// +kubebuilder:webhook:path=/mutate-challenges-ctfpilot-com-v1alpha1-flightplan,mutating=true,failurePolicy=fail,sideEffects=None,groups=challenges.ctfpilot.com,resources=flightplans,verbs=create;update,versions=v1alpha1,name=mflightplan-v1alpha1.kb.io,admissionReviewVersions=v1

// FlightPlanCustomDefaulter struct is responsible for setting default values on the custom resource of the
// Kind FlightPlan when those are created or updated.
//
// NOTE: The +kubebuilder:object:generate=false marker prevents controller-gen from generating DeepCopy methods,
// as it is used only for temporary operations and does not need to be deeply copied.
type FlightPlanCustomDefaulter struct {
}

// Default implements webhook.CustomDefaulter so a webhook will be registered for the Kind FlightPlan.
func (d *FlightPlanCustomDefaulter) Default(_ context.Context, obj *challengesv1alpha1.FlightPlan) error {
	flightplanlog.Info("Defaulting for FlightPlan", "name", obj.GetName())
	return nil
}

// NOTE: If you want to customise the 'path', use the flags '--defaulting-path' or '--validation-path'.
// +kubebuilder:webhook:path=/validate-challenges-ctfpilot-com-v1alpha1-flightplan,mutating=false,failurePolicy=fail,sideEffects=None,groups=challenges.ctfpilot.com,resources=flightplans,verbs=create;update,versions=v1alpha1,name=vflightplan-v1alpha1.kb.io,admissionReviewVersions=v1

// FlightPlanCustomValidator struct is responsible for validating the FlightPlan resource
// when it is created, updated, or deleted.
//
// NOTE: The +kubebuilder:object:generate=false marker prevents controller-gen from generating DeepCopy methods,
// as this struct is used only for temporary operations and does not need to be deeply copied.
type FlightPlanCustomValidator struct {
	// AllowedKinds is injected from main/controller config
	AllowedKinds map[string]bool
}

// ValidateCreate implements webhook.CustomValidator so a webhook will be registered for the type FlightPlan.
func (v *FlightPlanCustomValidator) ValidateCreate(_ context.Context, obj *challengesv1alpha1.FlightPlan) (admission.Warnings, error) {
	flightplanlog.Info("Validation for FlightPlan upon creation", "name", obj.GetName())
	return nil, v.validateResources(obj)
}

// ValidateUpdate implements webhook.CustomValidator so a webhook will be registered for the type FlightPlan.
func (v *FlightPlanCustomValidator) ValidateUpdate(_ context.Context, oldObj, newObj *challengesv1alpha1.FlightPlan) (admission.Warnings, error) {
	flightplanlog.Info("Validation for FlightPlan upon update", "name", newObj.GetName())
	return nil, v.validateResources(newObj)
}

// ValidateDelete implements webhook.CustomValidator so a webhook will be registered for the type FlightPlan.
func (v *FlightPlanCustomValidator) ValidateDelete(_ context.Context, obj *challengesv1alpha1.FlightPlan) (admission.Warnings, error) {
	return nil, nil
}

func (v *FlightPlanCustomValidator) validateResources(fp *challengesv1alpha1.FlightPlan) error {
	for i, raw := range fp.Spec.Resources {
		u := &unstructured.Unstructured{}
		if err := json.Unmarshal(raw.Raw, u); err != nil {
			return fmt.Errorf("resource[%d] is invalid JSON/YAML: %w", i, err)
		}

		kind := u.GetKind()
		if !v.AllowedKinds[kind] {
			// Format allowedKinds as a comma-separated string for the error message
			allowedKindsList := make([]string, 0, len(v.AllowedKinds))
			for k := range v.AllowedKinds {
				allowedKindsList = append(allowedKindsList, k)
			}
			return fmt.Errorf("resource[%d] uses forbidden Kind '%s'. Allowed: %s", i, kind, allowedKindsList)
		}
	}
	return nil
}

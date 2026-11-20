/*
Copyright 2026.

Licensed under the European Union Public Licence (the ‘EUPL’), Version 1.2 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at: https://eupl.eu/1.2/en/
*/

package v1alpha1

import (
	"context"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"k8s.io/apimachinery/pkg/runtime"

	challengesv1alpha1 "github.com/ctfpilot/flight-ops/api/v1alpha1"
)

var _ = Describe("FlightPlan Webhook", func() {
	var (
		obj       *challengesv1alpha1.FlightPlan
		oldObj    *challengesv1alpha1.FlightPlan
		validator FlightPlanCustomValidator
		defaulter FlightPlanCustomDefaulter
		ctx       context.Context
	)

	BeforeEach(func() {
		ctx = context.Background()
		obj = &challengesv1alpha1.FlightPlan{Spec: challengesv1alpha1.FlightPlanSpec{}}
		oldObj = &challengesv1alpha1.FlightPlan{Spec: challengesv1alpha1.FlightPlanSpec{}}
		validator = FlightPlanCustomValidator{AllowedKinds: map[string]bool{"Deployment": true, "Service": true}}
		defaulter = FlightPlanCustomDefaulter{}
	})

	Context("When creating FlightPlan under Defaulting Webhook", func() {
		It("returns no error", func() {
			err := defaulter.Default(ctx, obj)
			Expect(err).NotTo(HaveOccurred())
		})
	})

	Context("When creating or updating FlightPlan under Validating Webhook", func() {
		It("admits create when all resource kinds are allowed", func() {
			obj.Spec.Resources = []runtime.RawExtension{
				{Raw: []byte(`{"apiVersion":"apps/v1","kind":"Deployment","metadata":{"name":"app"}}`)},
				{Raw: []byte(`{"apiVersion":"v1","kind":"Service","metadata":{"name":"app"}}`)},
			}

			warnings, err := validator.ValidateCreate(ctx, obj)
			Expect(err).NotTo(HaveOccurred())
			Expect(warnings).To(BeNil())
		})

		It("denies create when a resource kind is forbidden", func() {
			obj.Spec.Resources = []runtime.RawExtension{
				{Raw: []byte(`{"apiVersion":"v1","kind":"ConfigMap","metadata":{"name":"forbidden"}}`)},
			}

			_, err := validator.ValidateCreate(ctx, obj)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("forbidden Kind"))
			Expect(err.Error()).To(ContainSubstring("ConfigMap"))
		})

		It("denies create when a resource payload is malformed", func() {
			obj.Spec.Resources = []runtime.RawExtension{
				{Raw: []byte(`{"apiVersion":"v1","kind":`)},
			}

			_, err := validator.ValidateCreate(ctx, obj)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("invalid JSON/YAML"))
		})

		It("validates update against the new resource set", func() {
			oldObj.Spec.Resources = []runtime.RawExtension{
				{Raw: []byte(`{"apiVersion":"apps/v1","kind":"Deployment","metadata":{"name":"app"}}`)},
			}
			obj.Spec.Resources = []runtime.RawExtension{
				{Raw: []byte(`{"apiVersion":"v1","kind":"Secret","metadata":{"name":"forbidden-secret"}}`)},
			}

			_, err := validator.ValidateUpdate(ctx, oldObj, obj)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("Secret"))
		})

		It("allows delete without validation", func() {
			warnings, err := validator.ValidateDelete(ctx, obj)
			Expect(err).NotTo(HaveOccurred())
			Expect(warnings).To(BeNil())
		})
	})
})

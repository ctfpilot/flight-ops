/*
Copyright 2026.

Licensed under the European Union Public Licence (the ‘EUPL’), Version 1.2 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at: https://eupl.eu/1.2/en/
*/

package controller

import (
	"context"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	challengesv1alpha1 "github.com/ctfpilot/flight-ops/api/v1alpha1"
)

var _ = Describe("FlightPlan Controller", func() {
	Context("When reconciling a resource", func() {
		const (
			resourceName      = "test-resource"
			resourceNamespace = "default"

			defaultCategory = "web"
		)

		ctx := context.Background()

		newFakeFlightPlanClient := func(objects ...client.Object) client.Client {
			builder := fake.NewClientBuilder().WithScheme(k8sClient.Scheme())
			if len(objects) > 0 {
				builder = builder.WithObjects(objects...)
			}
			builder = builder.WithIndex(&challengesv1alpha1.Flight{}, "spec.flightPlanRef", func(obj client.Object) []string {
				flight, ok := obj.(*challengesv1alpha1.Flight)
				if !ok || flight.Spec.FlightPlanRef == "" {
					return nil
				}
				return []string{flight.Spec.FlightPlanRef}
			})
			return builder.Build()
		}

		It("adds the cleanup finalizer on reconcile", func() {
			flightPlan := &challengesv1alpha1.FlightPlan{
				ObjectMeta: metav1.ObjectMeta{
					Name: resourceName,
				},
				Spec: challengesv1alpha1.FlightPlanSpec{
					Mode:     challengesv1alpha1.FlightModeShared,
					Category: defaultCategory,
					Resources: []runtime.RawExtension{
						{Raw: []byte(`{"apiVersion":"v1","kind":"ConfigMap","metadata":{"name":"dummy"}}`)},
					},
				},
			}

			reconciler := &FlightPlanReconciler{
				Client: k8sClient,
				Scheme: k8sClient.Scheme(),
			}

			Expect(k8sClient.Create(ctx, flightPlan)).To(Succeed())
			DeferCleanup(func() {
				_ = k8sClient.Delete(ctx, flightPlan)
			})

			_, err := reconciler.Reconcile(ctx, reconcile.Request{
				NamespacedName: types.NamespacedName{Name: resourceName, Namespace: resourceNamespace},
			})
			Expect(err).NotTo(HaveOccurred())

			updated := &challengesv1alpha1.FlightPlan{}
			Expect(k8sClient.Get(ctx, types.NamespacedName{Name: resourceName}, updated)).To(Succeed())
			Expect(updated.Finalizers).To(ContainElement(challengesv1alpha1.FinalizerFlightPlan))
		})

		It("requeues while active flights are being cleaned up", func() {
			deletionTime := metav1.NewTime(time.Now())
			flightPlan := &challengesv1alpha1.FlightPlan{
				ObjectMeta: metav1.ObjectMeta{
					Name:              resourceName,
					Finalizers:        []string{challengesv1alpha1.FinalizerFlightPlan},
					DeletionTimestamp: &deletionTime,
				},
			}
			flight := &challengesv1alpha1.Flight{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "flight-1",
					Namespace: resourceNamespace,
				},
				Spec: challengesv1alpha1.FlightSpec{
					FlightPlanRef: resourceName,
					OwnerID:       "owner-1",
					TerminalID:    "terminal-1",
					DeploymentID:  "deployment-1",
					Domain:        "inst.example.com",
				},
			}

			fakeClient := newFakeFlightPlanClient(flightPlan, flight)
			reconciler := &FlightPlanReconciler{
				Client: fakeClient,
				Scheme: k8sClient.Scheme(),
			}

			result, err, deleting := reconciler.finalizer(*flightPlan, ctx)
			Expect(err).NotTo(HaveOccurred())
			Expect(deleting).To(BeFalse())
			Expect(result.RequeueAfter).To(Equal(5 * time.Second))

			deletedFlight := &challengesv1alpha1.Flight{}
			err = fakeClient.Get(ctx, types.NamespacedName{Name: "flight-1", Namespace: resourceNamespace}, deletedFlight)
			Expect(errors.IsNotFound(err)).To(BeTrue())
		})

		It("removes the finalizer once no active flights remain", func() {
			const cleanupResourceName = "cleanup-resource"
			deletionTime := metav1.NewTime(time.Now())

			flightPlan := &challengesv1alpha1.FlightPlan{
				ObjectMeta: metav1.ObjectMeta{
					Name:              cleanupResourceName,
					Finalizers:        []string{challengesv1alpha1.FinalizerFlightPlan},
					DeletionTimestamp: &deletionTime,
				},
				Spec: challengesv1alpha1.FlightPlanSpec{
					Mode:     challengesv1alpha1.FlightModeShared,
					Category: defaultCategory,
					Resources: []runtime.RawExtension{
						{Raw: []byte(`{"apiVersion":"v1","kind":"ConfigMap","metadata":{"name":"dummy"}}`)},
					},
				},
			}

			fakeClient := newFakeFlightPlanClient(flightPlan)

			reconciler := &FlightPlanReconciler{
				Client: fakeClient,
				Scheme: k8sClient.Scheme(),
			}

			result, err, deleting := reconciler.finalizer(*flightPlan, ctx)
			Expect(err).NotTo(HaveOccurred())
			Expect(result.RequeueAfter).To(BeZero())
			Expect(deleting).To(BeTrue())

			updated := &challengesv1alpha1.FlightPlan{}
			err = fakeClient.Get(ctx, types.NamespacedName{Name: cleanupResourceName}, updated)
			Expect(errors.IsNotFound(err)).To(BeTrue())
		})
	})
})

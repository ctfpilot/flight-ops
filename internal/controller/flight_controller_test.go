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

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	challengesv1alpha1 "github.com/ctfpilot/flight-ops/api/v1alpha1"
)

var _ = Describe("Flight Controller", func() {
	Context("When reconciling a resource", func() {
		const (
			resourceName       = "test-resource"
			resourceNamespace  = "default"
			instancedNamespace = "flights-instanced"

			deploymentString = "Deployment"

			defaultFlightPlanRef = "sample-flightplan"
			defaultCategory      = "web"
			defaultOwnerID       = "owner-1"
			defaultTerminalID    = "terminal-1"
			defaultDomain        = "inst.example.com"
		)

		ctx := context.Background()

		It("rejects manifests with unsupported kinds", func() {
			reconciler := &FlightReconciler{
				AllowedKinds: map[string]bool{
					deploymentString: true,
				},
			}

			_, err := reconciler.validateAndUnmarshalResource([]byte(`{"apiVersion":"v1","kind":"ConfigMap","metadata":{"name":"bad"}}`))
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("not in the allowed kinds whitelist"))
		})

		It("marks flights outside the instanced namespace as failed", func() {
			flight := &challengesv1alpha1.Flight{
				ObjectMeta: metav1.ObjectMeta{
					Name:       resourceName,
					Namespace:  resourceNamespace,
					Finalizers: []string{challengesv1alpha1.FinalizerFlight},
				},
				Spec: challengesv1alpha1.FlightSpec{
					FlightPlanRef: defaultFlightPlanRef,
					OwnerID:       defaultOwnerID,
					TerminalID:    defaultTerminalID,
					DeploymentID:  "deployment-1",
					Domain:        defaultDomain,
				},
			}

			reconciler := &FlightReconciler{
				Client:             k8sClient,
				Scheme:             k8sClient.Scheme(),
				NamespaceInstanced: instancedNamespace,
				AllowedKinds: map[string]bool{
					deploymentString: true,
				},
			}

			Expect(k8sClient.Create(ctx, flight)).To(Succeed())
			DeferCleanup(func() {
				_ = k8sClient.Delete(ctx, flight)
			})

			_, err := reconciler.Reconcile(ctx, reconcile.Request{
				NamespacedName: types.NamespacedName{Name: resourceName, Namespace: resourceNamespace},
			})
			Expect(err).NotTo(HaveOccurred())

			updated := &challengesv1alpha1.Flight{}
			Expect(k8sClient.Get(ctx, types.NamespacedName{Name: resourceName, Namespace: resourceNamespace}, updated)).To(Succeed())
			Expect(updated.Status.Phase).To(Equal(challengesv1alpha1.PhaseFailed))
		})

		It("fails when the referenced FlightPlan is missing", func() {
			flight := &challengesv1alpha1.Flight{
				ObjectMeta: metav1.ObjectMeta{
					Name:       resourceName,
					Namespace:  instancedNamespace,
					Finalizers: []string{challengesv1alpha1.FinalizerFlight},
				},
				Spec: challengesv1alpha1.FlightSpec{
					FlightPlanRef: "missing-flightplan",
					OwnerID:       defaultOwnerID,
					TerminalID:    defaultTerminalID,
					DeploymentID:  "deployment-2",
					Domain:        defaultDomain,
				},
			}

			reconciler := &FlightReconciler{
				Client:             k8sClient,
				Scheme:             k8sClient.Scheme(),
				NamespaceInstanced: instancedNamespace,
				AllowedKinds: map[string]bool{
					deploymentString: true,
				},
			}

			namespace := &corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: instancedNamespace}}
			Expect(k8sClient.Create(ctx, namespace)).To(Succeed())
			DeferCleanup(func() {
				_ = k8sClient.Delete(ctx, namespace)
			})
			Expect(k8sClient.Create(ctx, flight)).To(Succeed())
			DeferCleanup(func() {
				_ = k8sClient.Delete(ctx, flight)
			})

			_, err := reconciler.Reconcile(ctx, reconcile.Request{
				NamespacedName: types.NamespacedName{Name: resourceName, Namespace: instancedNamespace},
			})
			Expect(err).NotTo(HaveOccurred())

			updated := &challengesv1alpha1.Flight{}
			Expect(k8sClient.Get(ctx, types.NamespacedName{Name: resourceName, Namespace: instancedNamespace}, updated)).To(Succeed())
			Expect(updated.Status.Phase).To(Equal(challengesv1alpha1.PhaseFailed))
		})

		It("marks a valid flight as in flight after provisioning workloads", func() {
			validNamespace := "flights-instanced-1"

			flightPlan := &challengesv1alpha1.FlightPlan{
				ObjectMeta: metav1.ObjectMeta{
					Name: defaultFlightPlanRef,
				},
				Spec: challengesv1alpha1.FlightPlanSpec{
					Mode:     challengesv1alpha1.FlightModeInstanced,
					Category: defaultCategory,
					Endpoints: challengesv1alpha1.EndpointsSpec{
						Subdomains: []challengesv1alpha1.SubdomainSpec{{Protocol: challengesv1alpha1.ProtocolWeb, Name: defaultCategory}},
					},
					Resources: []runtime.RawExtension{
						{Raw: []byte(`{"apiVersion":"apps/v1","kind":"Deployment","metadata":{"name":"app"},"spec":{"replicas":1,"selector":{"matchLabels":{"app":"app"}},"template":{"metadata":{"labels":{"app":"app"}},"spec":{"containers":[{"name":"app","image":"nginx:1.25"}]}}}}`)},
					},
				},
			}
			flight := &challengesv1alpha1.Flight{
				ObjectMeta: metav1.ObjectMeta{
					Name:       resourceName,
					Namespace:  validNamespace,
					Finalizers: []string{challengesv1alpha1.FinalizerFlight},
				},
				Spec: challengesv1alpha1.FlightSpec{
					FlightPlanRef: defaultFlightPlanRef,
					OwnerID:       defaultOwnerID,
					TerminalID:    defaultTerminalID,
					DeploymentID:  "deployment-3",
					Domain:        defaultDomain,
				},
			}

			namespace := &corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: validNamespace}}
			Expect(k8sClient.Create(ctx, namespace)).To(Succeed())
			DeferCleanup(func() {
				_ = k8sClient.Delete(ctx, namespace)
			})
			Expect(k8sClient.Create(ctx, flightPlan)).To(Succeed())
			DeferCleanup(func() {
				_ = k8sClient.Delete(ctx, flightPlan)
			})
			Expect(k8sClient.Create(ctx, flight)).To(Succeed())
			DeferCleanup(func() {
				_ = k8sClient.Delete(ctx, flight)
			})

			reconciler := &FlightReconciler{
				Client:             k8sClient,
				Scheme:             k8sClient.Scheme(),
				NamespaceInstanced: validNamespace,
				AllowedKinds: map[string]bool{
					deploymentString: true,
				},
			}

			_, err := reconciler.Reconcile(ctx, reconcile.Request{
				NamespacedName: types.NamespacedName{Name: resourceName, Namespace: validNamespace},
			})
			Expect(err).NotTo(HaveOccurred())

			updated := &challengesv1alpha1.Flight{}
			Expect(k8sClient.Get(ctx, types.NamespacedName{Name: resourceName, Namespace: validNamespace}, updated)).To(Succeed())
			Expect(updated.Status.Phase).To(Equal(challengesv1alpha1.PhaseInFlight))
			Expect(updated.Status.ConnectionStrings).To(ContainElement("https://web-deployment-3.inst.example.com"))
			Expect(updated.Status.ExpiresAt).NotTo(BeNil())
		})

		It("extend with TTL when the refresh label is set to true", func() {
			validNamespace := "flights-instanced-2"

			flightPlan := &challengesv1alpha1.FlightPlan{
				ObjectMeta: metav1.ObjectMeta{
					Name: defaultFlightPlanRef,
				},
				Spec: challengesv1alpha1.FlightPlanSpec{
					Mode:     challengesv1alpha1.FlightModeInstanced,
					Category: defaultCategory,
					Endpoints: challengesv1alpha1.EndpointsSpec{
						Subdomains: []challengesv1alpha1.SubdomainSpec{{Protocol: challengesv1alpha1.ProtocolWeb, Name: defaultCategory}},
					},
					Resources: []runtime.RawExtension{
						{Raw: []byte(`{"apiVersion":"apps/v1","kind":"Deployment","metadata":{"name":"app"},"spec":{"replicas":1,"selector":{"matchLabels":{"app":"app"}},"template":{"metadata":{"labels":{"app":"app"}},"spec":{"containers":[{"name":"app","image":"nginx:1.25"}]}}}}`)},
					},
					DefaultTTL: 3600, // 1 hour TTL
				},
			}
			flight := &challengesv1alpha1.Flight{
				ObjectMeta: metav1.ObjectMeta{
					Name:       resourceName,
					Namespace:  validNamespace,
					Finalizers: []string{challengesv1alpha1.FinalizerFlight},
				},
				Spec: challengesv1alpha1.FlightSpec{
					FlightPlanRef: defaultFlightPlanRef,
					OwnerID:       defaultOwnerID,
					TerminalID:    defaultTerminalID,
					DeploymentID:  "deployment-4",
					Domain:        defaultDomain,
				},
			}

			// Create a flight, and let the controller set the ExpiresAt field
			namespace := &corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: validNamespace}}
			Expect(k8sClient.Create(ctx, namespace)).To(Succeed())
			DeferCleanup(func() {
				_ = k8sClient.Delete(ctx, namespace)
			})
			Expect(k8sClient.Create(ctx, flightPlan)).To(Succeed())
			DeferCleanup(func() {
				_ = k8sClient.Delete(ctx, flightPlan)
			})
			Expect(k8sClient.Create(ctx, flight)).To(Succeed())
			DeferCleanup(func() {
				_ = k8sClient.Delete(ctx, flight)
			})

			reconciler := &FlightReconciler{
				Client:             k8sClient,
				Scheme:             k8sClient.Scheme(),
				NamespaceInstanced: validNamespace,
				AllowedKinds: map[string]bool{
					deploymentString: true,
				},
			}

			_, err := reconciler.Reconcile(ctx, reconcile.Request{
				NamespacedName: types.NamespacedName{Name: resourceName, Namespace: validNamespace},
			})
			if err != nil {
				Fail("Reconcile failed: " + err.Error())
			}

			updated := &challengesv1alpha1.Flight{}
			Expect(k8sClient.Get(ctx, types.NamespacedName{Name: resourceName, Namespace: validNamespace}, updated)).To(Succeed())
			Expect(updated.Status.ExpiresAt).NotTo(BeNil())
			originalExpiresAt := updated.Status.ExpiresAt.DeepCopy()

			// Timout of 1 seconds to ensure the ExpiresAt is extended
			time.Sleep(1 * time.Second)

			// Update the flight to set the refresh label to true
			updated.Labels = map[string]string{
				challengesv1alpha1.LabelFlightRefresh: "true",
			}
			Expect(k8sClient.Update(ctx, updated)).To(Succeed())

			// Reconcile again to trigger the refresh logic
			_, err = reconciler.Reconcile(ctx, reconcile.Request{
				NamespacedName: types.NamespacedName{Name: resourceName, Namespace: validNamespace},
			})
			Expect(err).NotTo(HaveOccurred())
			_, err = reconciler.Reconcile(ctx, reconcile.Request{
				NamespacedName: types.NamespacedName{Name: resourceName, Namespace: validNamespace},
			})
			Expect(err).NotTo(HaveOccurred())

			// Fetch the flight again and check if ExpiresAt has been extended
			final := &challengesv1alpha1.Flight{}
			Expect(k8sClient.Get(ctx, types.NamespacedName{Name: resourceName, Namespace: validNamespace}, final)).To(Succeed())
			Expect(final.Status.ExpiresAt).NotTo(BeNil())
			Expect(final.Status.ExpiresAt.After(originalExpiresAt.Time)).To(BeTrue(), "Expected ExpiresAt to be extended after refresh")

			// Verify the final object has the refresh label removed
			Expect(final.Labels).NotTo(HaveKey(challengesv1alpha1.LabelFlightRefresh))
		})

		It("rejects flight spec values that are not valid resource names", func() {
			validNamespace := "flights-instanced-4"

			flightPlan := &challengesv1alpha1.FlightPlan{
				ObjectMeta: metav1.ObjectMeta{
					Name: defaultFlightPlanRef,
				},
				Spec: challengesv1alpha1.FlightPlanSpec{
					Mode:     challengesv1alpha1.FlightModeInstanced,
					Category: defaultCategory,
					Endpoints: challengesv1alpha1.EndpointsSpec{
						Subdomains: []challengesv1alpha1.SubdomainSpec{{Protocol: challengesv1alpha1.ProtocolWeb, Name: defaultCategory}},
					},
					Resources: []runtime.RawExtension{
						{Raw: []byte(`{"apiVersion":"apps/v1","kind":"Deployment","metadata":{"name":"app"},"spec":{"replicas":1,"selector":{"matchLabels":{"app":"app"}},"template":{"metadata":{"labels":{"app":"app"}},"spec":{"containers":[{"name":"app","image":"nginx:1.25"}]}}}}`)},
					},
				},
			}
			flight := &challengesv1alpha1.Flight{
				ObjectMeta: metav1.ObjectMeta{
					Name:       resourceName,
					Namespace:  validNamespace,
					Finalizers: []string{challengesv1alpha1.FinalizerFlight},
				},
				Spec: challengesv1alpha1.FlightSpec{
					FlightPlanRef: defaultFlightPlanRef,
					OwnerID:       "invalid owner name",
					TerminalID:    defaultTerminalID,
					DeploymentID:  "deployment-6",
					Domain:        defaultDomain,
				},
			}

			namespace := &corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: validNamespace}}
			Expect(k8sClient.Create(ctx, namespace)).To(Succeed())
			DeferCleanup(func() {
				_ = k8sClient.Delete(ctx, namespace)
			})
			Expect(k8sClient.Create(ctx, flightPlan)).To(Succeed())
			DeferCleanup(func() {
				_ = k8sClient.Delete(ctx, flightPlan)
			})
			Expect(k8sClient.Create(ctx, flight)).To(Succeed())
			DeferCleanup(func() {
				_ = k8sClient.Delete(ctx, flight)
			})

			reconciler := &FlightReconciler{
				Client:             k8sClient,
				Scheme:             k8sClient.Scheme(),
				NamespaceInstanced: validNamespace,
				AllowedKinds: map[string]bool{
					deploymentString: true,
				},
			}

			_, err := reconciler.Reconcile(ctx, reconcile.Request{
				NamespacedName: types.NamespacedName{Name: resourceName, Namespace: validNamespace},
			})
			Expect(err).NotTo(HaveOccurred())

			updated := &challengesv1alpha1.Flight{}
			Expect(k8sClient.Get(ctx, types.NamespacedName{Name: resourceName, Namespace: validNamespace}, updated)).To(Succeed())
			Expect(updated.Status.Phase).To(Equal(challengesv1alpha1.PhaseFailed))
		})

		It("adds appropriate labels to itself based on flight spec", func() {
			validNamespace := "flights-instanced-3"

			flightPlan := &challengesv1alpha1.FlightPlan{
				ObjectMeta: metav1.ObjectMeta{
					Name: defaultFlightPlanRef,
				},
				Spec: challengesv1alpha1.FlightPlanSpec{
					Mode:     challengesv1alpha1.FlightModeInstanced,
					Category: defaultCategory,
					Endpoints: challengesv1alpha1.EndpointsSpec{
						Subdomains: []challengesv1alpha1.SubdomainSpec{{Protocol: challengesv1alpha1.ProtocolWeb, Name: defaultCategory}},
					},
					Resources: []runtime.RawExtension{
						{Raw: []byte(`{"apiVersion":"apps/v1","kind":"Deployment","metadata":{"name":"app"},"spec":{"replicas":1,"selector":{"matchLabels":{"app":"app"}},"template":{"metadata":{"labels":{"app":"app"}},"spec":{"containers":[{"name":"app","image":"nginx:1.25"}]}}}}`)},
					},
				},
			}
			flight := &challengesv1alpha1.Flight{
				ObjectMeta: metav1.ObjectMeta{
					Name:       resourceName,
					Namespace:  validNamespace,
					Finalizers: []string{challengesv1alpha1.FinalizerFlight},
				},
				Spec: challengesv1alpha1.FlightSpec{
					FlightPlanRef: defaultFlightPlanRef,
					OwnerID:       defaultOwnerID,
					TerminalID:    defaultTerminalID,
					DeploymentID:  "deployment-5",
					Domain:        defaultDomain,
				},
			}

			namespace := &corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: validNamespace}}
			Expect(k8sClient.Create(ctx, namespace)).To(Succeed())
			DeferCleanup(func() {
				_ = k8sClient.Delete(ctx, namespace)
			})
			Expect(k8sClient.Create(ctx, flightPlan)).To(Succeed())
			DeferCleanup(func() {
				_ = k8sClient.Delete(ctx, flightPlan)
			})
			Expect(k8sClient.Create(ctx, flight)).To(Succeed())
			DeferCleanup(func() {
				_ = k8sClient.Delete(ctx, flight)
			})

			reconciler := &FlightReconciler{
				Client:             k8sClient,
				Scheme:             k8sClient.Scheme(),
				NamespaceInstanced: validNamespace,
				AllowedKinds: map[string]bool{
					deploymentString: true,
				},
			}

			_, err := reconciler.Reconcile(ctx, reconcile.Request{
				NamespacedName: types.NamespacedName{Name: resourceName, Namespace: validNamespace},
			})
			Expect(err).NotTo(HaveOccurred())

			updated := &challengesv1alpha1.Flight{}
			Expect(k8sClient.Get(ctx, types.NamespacedName{Name: resourceName, Namespace: validNamespace}, updated)).To(Succeed())
			Expect(updated.Labels).To(HaveKeyWithValue(challengesv1alpha1.LabelFlightPlan, defaultFlightPlanRef))
			Expect(updated.Labels).To(HaveKeyWithValue(challengesv1alpha1.LabelFlightOwner, defaultOwnerID))
			Expect(updated.Labels).To(HaveKeyWithValue(challengesv1alpha1.LabelFlightTerminal, defaultTerminalID))
			Expect(updated.Labels).To(HaveKeyWithValue(challengesv1alpha1.LabelFlightID, "deployment-5"))
		})

		Context("renders template connection", func() {
			Context("correctly from empty connection template", func() {
				It("for web endpoint", func() {
					validNamespace := "flights-instanced-5"

					flightPlan := &challengesv1alpha1.FlightPlan{
						ObjectMeta: metav1.ObjectMeta{
							Name: defaultFlightPlanRef,
						},
						Spec: challengesv1alpha1.FlightPlanSpec{
							Mode:     challengesv1alpha1.FlightModeInstanced,
							Category: defaultCategory,
							Endpoints: challengesv1alpha1.EndpointsSpec{
								Subdomains: []challengesv1alpha1.SubdomainSpec{{Protocol: challengesv1alpha1.ProtocolWeb, Name: defaultCategory}},
							},
							Resources: []runtime.RawExtension{
								{Raw: []byte(`{"apiVersion":"apps/v1","kind":"Deployment","metadata":{"name":"app"},"spec":{"replicas":1,"selector":{"matchLabels":{"app":"app"}},"template":{"metadata":{"labels":{"app":"app"}},"spec":{"containers":[{"name":"app","image":"nginx:1.25"}]}}}}`)},
							},
						},
					}
					flight := &challengesv1alpha1.Flight{
						ObjectMeta: metav1.ObjectMeta{
							Name:       resourceName,
							Namespace:  validNamespace,
							Finalizers: []string{challengesv1alpha1.FinalizerFlight},
						},
						Spec: challengesv1alpha1.FlightSpec{
							FlightPlanRef: defaultFlightPlanRef,
							OwnerID:       defaultOwnerID,
							TerminalID:    defaultTerminalID,
							DeploymentID:  "deployment-6",
							Domain:        defaultDomain,
						},
					}

					namespace := &corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: validNamespace}}
					Expect(k8sClient.Create(ctx, namespace)).To(Succeed())
					DeferCleanup(func() {
						_ = k8sClient.Delete(ctx, namespace)
					})
					Expect(k8sClient.Create(ctx, flightPlan)).To(Succeed())
					DeferCleanup(func() {
						_ = k8sClient.Delete(ctx, flightPlan)
					})
					Expect(k8sClient.Create(ctx, flight)).To(Succeed())
					DeferCleanup(func() {
						_ = k8sClient.Delete(ctx, flight)
					})

					reconciler := &FlightReconciler{
						Client:             k8sClient,
						Scheme:             k8sClient.Scheme(),
						NamespaceInstanced: validNamespace,
						AllowedKinds: map[string]bool{
							deploymentString: true,
						},
					}

					_, err := reconciler.Reconcile(ctx, reconcile.Request{
						NamespacedName: types.NamespacedName{Name: resourceName, Namespace: validNamespace},
					})
					Expect(err).NotTo(HaveOccurred())

					updated := &challengesv1alpha1.Flight{}
					Expect(k8sClient.Get(ctx, types.NamespacedName{Name: resourceName, Namespace: validNamespace}, updated)).To(Succeed())

					Expect(updated.Status.Phase).To(Equal(challengesv1alpha1.PhaseInFlight))
					Expect(updated.Status.ConnectionStrings).To(HaveLen(1))
					Expect(updated.Status.ConnectionStrings).To(ContainElement("https://web-deployment-6.inst.example.com"))
				})

				It("for tcp endpoint", func() {
					validNamespace := "flights-instanced-6"

					flightPlan := &challengesv1alpha1.FlightPlan{
						ObjectMeta: metav1.ObjectMeta{
							Name: defaultFlightPlanRef,
						},
						Spec: challengesv1alpha1.FlightPlanSpec{
							Mode:     challengesv1alpha1.FlightModeInstanced,
							Category: defaultCategory,
							Endpoints: challengesv1alpha1.EndpointsSpec{
								Subdomains: []challengesv1alpha1.SubdomainSpec{{Protocol: challengesv1alpha1.ProtocolTCP, Name: "tcp"}},
							},
							Resources: []runtime.RawExtension{
								{Raw: []byte(`{"apiVersion":"apps/v1","kind":"Deployment","metadata":{"name":"app"},"spec":{"replicas":1,"selector":{"matchLabels":{"app":"app"}},"template":{"metadata":{"labels":{"app":"app"}},"spec":{"containers":[{"name":"app","image":"nginx:1.25"}]}}}}`)},
							},
						},
					}
					flight := &challengesv1alpha1.Flight{
						ObjectMeta: metav1.ObjectMeta{
							Name:       resourceName,
							Namespace:  validNamespace,
							Finalizers: []string{challengesv1alpha1.FinalizerFlight},
						},
						Spec: challengesv1alpha1.FlightSpec{
							FlightPlanRef: defaultFlightPlanRef,
							OwnerID:       defaultOwnerID,
							TerminalID:    defaultTerminalID,
							DeploymentID:  "deployment-7",
							Domain:        defaultDomain,
						},
					}

					namespace := &corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: validNamespace}}
					Expect(k8sClient.Create(ctx, namespace)).To(Succeed())
					DeferCleanup(func() {
						_ = k8sClient.Delete(ctx, namespace)
					})
					Expect(k8sClient.Create(ctx, flightPlan)).To(Succeed())
					DeferCleanup(func() {
						_ = k8sClient.Delete(ctx, flightPlan)
					})
					Expect(k8sClient.Create(ctx, flight)).To(Succeed())
					DeferCleanup(func() {
						_ = k8sClient.Delete(ctx, flight)
					})

					reconciler := &FlightReconciler{
						Client:             k8sClient,
						Scheme:             k8sClient.Scheme(),
						NamespaceInstanced: validNamespace,
						AllowedKinds: map[string]bool{
							deploymentString: true,
						},
					}

					_, err := reconciler.Reconcile(ctx, reconcile.Request{
						NamespacedName: types.NamespacedName{Name: resourceName, Namespace: validNamespace},
					})
					Expect(err).NotTo(HaveOccurred())

					updated := &challengesv1alpha1.Flight{}
					Expect(k8sClient.Get(ctx, types.NamespacedName{Name: resourceName, Namespace: validNamespace}, updated)).To(Succeed())

					Expect(updated.Status.Phase).To(Equal(challengesv1alpha1.PhaseInFlight))
					Expect(updated.Status.ConnectionStrings).To(HaveLen(1))
					Expect(updated.Status.ConnectionStrings).To(ContainElement("ncat --ssl tcp-deployment-7.inst.example.com 443"))
				})

				It("for multiple endpoints", func() {
					validNamespace := "flights-instanced-7"

					flightPlan := &challengesv1alpha1.FlightPlan{
						ObjectMeta: metav1.ObjectMeta{
							Name: defaultFlightPlanRef,
						},
						Spec: challengesv1alpha1.FlightPlanSpec{
							Mode:     challengesv1alpha1.FlightModeInstanced,
							Category: defaultCategory,
							Endpoints: challengesv1alpha1.EndpointsSpec{
								Subdomains: []challengesv1alpha1.SubdomainSpec{
									{Protocol: challengesv1alpha1.ProtocolWeb, Name: ""},
									{Protocol: challengesv1alpha1.ProtocolTCP, Name: "second"},
								},
							},
							Resources: []runtime.RawExtension{
								{Raw: []byte(`{"apiVersion":"apps/v1","kind":"Deployment","metadata":{"name":"app"},"spec":{"replicas":1,"selector":{"matchLabels":{"app":"app"}},"template":{"metadata":{"labels":{"app":"app"}},"spec":{"containers":[{"name":"app","image":"nginx:1.25"}]}}}}`)},
							},
						},
					}
					flight := &challengesv1alpha1.Flight{
						ObjectMeta: metav1.ObjectMeta{
							Name:       resourceName,
							Namespace:  validNamespace,
							Finalizers: []string{challengesv1alpha1.FinalizerFlight},
						},
						Spec: challengesv1alpha1.FlightSpec{
							FlightPlanRef: defaultFlightPlanRef,
							OwnerID:       defaultOwnerID,
							TerminalID:    defaultTerminalID,
							DeploymentID:  "deployment-8",
							Domain:        defaultDomain,
						},
					}

					namespace := &corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: validNamespace}}
					Expect(k8sClient.Create(ctx, namespace)).To(Succeed())
					DeferCleanup(func() {
						_ = k8sClient.Delete(ctx, namespace)
					})
					Expect(k8sClient.Create(ctx, flightPlan)).To(Succeed())
					DeferCleanup(func() {
						_ = k8sClient.Delete(ctx, flightPlan)
					})
					Expect(k8sClient.Create(ctx, flight)).To(Succeed())
					DeferCleanup(func() {
						_ = k8sClient.Delete(ctx, flight)
					})

					reconciler := &FlightReconciler{
						Client:             k8sClient,
						Scheme:             k8sClient.Scheme(),
						NamespaceInstanced: validNamespace,
						AllowedKinds: map[string]bool{
							deploymentString: true,
						},
					}

					_, err := reconciler.Reconcile(ctx, reconcile.Request{
						NamespacedName: types.NamespacedName{Name: resourceName, Namespace: validNamespace},
					})
					Expect(err).NotTo(HaveOccurred())

					updated := &challengesv1alpha1.Flight{}
					Expect(k8sClient.Get(ctx, types.NamespacedName{Name: resourceName, Namespace: validNamespace}, updated)).To(Succeed())

					Expect(updated.Status.Phase).To(Equal(challengesv1alpha1.PhaseInFlight))
					Expect(updated.Status.ConnectionStrings).To(HaveLen(2))
					Expect(updated.Status.ConnectionStrings).To(ContainElement("https://deployment-8.inst.example.com"))
					Expect(updated.Status.ConnectionStrings).To(ContainElement("ncat --ssl second-deployment-8.inst.example.com 443"))
				})
			})

			Context("correctly from a connection template", func() {
				It("for multiple endpoints", func() {
					validNamespace := "flights-instanced-8"

					flightPlan := &challengesv1alpha1.FlightPlan{
						ObjectMeta: metav1.ObjectMeta{
							Name: defaultFlightPlanRef,
						},
						Spec: challengesv1alpha1.FlightPlanSpec{
							Mode:     challengesv1alpha1.FlightModeInstanced,
							Category: defaultCategory,
							Endpoints: challengesv1alpha1.EndpointsSpec{
								Subdomains: []challengesv1alpha1.SubdomainSpec{
									{Protocol: challengesv1alpha1.ProtocolWeb, Name: "", ConnectionTemplate: "http://demo-{{ .host }}"},
									{Protocol: challengesv1alpha1.ProtocolTCP, Name: "second", ConnectionTemplate: "custom prefix {{ .host }} custom suffix"},
								},
							},
							Resources: []runtime.RawExtension{
								{Raw: []byte(`{"apiVersion":"apps/v1","kind":"Deployment","metadata":{"name":"app"},"spec":{"replicas":1,"selector":{"matchLabels":{"app":"app"}},"template":{"metadata":{"labels":{"app":"app"}},"spec":{"containers":[{"name":"app","image":"nginx:1.25"}]}}}}`)},
							},
						},
					}
					flight := &challengesv1alpha1.Flight{
						ObjectMeta: metav1.ObjectMeta{
							Name:       resourceName,
							Namespace:  validNamespace,
							Finalizers: []string{challengesv1alpha1.FinalizerFlight},
						},
						Spec: challengesv1alpha1.FlightSpec{
							FlightPlanRef: defaultFlightPlanRef,
							OwnerID:       defaultOwnerID,
							TerminalID:    defaultTerminalID,
							DeploymentID:  "deployment-9",
							Domain:        defaultDomain,
						},
					}

					namespace := &corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: validNamespace}}
					Expect(k8sClient.Create(ctx, namespace)).To(Succeed())
					DeferCleanup(func() {
						_ = k8sClient.Delete(ctx, namespace)
					})
					Expect(k8sClient.Create(ctx, flightPlan)).To(Succeed())
					DeferCleanup(func() {
						_ = k8sClient.Delete(ctx, flightPlan)
					})
					Expect(k8sClient.Create(ctx, flight)).To(Succeed())
					DeferCleanup(func() {
						_ = k8sClient.Delete(ctx, flight)
					})

					reconciler := &FlightReconciler{
						Client:             k8sClient,
						Scheme:             k8sClient.Scheme(),
						NamespaceInstanced: validNamespace,
						AllowedKinds: map[string]bool{
							deploymentString: true,
						},
					}

					_, err := reconciler.Reconcile(ctx, reconcile.Request{
						NamespacedName: types.NamespacedName{Name: resourceName, Namespace: validNamespace},
					})
					Expect(err).NotTo(HaveOccurred())

					updated := &challengesv1alpha1.Flight{}
					Expect(k8sClient.Get(ctx, types.NamespacedName{Name: resourceName, Namespace: validNamespace}, updated)).To(Succeed())

					Expect(updated.Status.Phase).To(Equal(challengesv1alpha1.PhaseInFlight))
					Expect(updated.Status.ConnectionStrings).To(HaveLen(2))
					Expect(updated.Status.ConnectionStrings).To(ContainElement("http://demo-deployment-9.inst.example.com"))
					Expect(updated.Status.ConnectionStrings).To(ContainElement("custom prefix second-deployment-9.inst.example.com custom suffix"))
				})
			})
		})
	})
})

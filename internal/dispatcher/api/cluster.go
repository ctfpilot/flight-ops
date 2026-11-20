package api

import (
	"log"

	challengesv1alpha1 "github.com/ctfpilot/flight-ops/api/v1alpha1"
	"k8s.io/apimachinery/pkg/runtime"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/client-go/kubernetes"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func (api *Api) initClusterClient() error {
	log.Println("Initializing Kubernetes client...")

	// Create a new Kubernetes client
	config, err := rest.InClusterConfig()
	if err != nil {
		log.Println("Error creating in-cluster config:", err)
		return err
	}

	api.clientSet, err = kubernetes.NewForConfig(config)
	if err != nil {
		log.Println("Error creating Kubernetes client:", err)
		return err
	}

	scheme := runtime.NewScheme()
	utilruntime.Must(clientgoscheme.AddToScheme(scheme))
	utilruntime.Must(challengesv1alpha1.AddToScheme(scheme))

	k8sClient, err := client.New(config, client.Options{
		Scheme: scheme,
	})
	if err != nil {
		log.Println("Error creating controller-runtime client:", err)
		return err
	}

	// Check access to the Kubernetes API
	_, err = api.clientSet.Discovery().ServerVersion()
	if err != nil {
		log.Println("Error checking access to Kubernetes API:", err)
		return err
	}

	api.k8sClient = k8sClient
	log.Println("Kubernetes client initialized successfully")

	return nil
}

func (api *Api) checkAccess() error {
	// Check access to the Kubernetes API
	_, err := api.clientSet.Discovery().ServerVersion()
	if err != nil {
		return err
	}

	return nil
}

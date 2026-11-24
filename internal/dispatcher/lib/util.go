package lib

import (
	"crypto/hmac"
	"crypto/sha256"
	"fmt"

	"k8s.io/apimachinery/pkg/api/validation"
)

// Generate Deployment ID, which is a unique identifier for a specific execution of a flight.
// Based on the same owner, terminal and flight plan, this ID will be identical for the same flight plan execution, but different for different owners, terminals, or flight plans.
func (d *Dispatcher) generateDeploymentID(flightPlan string) (string, error) {
	base := flightPlan + "-" + d.Owner + "-" + d.Terminal + "-" + d.Namespace
	encryptionKey := []byte(d.serverSecret)

	encrypter := hmac.New(sha256.New, encryptionKey)
	_, err := encrypter.Write([]byte(base))
	if err != nil {
		return "", err
	}

	encrypted := encrypter.Sum(nil)

	randLength := 8
	randPart := encrypted[:randLength]

	// Limit size of deployment id
	if len(flightPlan) > 32 {
		flightPlan = flightPlan[:32]
	}

	return fmt.Sprintf("%s-%x-%s", flightPlan, randPart, d.Terminal), nil
}

// ValidK8sName checks if the provided name is a valid Kubernetes resource name.
func ValidK8sName(name string) bool {
	errs := validation.NameIsDNSSubdomain(name, false)
	return len(errs) == 0
}

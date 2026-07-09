package controller

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"

	orbv1alpha1 "github.com/joelanford/orb-operator/api/v1alpha1"
)

func templateHash(tmpl orbv1alpha1.ClusterObjectDeploymentTemplate) (string, error) {
	data, err := json.Marshal(tmpl)
	if err != nil {
		return "", fmt.Errorf("marshalling template for hash: %w", err)
	}
	sum := sha256.Sum256(data)
	return fmt.Sprintf("%x", sum[:4]), nil
}

package controller

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"

	orbv1alpha1 "github.com/joelanford/orb-operator/api/v1alpha1"
)

const labelTemplateHash = "orb.operatorframework.io/template-hash"

func templateHash(tmpl orbv1alpha1.ClusterObjectSetTemplate) string {
	data, err := json.Marshal(tmpl)
	if err != nil {
		panic(fmt.Sprintf("marshalling template for hash: %v", err))
	}
	sum := sha256.Sum256(data)
	return fmt.Sprintf("%x", sum[:4])
}

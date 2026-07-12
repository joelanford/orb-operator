package template

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"maps"

	metav1ac "k8s.io/client-go/applyconfigurations/meta/v1"

	orbv1alpha1 "github.com/joelanford/orb-operator/api/v1alpha1"
	cosac "github.com/joelanford/orb-operator/applyconfigurations/api/v1alpha1"
)

func Hash(tmpl orbv1alpha1.ClusterObjectDeploymentTemplate) (string, error) {
	data, err := json.Marshal(tmpl)
	if err != nil {
		return "", fmt.Errorf("marshalling template for hash: %w", err)
	}
	sum := sha256.Sum256(data)
	return fmt.Sprintf("%x", sum[:4]), nil
}

func BuildCOS(cod *orbv1alpha1.ClusterObjectDeployment, revision uint32, hash string) (*cosac.ClusterObjectSetApplyConfiguration, error) {
	labels := maps.Clone(cod.Spec.Template.Metadata.Labels)
	if labels == nil {
		labels = map[string]string{}
	}
	labels[LabelTemplateHash] = hash

	tmplSpecJSON, err := json.Marshal(cod.Spec.Template.Spec)
	if err != nil {
		return nil, err
	}

	var cosSpec cosac.ClusterObjectSetSpecApplyConfiguration
	if err := json.Unmarshal(tmplSpecJSON, &cosSpec); err != nil {
		return nil, err
	}

	cosSpec.WithGroup(cod.Name).
		WithRevision(revision).
		WithLifecycleState(orbv1alpha1.LifecycleStateActive)

	name := fmt.Sprintf("%s-%d", cod.Name, revision)
	cos := cosac.ClusterObjectSet(name).
		WithLabels(labels).
		WithAnnotations(maps.Clone(cod.Spec.Template.Metadata.Annotations)).
		WithSpec(&cosSpec)

	SetControllerReference(cod, cos)
	return cos, nil
}

func SetControllerReference(cod *orbv1alpha1.ClusterObjectDeployment, cos *cosac.ClusterObjectSetApplyConfiguration) {
	gvk := orbv1alpha1.GroupVersion.WithKind("ClusterObjectDeployment")
	cos.WithOwnerReferences(metav1ac.OwnerReference().
		WithAPIVersion(gvk.GroupVersion().String()).
		WithKind(gvk.Kind).
		WithName(cod.Name).
		WithUID(cod.UID).
		WithController(true).
		WithBlockOwnerDeletion(true),
	)
}

const LabelTemplateHash = "orb.operatorframework.io/template-hash"

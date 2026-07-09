package controller

import (
	"context"
	"fmt"

	"sigs.k8s.io/controller-runtime/pkg/client"

	orbv1alpha1 "github.com/joelanford/orb-operator/api/v1alpha1"
	cosac "github.com/joelanford/orb-operator/applyconfigurations/api/v1alpha1"
)

func applyCOS(ctx context.Context, c client.Client, cos *orbv1alpha1.ClusterObjectSet, fieldOwner string, needsApply func(*orbv1alpha1.ClusterObjectSet) bool, mutate func(*cosac.ClusterObjectSetApplyConfiguration)) (bool, error) {
	if !needsApply(cos) {
		return false, nil
	}
	ac, err := cosac.ExtractClusterObjectSet(cos, fieldOwner)
	if err != nil {
		return false, fmt.Errorf("extracting apply config: %w", err)
	}
	mutate(ac)
	return true, c.Apply(ctx, ac, client.FieldOwner(fieldOwner), client.ForceOwnership)
}

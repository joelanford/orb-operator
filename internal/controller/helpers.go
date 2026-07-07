package controller

import (
	"context"
	"fmt"

	orbv1alpha1 "github.com/joelanford/orb-operator/api/v1alpha1"
	cosrac "github.com/joelanford/orb-operator/applyconfigurations/api/v1alpha1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func applyCOSR(ctx context.Context, c client.Client, cosr *orbv1alpha1.ClusterObjectSetRevision, fieldOwner string, needsApply func(*orbv1alpha1.ClusterObjectSetRevision) bool, mutate func(*cosrac.ClusterObjectSetRevisionApplyConfiguration)) (bool, error) {
	if !needsApply(cosr) {
		return false, nil
	}
	ac, err := cosrac.ExtractClusterObjectSetRevision(cosr, fieldOwner)
	if err != nil {
		return false, fmt.Errorf("extracting apply config: %w", err)
	}
	mutate(ac)
	return true, c.Apply(ctx, ac, client.FieldOwner(fieldOwner), client.ForceOwnership)
}

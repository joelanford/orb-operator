package cosutil

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	orbv1alpha1 "github.com/joelanford/orb-operator/api/v1alpha1"
	cosac "github.com/joelanford/orb-operator/applyconfigurations/api/v1alpha1"
)

func Apply(ctx context.Context, c client.Client, cos *orbv1alpha1.ClusterObjectSet, fieldOwner string, needsApply func(*orbv1alpha1.ClusterObjectSet) bool, mutate func(*cosac.ClusterObjectSetApplyConfiguration)) (bool, error) {
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

func RemoveFinalizer(ctx context.Context, c client.Client, cos *orbv1alpha1.ClusterObjectSet, fieldOwner, finalizer string) error {
	if !controllerutil.ContainsFinalizer(cos, finalizer) {
		return nil
	}
	patch := client.MergeFromWithOptions(cos.DeepCopy(), client.MergeFromWithOptimisticLock{})
	controllerutil.RemoveFinalizer(cos, finalizer)
	ClearFinalizerFieldOwnership(cos.ManagedFields, fieldOwner, finalizer)
	return c.Patch(ctx, cos, patch)
}

func WaitForFinalizerRemoval(ctx context.Context, c client.Client, key client.ObjectKey, finalizer string) error {
	return wait.PollUntilContextTimeout(ctx, 50*time.Millisecond, 5*time.Second, true, func(ctx context.Context) (bool, error) {
		var cos orbv1alpha1.ClusterObjectSet
		if err := c.Get(ctx, key, &cos); err != nil {
			return apierrors.IsNotFound(err), client.IgnoreNotFound(err)
		}
		return !controllerutil.ContainsFinalizer(&cos, finalizer), nil
	})
}

func ClearFinalizerFieldOwnership(managedFields []metav1.ManagedFieldsEntry, manager, finalizer string) {
	key := "v:" + finalizer
	for i := range managedFields {
		e := &managedFields[i]
		if e.Manager != manager || e.FieldsV1 == nil {
			continue
		}
		var fields map[string]any
		if err := json.Unmarshal(e.FieldsV1.GetRawBytes(), &fields); err != nil {
			continue
		}
		fMeta, _ := fields["f:metadata"].(map[string]any)
		if fMeta == nil {
			continue
		}
		fFinalizers, _ := fMeta["f:finalizers"].(map[string]any)
		if fFinalizers == nil {
			continue
		}
		delete(fFinalizers, key)
		if len(fFinalizers) == 0 {
			delete(fMeta, "f:finalizers")
		}
		if len(fMeta) == 0 {
			delete(fields, "f:metadata")
		}
		raw, err := json.Marshal(fields)
		if err != nil {
			continue
		}
		e.FieldsV1.SetRawBytes(raw)
	}
}

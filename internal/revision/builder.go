package revision

import (
	"fmt"

	"pkg.package-operator.run/boxcutter"
	"pkg.package-operator.run/boxcutter/probing"
	"sigs.k8s.io/controller-runtime/pkg/client"

	orbv1alpha1 "github.com/joelanford/orb-operator/api/v1alpha1"
	"github.com/joelanford/orb-operator/internal/assertions"
	"github.com/joelanford/orb-operator/internal/object"
)

func Build(
	cos *orbv1alpha1.ClusterObjectSet,
	resolved *object.Result,
	siblings []*orbv1alpha1.ClusterObjectSet,
	ownerStrategy boxcutter.OwnerStrategy,
) boxcutter.Revision {
	phases := make([]boxcutter.Phase, 0, len(resolved.Phases))

	for _, rp := range resolved.Phases {
		objects := make([]client.Object, 0, len(rp.Objects))
		var phaseReconcileOpts []boxcutter.PhaseReconcileOption

		if rp.CollisionProtection != nil {
			phaseReconcileOpts = append(phaseReconcileOpts, MapCollisionProtection(*rp.CollisionProtection))
		}

		for _, ro := range rp.Objects {
			objects = append(objects, ro.Obj)

			var objOpts []boxcutter.ObjectReconcileOption

			probe, err := assertions.ProbeForAssertions(ro.Assertions)
			if err != nil {
				probe = boxcutter.ProbeFunc(func(_ client.Object) probing.Result {
					return probing.FalseResult(fmt.Sprintf("invalid assertion: %v", err))
				})
			}
			if probe != nil {
				objOpts = append(objOpts, boxcutter.WithProbe(boxcutter.ProgressProbeType, probe))
			}

			if ro.CollisionProtection != nil {
				objOpts = append(objOpts, MapCollisionProtection(*ro.CollisionProtection))
			}

			if len(objOpts) > 0 {
				phaseReconcileOpts = append(phaseReconcileOpts,
					boxcutter.WithObjectReconcileOptions(ro.Obj, objOpts...),
				)
			}
		}

		phase := boxcutter.NewPhaseWithOwner(rp.Name, objects, cos, ownerStrategy)
		if len(phaseReconcileOpts) > 0 {
			phase.WithReconcileOptions(phaseReconcileOpts...)
		}
		phases = append(phases, phase)
	}

	var reconcileOpts []boxcutter.RevisionReconcileOption

	if cos.Spec.CollisionProtection != nil {
		reconcileOpts = append(reconcileOpts, MapCollisionProtection(*cos.Spec.CollisionProtection))
	} else {
		reconcileOpts = append(reconcileOpts, MapCollisionProtection(orbv1alpha1.CollisionProtectionPrevent))
	}

	if len(siblings) > 0 {
		siblingObjs := make([]client.Object, 0, len(siblings))
		for _, s := range siblings {
			siblingObjs = append(siblingObjs, s)
		}
		reconcileOpts = append(reconcileOpts, boxcutter.WithSiblingOwners(siblingObjs))
	}

	rev := boxcutter.NewRevisionWithOwner(
		cos.Name,
		int64(cos.Spec.Revision),
		phases,
		cos,
		ownerStrategy,
	)
	if len(reconcileOpts) > 0 {
		rev.WithReconcileOptions(reconcileOpts...)
	}
	return rev
}

func MapCollisionProtection(cp orbv1alpha1.CollisionProtection) boxcutter.WithCollisionProtection {
	switch cp {
	case orbv1alpha1.CollisionProtectionIfNoController:
		return boxcutter.WithCollisionProtection(boxcutter.CollisionProtectionIfNoController)
	case orbv1alpha1.CollisionProtectionNone:
		return boxcutter.WithCollisionProtection(boxcutter.CollisionProtectionNone)
	default:
		return boxcutter.WithCollisionProtection(boxcutter.CollisionProtectionPrevent)
	}
}

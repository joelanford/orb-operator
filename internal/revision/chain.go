package revision

import (
	"cmp"
	"slices"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	orbv1alpha1 "github.com/joelanford/orb-operator/api/v1alpha1"
)

type Chain struct {
	LatestActive *orbv1alpha1.ClusterObjectSet
	Predecessors []*orbv1alpha1.ClusterObjectSet
	Archived     []*orbv1alpha1.ClusterObjectSet
	Deleted      []*orbv1alpha1.ClusterObjectSet
}

func BuildChain(members []orbv1alpha1.ClusterObjectSet) Chain {
	slices.SortFunc(members, func(a, b orbv1alpha1.ClusterObjectSet) int {
		return cmp.Compare(b.Spec.Revision, a.Spec.Revision)
	})

	var ch Chain
	for i := range members {
		m := &members[i]
		switch {
		case !m.DeletionTimestamp.IsZero():
			ch.Deleted = append(ch.Deleted, m)
		case m.Spec.LifecycleState == orbv1alpha1.LifecycleStateArchived:
			ch.Archived = append(ch.Archived, m)
		case ch.LatestActive == nil:
			ch.LatestActive = m
		default:
			ch.Predecessors = append(ch.Predecessors, m)
		}
	}
	return ch
}

func (ch Chain) SiblingsOf(cos *orbv1alpha1.ClusterObjectSet) []*orbv1alpha1.ClusterObjectSet {
	var siblings []*orbv1alpha1.ClusterObjectSet
	if ch.LatestActive != nil && ch.LatestActive.Name != cos.Name {
		siblings = append(siblings, ch.LatestActive)
	}
	for _, p := range ch.Predecessors {
		if p.Name != cos.Name {
			siblings = append(siblings, p)
		}
	}
	return siblings
}

type controllerOwnerKey struct {
	Kind string
	Name string
}

func controllerOwnerKeyOf(cos *orbv1alpha1.ClusterObjectSet) controllerOwnerKey {
	ref := metav1.GetControllerOf(cos)
	if ref == nil {
		return controllerOwnerKey{}
	}
	return controllerOwnerKey{Kind: ref.Kind, Name: ref.Name}
}

func FilterByOwner(members []orbv1alpha1.ClusterObjectSet, cos *orbv1alpha1.ClusterObjectSet) []orbv1alpha1.ClusterObjectSet {
	key := controllerOwnerKeyOf(cos)
	var result []orbv1alpha1.ClusterObjectSet
	for _, m := range members {
		if controllerOwnerKeyOf(&m) == key {
			result = append(result, m)
		}
	}
	return result
}

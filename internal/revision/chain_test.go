package revision

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	orbv1alpha1 "github.com/joelanford/orb-operator/api/v1alpha1"
)

func TestBuildChain(t *testing.T) {
	t.Run("empty members", func(t *testing.T) {
		ch := BuildChain(nil)
		assert.Nil(t, ch.LatestActive)
		assert.Empty(t, ch.Predecessors)
		assert.Empty(t, ch.Archived)
		assert.Empty(t, ch.Deleted)
	})

	t.Run("single active", func(t *testing.T) {
		members := []orbv1alpha1.ClusterObjectSet{
			makeCOS("cos-1", 1, orbv1alpha1.LifecycleStateActive, false),
		}
		ch := BuildChain(members)
		require.NotNil(t, ch.LatestActive)
		assert.Equal(t, "cos-1", ch.LatestActive.Name)
		assert.Empty(t, ch.Predecessors)
	})

	t.Run("multiple active sorted by revision descending", func(t *testing.T) {
		members := []orbv1alpha1.ClusterObjectSet{
			makeCOS("cos-1", 1, orbv1alpha1.LifecycleStateActive, false),
			makeCOS("cos-3", 3, orbv1alpha1.LifecycleStateActive, false),
			makeCOS("cos-2", 2, orbv1alpha1.LifecycleStateActive, false),
		}
		ch := BuildChain(members)
		require.NotNil(t, ch.LatestActive)
		assert.Equal(t, "cos-3", ch.LatestActive.Name)
		require.Len(t, ch.Predecessors, 2)
	})

	t.Run("categorizes archived and deleted", func(t *testing.T) {
		members := []orbv1alpha1.ClusterObjectSet{
			makeCOS("active", 3, orbv1alpha1.LifecycleStateActive, false),
			makeCOS("archived", 1, orbv1alpha1.LifecycleStateArchived, false),
			makeCOS("deleted", 2, orbv1alpha1.LifecycleStateActive, true),
		}
		ch := BuildChain(members)
		require.NotNil(t, ch.LatestActive)
		assert.Equal(t, "active", ch.LatestActive.Name)
		require.Len(t, ch.Archived, 1)
		assert.Equal(t, "archived", ch.Archived[0].Name)
		require.Len(t, ch.Deleted, 1)
		assert.Equal(t, "deleted", ch.Deleted[0].Name)
	})
}

func TestChain_SiblingsOf(t *testing.T) {
	t.Run("excludes self", func(t *testing.T) {
		cos1 := makeCOS("cos-1", 1, orbv1alpha1.LifecycleStateActive, false)
		cos2 := makeCOS("cos-2", 2, orbv1alpha1.LifecycleStateActive, false)
		ch := BuildChain([]orbv1alpha1.ClusterObjectSet{cos1, cos2})

		siblings := ch.SiblingsOf(&cos2)
		require.Len(t, siblings, 1)
		assert.Equal(t, "cos-1", siblings[0].Name)
	})

	t.Run("returns nil when alone", func(t *testing.T) {
		cos1 := makeCOS("cos-1", 1, orbv1alpha1.LifecycleStateActive, false)
		ch := BuildChain([]orbv1alpha1.ClusterObjectSet{cos1})

		siblings := ch.SiblingsOf(&cos1)
		assert.Empty(t, siblings)
	})
}

func TestFilterByOwner(t *testing.T) {
	t.Run("filters by matching controller owner", func(t *testing.T) {
		owned := makeCOS("owned", 1, orbv1alpha1.LifecycleStateActive, false)
		owned.OwnerReferences = []metav1.OwnerReference{{
			Kind:       "ClusterObjectDeployment",
			Name:       "my-cod",
			Controller: boolPtr(true),
		}}

		orphan := makeCOS("orphan", 2, orbv1alpha1.LifecycleStateActive, false)

		otherOwned := makeCOS("other", 3, orbv1alpha1.LifecycleStateActive, false)
		otherOwned.OwnerReferences = []metav1.OwnerReference{{
			Kind:       "ClusterObjectDeployment",
			Name:       "other-cod",
			Controller: boolPtr(true),
		}}

		result := FilterByOwner([]orbv1alpha1.ClusterObjectSet{owned, orphan, otherOwned}, &owned)
		require.Len(t, result, 1)
		assert.Equal(t, "owned", result[0].Name)
	})

	t.Run("groups orphans together", func(t *testing.T) {
		o1 := makeCOS("o1", 1, orbv1alpha1.LifecycleStateActive, false)
		o2 := makeCOS("o2", 2, orbv1alpha1.LifecycleStateActive, false)

		result := FilterByOwner([]orbv1alpha1.ClusterObjectSet{o1, o2}, &o1)
		assert.Len(t, result, 2)
	})
}

func makeCOS(name string, rev uint32, state orbv1alpha1.LifecycleState, deleted bool) orbv1alpha1.ClusterObjectSet {
	cos := orbv1alpha1.ClusterObjectSet{}
	cos.Name = name
	cos.Spec.Revision = rev
	cos.Spec.LifecycleState = state
	if deleted {
		now := metav1.NewTime(time.Now())
		cos.DeletionTimestamp = &now
	}
	return cos
}

func boolPtr(b bool) *bool { return &b }

package cos

import (
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"pkg.package-operator.run/boxcutter/machinery"
	boxcuttertypes "pkg.package-operator.run/boxcutter/machinery/types"
	"pkg.package-operator.run/boxcutter/validation"

	orbv1alpha1 "github.com/joelanford/orb-operator/api/v1alpha1"
	orberrors "github.com/joelanford/orb-operator/internal/errors"
)

func TestApply(t *testing.T) {
	t.Run("sets condition with observed generation", func(t *testing.T) {
		cos := &orbv1alpha1.ClusterObjectSet{}
		cos.Generation = 5
		u := Update{
			Condition: newCondition(metav1.ConditionTrue, "Available", "ok"),
		}
		Apply(cos, u)
		require.Len(t, cos.Status.Conditions, 1)
		assert.Equal(t, int64(5), cos.Status.Conditions[0].ObservedGeneration)
	})

	t.Run("sets observed phases when non-nil", func(t *testing.T) {
		cos := &orbv1alpha1.ClusterObjectSet{}
		phases := []orbv1alpha1.ObservedPhase{{Name: "p1", Status: orbv1alpha1.PhaseStatusAvailable}}
		u := Update{
			Condition:      newCondition(metav1.ConditionTrue, "Available", "ok"),
			ObservedPhases: &phases,
		}
		Apply(cos, u)
		assert.Equal(t, phases, cos.Status.ObservedPhases)
	})

	t.Run("does not touch phases when nil", func(t *testing.T) {
		existing := []orbv1alpha1.ObservedPhase{{Name: "p1"}}
		cos := &orbv1alpha1.ClusterObjectSet{}
		cos.Status.ObservedPhases = existing
		u := Update{
			Condition: newCondition(metav1.ConditionTrue, "Available", "ok"),
		}
		Apply(cos, u)
		assert.Equal(t, existing, cos.Status.ObservedPhases)
	})

	t.Run("sets completed at when not already set", func(t *testing.T) {
		cos := &orbv1alpha1.ClusterObjectSet{}
		now := metav1.Now()
		u := Update{
			Condition:   newCondition(metav1.ConditionTrue, "Available", "ok"),
			CompletedAt: &now,
		}
		Apply(cos, u)
		assert.Equal(t, &now, cos.Status.CompletedAt)
	})

	t.Run("preserves existing completed at", func(t *testing.T) {
		earlier := metav1.NewTime(time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC))
		cos := &orbv1alpha1.ClusterObjectSet{}
		cos.Status.CompletedAt = &earlier
		now := metav1.Now()
		u := Update{
			Condition:   newCondition(metav1.ConditionTrue, "Available", "ok"),
			CompletedAt: &now,
		}
		Apply(cos, u)
		assert.Equal(t, &earlier, cos.Status.CompletedAt)
	})
}

func TestFromReconcile(t *testing.T) {
	now := time.Date(2025, 1, 1, 12, 0, 0, 0, time.UTC)

	t.Run("resolution error with no hash", func(t *testing.T) {
		cos := &orbv1alpha1.ClusterObjectSet{}
		err := &orberrors.ObjectResolutionError{Err: fmt.Errorf("slice missing")}
		u := FromReconcile(cos, nil, err, now)
		assert.Equal(t, metav1.ConditionFalse, u.Condition.Status)
		assert.Equal(t, orbv1alpha1.ReasonInvalidRevision, u.Condition.Reason)
		assert.Nil(t, u.ObservedPhases)
	})

	t.Run("resolution error with existing hash", func(t *testing.T) {
		cos := &orbv1alpha1.ClusterObjectSet{}
		cos.Status.ResolvedContentHash = "abc"
		err := &orberrors.ObjectResolutionError{Err: fmt.Errorf("hash mismatch")}
		u := FromReconcile(cos, nil, err, now)
		assert.Equal(t, metav1.ConditionUnknown, u.Condition.Status)
		assert.Equal(t, orbv1alpha1.ReasonInvalidRevision, u.Condition.Reason)
	})

	t.Run("internal error clears phases", func(t *testing.T) {
		cos := &orbv1alpha1.ClusterObjectSet{}
		err := &orberrors.InternalError{Err: fmt.Errorf("engine setup")}
		u := FromReconcile(cos, nil, err, now)
		assert.Equal(t, metav1.ConditionUnknown, u.Condition.Status)
		assert.Equal(t, orbv1alpha1.ReasonInternalError, u.Condition.Reason)
		require.NotNil(t, u.ObservedPhases)
		assert.Empty(t, *u.ObservedPhases)
	})

	t.Run("plain error sets reconcile error", func(t *testing.T) {
		cos := &orbv1alpha1.ClusterObjectSet{
			Spec: cosSpecWithPhases("p1"),
		}
		result := &fakeRevisionResult{
			phases: []machinery.PhaseResult{
				&fakePhaseResult{name: "p1", complete: false},
			},
		}
		u := FromReconcile(cos, result, fmt.Errorf("reconcile failed"), now)
		assert.Equal(t, metav1.ConditionUnknown, u.Condition.Status)
		assert.Equal(t, orbv1alpha1.ReasonReconcileError, u.Condition.Reason)
		require.NotNil(t, u.ObservedPhases)
	})

	t.Run("validation error", func(t *testing.T) {
		cos := &orbv1alpha1.ClusterObjectSet{
			Spec: cosSpecWithPhases("p1"),
		}
		result := &fakeRevisionResult{
			validationError: &validation.RevisionValidationError{
				Phases: []validation.PhaseValidationError{
					{PhaseName: "p1", PhaseError: fmt.Errorf("bad")},
				},
			},
		}
		u := FromReconcile(cos, result, nil, now)
		assert.Equal(t, metav1.ConditionFalse, u.Condition.Status)
		assert.Equal(t, orbv1alpha1.ReasonInvalidRevision, u.Condition.Reason)
	})

	t.Run("progressed", func(t *testing.T) {
		cos := &orbv1alpha1.ClusterObjectSet{
			Spec: cosSpecWithPhases("p1"),
		}
		result := &fakeRevisionResult{
			progressed: true,
			phases: []machinery.PhaseResult{
				&fakePhaseResult{name: "p1", complete: true},
			},
		}
		u := FromReconcile(cos, result, nil, now)
		assert.Equal(t, metav1.ConditionFalse, u.Condition.Status)
		assert.Equal(t, orbv1alpha1.ReasonSuperseded, u.Condition.Reason)
		assert.Nil(t, u.CompletedAt)
	})

	t.Run("complete sets available and completed at", func(t *testing.T) {
		cos := &orbv1alpha1.ClusterObjectSet{
			Spec: cosSpecWithPhases("p1"),
		}
		result := &fakeRevisionResult{
			complete: true,
			phases: []machinery.PhaseResult{
				&fakePhaseResult{name: "p1", complete: true},
			},
		}
		u := FromReconcile(cos, result, nil, now)
		assert.Equal(t, metav1.ConditionTrue, u.Condition.Status)
		assert.Equal(t, orbv1alpha1.ReasonAvailable, u.Condition.Reason)
		require.NotNil(t, u.CompletedAt)
	})

	t.Run("in progress", func(t *testing.T) {
		cos := &orbv1alpha1.ClusterObjectSet{
			Spec: cosSpecWithPhases("p1"),
		}
		result := &fakeRevisionResult{
			phases: []machinery.PhaseResult{
				&fakePhaseResult{name: "p1", complete: false},
			},
		}
		u := FromReconcile(cos, result, nil, now)
		assert.Equal(t, metav1.ConditionFalse, u.Condition.Status)
		assert.Equal(t, orbv1alpha1.ReasonUnavailable, u.Condition.Reason)
		assert.Nil(t, u.CompletedAt)
	})
}

func TestFromTeardown(t *testing.T) {
	now := time.Date(2025, 1, 1, 12, 0, 0, 0, time.UTC)

	t.Run("resolution error", func(t *testing.T) {
		cos := &orbv1alpha1.ClusterObjectSet{}
		err := &orberrors.ObjectResolutionError{Err: fmt.Errorf("missing")}
		u := FromTeardown(cos, nil, err, now)
		assert.Equal(t, metav1.ConditionFalse, u.Condition.Status)
		assert.Equal(t, orbv1alpha1.ReasonInvalidRevision, u.Condition.Reason)
	})

	t.Run("internal error", func(t *testing.T) {
		cos := &orbv1alpha1.ClusterObjectSet{}
		err := &orberrors.InternalError{Err: fmt.Errorf("engine")}
		u := FromTeardown(cos, nil, err, now)
		assert.Equal(t, metav1.ConditionUnknown, u.Condition.Status)
		assert.Equal(t, orbv1alpha1.ReasonInternalError, u.Condition.Reason)
	})

	t.Run("teardown error", func(t *testing.T) {
		cos := &orbv1alpha1.ClusterObjectSet{
			Spec: cosSpecWithPhases("p1"),
		}
		result := &fakeTeardownResult{
			phases: []machinery.PhaseTeardownResult{
				&fakePhaseTeardownResult{name: "p1", complete: false},
			},
		}
		u := FromTeardown(cos, result, fmt.Errorf("teardown failed"), now)
		assert.Equal(t, metav1.ConditionUnknown, u.Condition.Status)
		assert.Equal(t, orbv1alpha1.ReasonTeardownError, u.Condition.Reason)
	})

	t.Run("teardown in progress", func(t *testing.T) {
		cos := &orbv1alpha1.ClusterObjectSet{
			Spec: cosSpecWithPhases("p1"),
		}
		result := &fakeTeardownResult{
			complete: false,
			phases: []machinery.PhaseTeardownResult{
				&fakePhaseTeardownResult{name: "p1", complete: false},
			},
		}
		u := FromTeardown(cos, result, nil, now)
		assert.Equal(t, metav1.ConditionFalse, u.Condition.Status)
		assert.Equal(t, orbv1alpha1.ReasonArchived, u.Condition.Reason)
		assert.Contains(t, u.Condition.Message, "in progress")
	})

	t.Run("teardown complete", func(t *testing.T) {
		cos := &orbv1alpha1.ClusterObjectSet{
			Spec: cosSpecWithPhases("p1"),
		}
		result := &fakeTeardownResult{
			complete: true,
			phases: []machinery.PhaseTeardownResult{
				&fakePhaseTeardownResult{name: "p1", complete: true},
			},
		}
		u := FromTeardown(cos, result, nil, now)
		assert.Equal(t, metav1.ConditionFalse, u.Condition.Status)
		assert.Equal(t, orbv1alpha1.ReasonArchived, u.Condition.Reason)
		assert.Contains(t, u.Condition.Message, "teardown complete")
	})
}

func TestBuildObservedPhases(t *testing.T) {
	specPhases := []orbv1alpha1.Phase{
		{Name: "phase-1"},
		{Name: "phase-2"},
		{Name: "phase-3"},
	}

	t.Run("all phases unknown with waiting message when no results", func(t *testing.T) {
		result := buildObservedPhases(specPhases, nil)
		require.Len(t, result, 3)
		for _, op := range result {
			assert.Equal(t, orbv1alpha1.PhaseStatusUnknown, op.Status)
			assert.Equal(t, "Waiting for earlier phases to complete", op.Error)
		}
	})

	t.Run("complete phase", func(t *testing.T) {
		results := []machinery.PhaseResult{
			&fakePhaseResult{name: "phase-1", complete: true},
		}
		observed := buildObservedPhases(specPhases, results)
		require.Len(t, observed, 3)
		assert.Equal(t, orbv1alpha1.PhaseStatusAvailable, observed[0].Status)
		assert.Equal(t, orbv1alpha1.PhaseStatusUnknown, observed[1].Status)
	})

	t.Run("reconciling phase with incomplete objects", func(t *testing.T) {
		obj := &unstructured.Unstructured{}
		obj.SetGroupVersionKind(corev1.SchemeGroupVersion.WithKind("ConfigMap"))
		obj.SetName("my-cm")
		obj.SetNamespace("default")

		results := []machinery.PhaseResult{
			&fakePhaseResult{name: "phase-1", complete: true},
			&fakePhaseResult{
				name:     "phase-2",
				complete: false,
				objects: []machinery.ObjectResult{
					&fakeObjectResult{obj: obj, complete: false, probes: boxcuttertypes.ProbeResultContainer{
						boxcuttertypes.ProgressProbeType: {Status: boxcuttertypes.ProbeStatusFalse, Messages: []string{"condition Available is not True"}},
					}},
				},
			},
		}
		observed := buildObservedPhases(specPhases, results)
		assert.Equal(t, orbv1alpha1.PhaseStatusReconciling, observed[1].Status)
		require.Len(t, observed[1].IncompleteObjects, 1)
		assert.Equal(t, "my-cm", observed[1].IncompleteObjects[0].Name)
	})

	t.Run("collision produces message", func(t *testing.T) {
		obj := &unstructured.Unstructured{}
		obj.SetGroupVersionKind(corev1.SchemeGroupVersion.WithKind("ConfigMap"))
		obj.SetName("collided-cm")

		results := []machinery.PhaseResult{
			&fakePhaseResult{
				name:     "phase-1",
				complete: false,
				objects: []machinery.ObjectResult{
					&fakeCollisionResult{
						fakeObjectResult: fakeObjectResult{obj: obj, complete: false, probes: boxcuttertypes.ProbeResultContainer{}},
					},
				},
			},
		}
		observed := buildObservedPhases(specPhases, results)
		assert.Contains(t, observed[0].IncompleteObjects[0].Messages, "object ownership collision")
	})
}

func TestPreserveCompletionTimes(t *testing.T) {
	now := time.Date(2025, 1, 1, 12, 0, 0, 0, time.UTC)
	earlier := metav1.NewTime(now.Add(-time.Hour))

	t.Run("sets completedAt on first Available", func(t *testing.T) {
		current := []orbv1alpha1.ObservedPhase{
			{Name: "phase-1", Status: orbv1alpha1.PhaseStatusAvailable},
		}
		PreserveCompletionTimes(nil, current, now)
		require.NotNil(t, current[0].CompletedAt)
		assert.Equal(t, metav1.NewTime(now), *current[0].CompletedAt)
	})

	t.Run("preserves existing completedAt", func(t *testing.T) {
		existing := []orbv1alpha1.ObservedPhase{
			{Name: "phase-1", CompletedAt: &earlier},
		}
		current := []orbv1alpha1.ObservedPhase{
			{Name: "phase-1", Status: orbv1alpha1.PhaseStatusAvailable},
		}
		PreserveCompletionTimes(existing, current, now)
		assert.Equal(t, earlier, *current[0].CompletedAt)
	})
}

func TestTruncateMessage(t *testing.T) {
	t.Run("short message unchanged", func(t *testing.T) {
		assert.Equal(t, "hello", truncateMessage("hello"))
	})

	t.Run("over maxMessageLength truncated with ellipsis", func(t *testing.T) {
		s := strings.Repeat("a", maxMessageLength+10)
		result := truncateMessage(s)
		assert.Len(t, []rune(result), maxMessageLength)
		assert.True(t, strings.HasSuffix(result, "..."))
	})
}

// Test fakes

type fakeRevisionResult struct {
	validationError *validation.RevisionValidationError
	phases          []machinery.PhaseResult
	progressed      bool
	complete        bool
}

func (f *fakeRevisionResult) GetValidationError() *validation.RevisionValidationError {
	return f.validationError
}
func (f *fakeRevisionResult) GetPhases() []machinery.PhaseResult { return f.phases }
func (f *fakeRevisionResult) InTransition() bool                 { return false }
func (f *fakeRevisionResult) IsComplete() bool                   { return f.complete }
func (f *fakeRevisionResult) HasProgressed() bool                { return f.progressed }
func (f *fakeRevisionResult) String() string                     { return "" }

type fakePhaseResult struct {
	name            string
	complete        bool
	objects         []machinery.ObjectResult
	validationError *validation.PhaseValidationError
}

func (f *fakePhaseResult) GetName() string                      { return f.name }
func (f *fakePhaseResult) IsComplete() bool                     { return f.complete }
func (f *fakePhaseResult) GetObjects() []machinery.ObjectResult { return f.objects }
func (f *fakePhaseResult) InTransition() bool                   { return false }
func (f *fakePhaseResult) HasProgressed() bool                  { return false }
func (f *fakePhaseResult) String() string                       { return f.name }
func (f *fakePhaseResult) GetValidationError() *validation.PhaseValidationError {
	return f.validationError
}

type fakeObjectResult struct {
	obj      machinery.Object
	complete bool
	probes   boxcuttertypes.ProbeResultContainer
	action   machinery.Action
}

func (f *fakeObjectResult) Object() machinery.Object                          { return f.obj }
func (f *fakeObjectResult) IsComplete() bool                                  { return f.complete }
func (f *fakeObjectResult) IsPaused() bool                                    { return false }
func (f *fakeObjectResult) ProbeResults() boxcuttertypes.ProbeResultContainer { return f.probes }
func (f *fakeObjectResult) Action() machinery.Action                          { return f.action }
func (f *fakeObjectResult) String() string                                    { return "" }

type fakeCollisionResult struct {
	fakeObjectResult
}

func (f *fakeCollisionResult) Action() machinery.Action { return machinery.ActionCollision }

type fakeTeardownResult struct {
	complete bool
	phases   []machinery.PhaseTeardownResult
}

func (f *fakeTeardownResult) IsComplete() bool                           { return f.complete }
func (f *fakeTeardownResult) GetPhases() []machinery.PhaseTeardownResult { return f.phases }
func (f *fakeTeardownResult) GetActivePhaseName() (string, bool)         { return "", false }
func (f *fakeTeardownResult) GetWaitingPhaseNames() []string             { return nil }
func (f *fakeTeardownResult) GetGonePhaseNames() []string                { return nil }
func (f *fakeTeardownResult) String() string                             { return "" }

type fakePhaseTeardownResult struct {
	name     string
	complete bool
	waiting  []boxcuttertypes.ObjectRef
}

func (f *fakePhaseTeardownResult) GetName() string                     { return f.name }
func (f *fakePhaseTeardownResult) IsComplete() bool                    { return f.complete }
func (f *fakePhaseTeardownResult) Gone() []boxcuttertypes.ObjectRef    { return nil }
func (f *fakePhaseTeardownResult) Waiting() []boxcuttertypes.ObjectRef { return f.waiting }
func (f *fakePhaseTeardownResult) String() string                      { return f.name }

func cosSpecWithPhases(names ...string) orbv1alpha1.ClusterObjectSetSpec {
	phases := make([]orbv1alpha1.Phase, len(names))
	for i, n := range names {
		phases[i] = orbv1alpha1.Phase{Name: n}
	}
	return orbv1alpha1.ClusterObjectSetSpec{
		ClusterObjectDeploymentTemplateSpec: orbv1alpha1.ClusterObjectDeploymentTemplateSpec{
			Phases: phases,
		},
	}
}

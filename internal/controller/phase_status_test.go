package controller

import (
	"fmt"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/types"
	"pkg.package-operator.run/boxcutter/machinery"
	boxcuttertypes "pkg.package-operator.run/boxcutter/machinery/types"
	"pkg.package-operator.run/boxcutter/validation"

	orbv1alpha1 "github.com/joelanford/orb-operator/api/v1alpha1"
)

func TestBuildObservedPhases(t *testing.T) {
	specPhases := []orbv1alpha1.Phase{
		{Name: "phase-1"},
		{Name: "phase-2"},
		{Name: "phase-3"},
	}

	t.Run("all phases unknown when no results", func(t *testing.T) {
		result := buildObservedPhases(specPhases, nil)
		require.Len(t, result, 3)
		for _, op := range result {
			assert.Equal(t, orbv1alpha1.PhaseStatusUnknown, op.Status)
			assert.Empty(t, op.IncompleteObjects)
		}
	})

	t.Run("complete phase", func(t *testing.T) {
		results := []machinery.PhaseResult{
			&fakePhaseResult{name: "phase-1", complete: true},
		}
		observed := buildObservedPhases(specPhases, results)
		require.Len(t, observed, 3)
		assert.Equal(t, orbv1alpha1.PhaseStatusAvailable, observed[0].Status)
		assert.Empty(t, observed[0].IncompleteObjects)
		assert.Equal(t, orbv1alpha1.PhaseStatusUnknown, observed[1].Status)
		assert.Equal(t, orbv1alpha1.PhaseStatusUnknown, observed[2].Status)
	})

	t.Run("reconciling phase with incomplete objects", func(t *testing.T) {
		obj := &unstructured.Unstructured{}
		obj.SetGroupVersionKind(corev1.SchemeGroupVersion.WithKind("ConfigMap"))
		obj.SetName("my-cm")
		obj.SetNamespace("default")

		results := []machinery.PhaseResult{
			&fakePhaseResult{
				name:     "phase-1",
				complete: true,
			},
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
		require.Len(t, observed, 3)

		assert.Equal(t, orbv1alpha1.PhaseStatusAvailable, observed[0].Status)

		assert.Equal(t, orbv1alpha1.PhaseStatusReconciling, observed[1].Status)
		require.Len(t, observed[1].IncompleteObjects, 1)
		assert.Equal(t, "ConfigMap", observed[1].IncompleteObjects[0].Kind)
		assert.Equal(t, "my-cm", observed[1].IncompleteObjects[0].Name)
		assert.Equal(t, "default", observed[1].IncompleteObjects[0].Namespace)
		assert.Contains(t, observed[1].IncompleteObjects[0].Messages[0], "condition Available is not True")

		assert.Equal(t, orbv1alpha1.PhaseStatusUnknown, observed[2].Status)
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
						fakeObjectResult: fakeObjectResult{
							obj:      obj,
							complete: false,
							probes:   boxcuttertypes.ProbeResultContainer{},
						},
						conflictingOwner: &metav1.OwnerReference{Name: "other"},
					},
				},
			},
		}
		observed := buildObservedPhases(specPhases, results)
		require.Len(t, observed[0].IncompleteObjects, 1)
		assert.Contains(t, observed[0].IncompleteObjects[0].Messages, "object ownership collision")
	})

	t.Run("validation error produces object status from object refs", func(t *testing.T) {
		verr := &validation.PhaseValidationError{
			PhaseName: "phase-1",
			Objects: []validation.ObjectValidationError{
				{
					ObjectRef: boxcuttertypes.ObjectRef{
						GroupVersionKind: corev1.SchemeGroupVersion.WithKind("ConfigMap"),
						ObjectKey:        types.NamespacedName{Namespace: "default", Name: "bad-cm"},
					},
					Errors: []error{fmt.Errorf("dry-run failed")},
				},
			},
		}
		results := []machinery.PhaseResult{
			&fakePhaseResult{
				name:            "phase-1",
				complete:        false,
				validationError: verr,
			},
		}
		observed := buildObservedPhases(specPhases, results)
		require.Len(t, observed, 3)
		assert.Equal(t, orbv1alpha1.PhaseStatusReconciling, observed[0].Status)
		require.Len(t, observed[0].IncompleteObjects, 1)
		assert.Empty(t, observed[0].IncompleteObjects[0].Group)
		assert.Equal(t, "v1", observed[0].IncompleteObjects[0].Version)
		assert.Equal(t, "ConfigMap", observed[0].IncompleteObjects[0].Kind)
		assert.Equal(t, "default", observed[0].IncompleteObjects[0].Namespace)
		assert.Equal(t, "bad-cm", observed[0].IncompleteObjects[0].Name)
		require.Len(t, observed[0].IncompleteObjects[0].Messages, 1)
		assert.Contains(t, observed[0].IncompleteObjects[0].Messages[0], "validation error: dry-run failed")
	})

	t.Run("phase-level validation error populates error field", func(t *testing.T) {
		verr := &validation.PhaseValidationError{
			PhaseName:  "phase-1",
			PhaseError: fmt.Errorf("invalid phase name"),
		}
		results := []machinery.PhaseResult{
			&fakePhaseResult{
				name:            "phase-1",
				complete:        false,
				validationError: verr,
			},
		}
		observed := buildObservedPhases(specPhases, results)
		require.Len(t, observed, 3)
		assert.Equal(t, orbv1alpha1.PhaseStatusReconciling, observed[0].Status)
		assert.Equal(t, "validation error: invalid phase name", observed[0].Error)
		assert.Empty(t, observed[0].IncompleteObjects)
	})

	t.Run("incomplete object with no probes or collision gets fallback message", func(t *testing.T) {
		obj := &unstructured.Unstructured{}
		obj.SetGroupVersionKind(corev1.SchemeGroupVersion.WithKind("ConfigMap"))
		obj.SetName("pending-cm")

		results := []machinery.PhaseResult{
			&fakePhaseResult{
				name:     "phase-1",
				complete: false,
				objects: []machinery.ObjectResult{
					&fakeObjectResult{obj: obj, complete: false, probes: boxcuttertypes.ProbeResultContainer{}},
				},
			},
		}
		observed := buildObservedPhases(specPhases, results)
		require.Len(t, observed[0].IncompleteObjects, 1)
		assert.Equal(t, []string{"not yet complete"}, observed[0].IncompleteObjects[0].Messages)
	})

	t.Run("long probe message is truncated to 1024 chars", func(t *testing.T) {
		obj := &unstructured.Unstructured{}
		obj.SetGroupVersionKind(corev1.SchemeGroupVersion.WithKind("ConfigMap"))
		obj.SetName("long-msg-cm")

		longMessage := strings.Repeat("x", 2000)
		results := []machinery.PhaseResult{
			&fakePhaseResult{
				name:     "phase-1",
				complete: false,
				objects: []machinery.ObjectResult{
					&fakeObjectResult{obj: obj, complete: false, probes: boxcuttertypes.ProbeResultContainer{
						boxcuttertypes.ProgressProbeType: {Status: boxcuttertypes.ProbeStatusFalse, Messages: []string{longMessage}},
					}},
				},
			},
		}
		observed := buildObservedPhases(specPhases, results)
		require.Len(t, observed[0].IncompleteObjects, 1)
		msg := observed[0].IncompleteObjects[0].Messages[0]
		assert.LessOrEqual(t, len(msg), 1024)
		assert.True(t, strings.HasSuffix(msg, "..."))
	})

	t.Run("long validation error is truncated to 1024 chars", func(t *testing.T) {
		longErr := strings.Repeat("e", 2000)
		verr := &validation.PhaseValidationError{
			PhaseName: "phase-1",
			Objects: []validation.ObjectValidationError{
				{
					ObjectRef: boxcuttertypes.ObjectRef{
						GroupVersionKind: corev1.SchemeGroupVersion.WithKind("ConfigMap"),
						ObjectKey:        types.NamespacedName{Namespace: "default", Name: "bad-cm"},
					},
					Errors: []error{fmt.Errorf("%s", longErr)},
				},
			},
		}
		results := []machinery.PhaseResult{
			&fakePhaseResult{
				name:            "phase-1",
				complete:        false,
				validationError: verr,
			},
		}
		observed := buildObservedPhases(specPhases, results)
		require.Len(t, observed[0].IncompleteObjects, 1)
		msg := observed[0].IncompleteObjects[0].Messages[0]
		assert.LessOrEqual(t, len(msg), 1024)
		assert.True(t, strings.HasSuffix(msg, "..."))
	})

	t.Run("all phases complete", func(t *testing.T) {
		results := []machinery.PhaseResult{
			&fakePhaseResult{name: "phase-1", complete: true},
			&fakePhaseResult{name: "phase-2", complete: true},
			&fakePhaseResult{name: "phase-3", complete: true},
		}
		observed := buildObservedPhases(specPhases, results)
		require.Len(t, observed, 3)
		for _, op := range observed {
			assert.Equal(t, orbv1alpha1.PhaseStatusAvailable, op.Status)
			assert.Empty(t, op.IncompleteObjects)
		}
	})
}

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
	conflictingOwner *metav1.OwnerReference
}

func (f *fakeCollisionResult) Action() machinery.Action { return machinery.ActionCollision }

func TestBuildTeardownObservedPhases(t *testing.T) {
	specPhases := []orbv1alpha1.Phase{
		{Name: "phase-1"},
		{Name: "phase-2"},
		{Name: "phase-3"},
	}

	t.Run("all phases unknown when no results", func(t *testing.T) {
		result := buildTeardownObservedPhases(specPhases, nil)
		require.Len(t, result, 3)
		for _, op := range result {
			assert.Equal(t, orbv1alpha1.PhaseStatusUnknown, op.Status)
			assert.Empty(t, op.IncompleteObjects)
		}
	})

	t.Run("teardown complete phase", func(t *testing.T) {
		results := []machinery.PhaseTeardownResult{
			&fakePhaseTeardownResult{name: "phase-1", complete: true},
		}
		observed := buildTeardownObservedPhases(specPhases, results)
		require.Len(t, observed, 3)
		assert.Equal(t, orbv1alpha1.PhaseStatusTeardownComplete, observed[0].Status)
		assert.Empty(t, observed[0].IncompleteObjects)
		assert.Equal(t, orbv1alpha1.PhaseStatusUnknown, observed[1].Status)
		assert.Equal(t, orbv1alpha1.PhaseStatusUnknown, observed[2].Status)
	})

	t.Run("tearing down phase with waiting objects", func(t *testing.T) {
		results := []machinery.PhaseTeardownResult{
			&fakePhaseTeardownResult{
				name:     "phase-1",
				complete: false,
				waiting: []boxcuttertypes.ObjectRef{
					{
						GroupVersionKind: corev1.SchemeGroupVersion.WithKind("ConfigMap"),
						ObjectKey:        types.NamespacedName{Namespace: "default", Name: "my-cm"},
					},
				},
			},
		}
		observed := buildTeardownObservedPhases(specPhases, results)
		require.Len(t, observed, 3)
		assert.Equal(t, orbv1alpha1.PhaseStatusTearingDown, observed[0].Status)
		require.Len(t, observed[0].IncompleteObjects, 1)
		assert.Equal(t, "ConfigMap", observed[0].IncompleteObjects[0].Kind)
		assert.Equal(t, "my-cm", observed[0].IncompleteObjects[0].Name)
		assert.Equal(t, "default", observed[0].IncompleteObjects[0].Namespace)
		assert.Equal(t, []string{"awaiting deletion"}, observed[0].IncompleteObjects[0].Messages)
	})

	t.Run("mixed complete and in-progress phases", func(t *testing.T) {
		results := []machinery.PhaseTeardownResult{
			&fakePhaseTeardownResult{name: "phase-3", complete: true},
			&fakePhaseTeardownResult{
				name:     "phase-2",
				complete: false,
				waiting: []boxcuttertypes.ObjectRef{
					{
						GroupVersionKind: corev1.SchemeGroupVersion.WithKind("Secret"),
						ObjectKey:        types.NamespacedName{Namespace: "ns", Name: "s1"},
					},
				},
			},
		}
		observed := buildTeardownObservedPhases(specPhases, results)
		require.Len(t, observed, 3)
		assert.Equal(t, orbv1alpha1.PhaseStatusUnknown, observed[0].Status)
		assert.Equal(t, orbv1alpha1.PhaseStatusTearingDown, observed[1].Status)
		require.Len(t, observed[1].IncompleteObjects, 1)
		assert.Equal(t, orbv1alpha1.PhaseStatusTeardownComplete, observed[2].Status)
	})

	t.Run("all phases teardown complete", func(t *testing.T) {
		results := []machinery.PhaseTeardownResult{
			&fakePhaseTeardownResult{name: "phase-1", complete: true},
			&fakePhaseTeardownResult{name: "phase-2", complete: true},
			&fakePhaseTeardownResult{name: "phase-3", complete: true},
		}
		observed := buildTeardownObservedPhases(specPhases, results)
		require.Len(t, observed, 3)
		for _, op := range observed {
			assert.Equal(t, orbv1alpha1.PhaseStatusTeardownComplete, op.Status)
			assert.Empty(t, op.IncompleteObjects)
		}
	})
}

type fakePhaseTeardownResult struct {
	name     string
	complete bool
	gone     []boxcuttertypes.ObjectRef
	waiting  []boxcuttertypes.ObjectRef
}

func (f *fakePhaseTeardownResult) GetName() string                     { return f.name }
func (f *fakePhaseTeardownResult) IsComplete() bool                    { return f.complete }
func (f *fakePhaseTeardownResult) Gone() []boxcuttertypes.ObjectRef    { return f.gone }
func (f *fakePhaseTeardownResult) Waiting() []boxcuttertypes.ObjectRef { return f.waiting }
func (f *fakePhaseTeardownResult) String() string                      { return f.name }

func TestTruncateMessage(t *testing.T) {
	t.Run("short message unchanged", func(t *testing.T) {
		assert.Equal(t, "hello", truncateMessage("hello"))
	})

	t.Run("exactly maxMessageLength unchanged", func(t *testing.T) {
		s := strings.Repeat("a", maxMessageLength)
		assert.Equal(t, s, truncateMessage(s))
	})

	t.Run("over maxMessageLength truncated with ellipsis", func(t *testing.T) {
		s := strings.Repeat("a", maxMessageLength+10)
		result := truncateMessage(s)
		assert.Len(t, []rune(result), maxMessageLength)
		assert.True(t, strings.HasSuffix(result, "..."))
	})

	t.Run("multi-byte runes not split", func(t *testing.T) {
		s := strings.Repeat("\U0001F600", maxMessageLength+1)
		result := truncateMessage(s)
		assert.Len(t, []rune(result), maxMessageLength)
		assert.True(t, strings.HasSuffix(result, "..."))
		assert.Equal(t, '\U0001F600', []rune(result)[0])
	})

	t.Run("maxMessageLength counts runes not bytes", func(t *testing.T) {
		s := strings.Repeat("é", maxMessageLength)
		assert.Equal(t, s, truncateMessage(s))
	})
}

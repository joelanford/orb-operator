package revision

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/openapi"
	"k8s.io/client-go/restmapper"
	"pkg.package-operator.run/boxcutter"
	"pkg.package-operator.run/boxcutter/machinery"
	"pkg.package-operator.run/boxcutter/machinery/types"
	"pkg.package-operator.run/boxcutter/validation"
	"sigs.k8s.io/controller-runtime/pkg/client"
	fakeclient "sigs.k8s.io/controller-runtime/pkg/client/fake"

	orbv1alpha1 "github.com/joelanford/orb-operator/api/v1alpha1"
)

func TestEngine_completedPhaseNames(t *testing.T) {
	earlier := metav1.Now()

	t.Run("returns completed phases", func(t *testing.T) {
		e := &Engine{
			existingOPs: []orbv1alpha1.ObservedPhase{
				{Name: "phase-1", CompletedAt: &earlier},
				{Name: "phase-2"},
				{Name: "phase-3", CompletedAt: &earlier},
			},
		}
		names := e.completedPhaseNames()
		assert.True(t, names["phase-1"])
		assert.False(t, names["phase-2"])
		assert.True(t, names["phase-3"])
	})

	t.Run("empty phases", func(t *testing.T) {
		e := &Engine{}
		names := e.completedPhaseNames()
		assert.Empty(t, names)
	})
}

func TestNewEngine(t *testing.T) {
	scheme := runtime.NewScheme()
	mapper := restmapper.NewDiscoveryRESTMapper(nil)
	fakeClient := fakeclient.NewClientBuilder().Build()
	opts := boxcutter.RevisionEngineOptions{
		Scheme:           scheme,
		FieldOwner:       "test",
		SystemPrefix:     "test.io",
		ManagedBy:        "test",
		DiscoveryClient:  &fakeDiscoveryClient{},
		RestMapper:       mapper,
		Writer:           fakeClient,
		Reader:           fakeClient,
		UnfilteredReader: fakeClient,
	}
	engine, err := NewEngine(opts, nil)
	require.NoError(t, err)
	assert.NotNil(t, engine)
}

type fakeDiscoveryClient struct{}

func (f *fakeDiscoveryClient) OpenAPIV3() openapi.Client { return nil }

func TestResult_GetValidationError(t *testing.T) {
	verr := &validation.RevisionValidationError{}
	r := &Result{gated: &fakeRevResult{validationError: verr}}
	assert.Equal(t, verr, r.GetValidationError())
}

func TestResult_GetPhases(t *testing.T) {
	gatedPhase := &fakePhaseResult{name: "gated"}
	driftPhase := &fakePhaseResult{name: "drift"}
	readOnlyPhase := &fakePhaseResult{name: "readonly"}
	r := &Result{
		gated:           &fakeRevResult{phases: []machinery.PhaseResult{gatedPhase}},
		driftResults:    []machinery.PhaseResult{driftPhase},
		readOnlyResults: []machinery.PhaseResult{readOnlyPhase},
	}
	phases := r.GetPhases()
	require.Len(t, phases, 3)
	assert.Equal(t, "gated", phases[0].GetName())
	assert.Equal(t, "drift", phases[1].GetName())
	assert.Equal(t, "readonly", phases[2].GetName())
}

func TestResult_InTransition(t *testing.T) {
	t.Run("gated in transition", func(t *testing.T) {
		r := &Result{gated: &fakeRevResult{inTransition: true}}
		assert.True(t, r.InTransition())
	})

	t.Run("drift incomplete", func(t *testing.T) {
		r := &Result{
			gated:        &fakeRevResult{},
			driftResults: []machinery.PhaseResult{&fakePhaseResult{complete: false}},
		}
		assert.True(t, r.InTransition())
	})

	t.Run("all complete", func(t *testing.T) {
		r := &Result{
			gated:        &fakeRevResult{},
			driftResults: []machinery.PhaseResult{&fakePhaseResult{complete: true}},
		}
		assert.False(t, r.InTransition())
	})
}

func TestResult_IsComplete(t *testing.T) {
	t.Run("gated incomplete", func(t *testing.T) {
		r := &Result{gated: &fakeRevResult{complete: false}}
		assert.False(t, r.IsComplete())
	})

	t.Run("drift incomplete", func(t *testing.T) {
		r := &Result{
			gated:        &fakeRevResult{complete: true},
			driftResults: []machinery.PhaseResult{&fakePhaseResult{complete: false}},
		}
		assert.False(t, r.IsComplete())
	})

	t.Run("all complete", func(t *testing.T) {
		r := &Result{
			gated:        &fakeRevResult{complete: true},
			driftResults: []machinery.PhaseResult{&fakePhaseResult{complete: true}},
		}
		assert.True(t, r.IsComplete())
	})

	t.Run("no drift results", func(t *testing.T) {
		r := &Result{gated: &fakeRevResult{complete: true}}
		assert.True(t, r.IsComplete())
	})
}

func TestResult_HasProgressed(t *testing.T) {
	r := &Result{gated: &fakeRevResult{progressed: true}}
	assert.True(t, r.HasProgressed())

	r2 := &Result{gated: &fakeRevResult{progressed: false}}
	assert.False(t, r2.HasProgressed())
}

func TestResult_String(t *testing.T) {
	r := &Result{gated: &fakeRevResult{str: "test-result"}}
	assert.Equal(t, "test-result", r.String())
}

type fakeRevResult struct {
	validationError *validation.RevisionValidationError
	phases          []machinery.PhaseResult
	complete        bool
	progressed      bool
	inTransition    bool
	str             string
}

func (f *fakeRevResult) GetValidationError() *validation.RevisionValidationError {
	return f.validationError
}
func (f *fakeRevResult) GetPhases() []machinery.PhaseResult { return f.phases }
func (f *fakeRevResult) IsComplete() bool                   { return f.complete }
func (f *fakeRevResult) HasProgressed() bool                { return f.progressed }
func (f *fakeRevResult) InTransition() bool                 { return f.inTransition }
func (f *fakeRevResult) String() string                     { return f.str }

type fakePhaseResult struct {
	name     string
	complete bool
}

func (f *fakePhaseResult) GetName() string                                      { return f.name }
func (f *fakePhaseResult) IsComplete() bool                                     { return f.complete }
func (f *fakePhaseResult) GetObjects() []machinery.ObjectResult                 { return nil }
func (f *fakePhaseResult) InTransition() bool                                   { return false }
func (f *fakePhaseResult) HasProgressed() bool                                  { return false }
func (f *fakePhaseResult) String() string                                       { return f.name }
func (f *fakePhaseResult) GetValidationError() *validation.PhaseValidationError { return nil }

type fakePhase struct {
	name string
}

func (f *fakePhase) GetName() string                                   { return f.name }
func (f *fakePhase) GetObjects() []client.Object                       { return nil }
func (f *fakePhase) GetReconcileOptions() []types.PhaseReconcileOption { return nil }
func (f *fakePhase) GetTeardownOptions() []types.PhaseTeardownOption   { return nil }

type fakeRevision struct {
	phases []types.Phase
}

func (f *fakeRevision) GetName() string                                      { return "rev" }
func (f *fakeRevision) GetRevisionNumber() int64                             { return 1 }
func (f *fakeRevision) GetPhases() []types.Phase                             { return f.phases }
func (f *fakeRevision) GetReconcileOptions() []types.RevisionReconcileOption { return nil }
func (f *fakeRevision) GetTeardownOptions() []types.RevisionTeardownOption   { return nil }

func TestSplitPhases(t *testing.T) {
	phases := func(names ...string) []types.Phase {
		p := make([]types.Phase, len(names))
		for i, n := range names {
			p[i] = &fakePhase{name: n}
		}
		return p
	}
	phaseNames := func(pp []types.Phase) []string {
		names := make([]string, len(pp))
		for i, p := range pp {
			names[i] = p.GetName()
		}
		return names
	}

	t.Run("all gated", func(t *testing.T) {
		rev := &fakeRevision{phases: phases("a", "b", "c")}
		gated := map[string]struct{}{"a": {}, "b": {}, "c": {}}
		drift, readOnly := splitPhases(rev, gated, nil)
		assert.Empty(t, drift)
		assert.Empty(t, readOnly)
	})

	t.Run("all completed non-gated", func(t *testing.T) {
		rev := &fakeRevision{phases: phases("a", "b", "c")}
		gated := map[string]struct{}{"a": {}}
		completed := map[string]bool{"b": true, "c": true}
		drift, readOnly := splitPhases(rev, gated, completed)
		assert.Equal(t, []string{"b", "c"}, phaseNames(drift))
		assert.Empty(t, readOnly)
	})

	t.Run("no completed non-gated", func(t *testing.T) {
		rev := &fakeRevision{phases: phases("a", "b", "c")}
		gated := map[string]struct{}{"a": {}}
		drift, readOnly := splitPhases(rev, gated, nil)
		assert.Empty(t, drift)
		assert.Equal(t, []string{"b", "c"}, phaseNames(readOnly))
	})

	t.Run("mix completed and non-completed", func(t *testing.T) {
		rev := &fakeRevision{phases: phases("a", "b", "c", "d", "e")}
		gated := map[string]struct{}{"a": {}}
		completed := map[string]bool{"b": true, "c": true}
		drift, readOnly := splitPhases(rev, gated, completed)
		assert.Equal(t, []string{"b", "c", "d"}, phaseNames(drift))
		assert.Equal(t, []string{"e"}, phaseNames(readOnly))
	})

	t.Run("single non-gated non-completed phase", func(t *testing.T) {
		rev := &fakeRevision{phases: phases("a", "b")}
		gated := map[string]struct{}{"a": {}}
		drift, readOnly := splitPhases(rev, gated, nil)
		assert.Empty(t, drift)
		assert.Equal(t, []string{"b"}, phaseNames(readOnly))
	})

	t.Run("gate+1 is the last phase", func(t *testing.T) {
		rev := &fakeRevision{phases: phases("a", "b", "c")}
		gated := map[string]struct{}{"a": {}}
		completed := map[string]bool{"b": true}
		drift, readOnly := splitPhases(rev, gated, completed)
		assert.Equal(t, []string{"b", "c"}, phaseNames(drift))
		assert.Empty(t, readOnly)
	})

	t.Run("no phases", func(t *testing.T) {
		rev := &fakeRevision{}
		drift, readOnly := splitPhases(rev, nil, nil)
		assert.Empty(t, drift)
		assert.Empty(t, readOnly)
	})
}

package revision

import (
	"context"

	"pkg.package-operator.run/boxcutter"
	"pkg.package-operator.run/boxcutter/machinery"
	"pkg.package-operator.run/boxcutter/machinery/types"
	"pkg.package-operator.run/boxcutter/validation"

	orbv1alpha1 "github.com/joelanford/orb-operator/api/v1alpha1"
)

type Engine struct {
	revision    *boxcutter.RevisionEngine
	phase       *machinery.PhaseEngine
	existingOPs []orbv1alpha1.ObservedPhase
}

func NewEngine(opts boxcutter.RevisionEngineOptions, existingOPs []orbv1alpha1.ObservedPhase) (*Engine, error) {
	re, err := boxcutter.NewRevisionEngine(opts)
	if err != nil {
		return nil, err
	}
	pe, err := boxcutter.NewPhaseEngine(opts)
	if err != nil {
		return nil, err
	}
	return &Engine{
		revision:    re,
		phase:       pe,
		existingOPs: existingOPs,
	}, nil
}

type Result struct {
	gated           machinery.RevisionResult
	driftResults    []machinery.PhaseResult
	readOnlyResults []machinery.PhaseResult
}

func (r *Result) GetValidationError() *validation.RevisionValidationError {
	return r.gated.GetValidationError()
}

func (r *Result) GetPhases() []machinery.PhaseResult {
	result := append(r.gated.GetPhases(), r.driftResults...)
	return append(result, r.readOnlyResults...)
}

func (r *Result) InTransition() bool {
	if r.gated.InTransition() {
		return true
	}
	for _, dr := range r.driftResults {
		if !dr.IsComplete() {
			return true
		}
	}
	return false
}

func (r *Result) IsComplete() bool {
	if !r.gated.IsComplete() {
		return false
	}
	for _, dr := range r.driftResults {
		if !dr.IsComplete() {
			return false
		}
	}
	return true
}

func (r *Result) HasProgressed() bool {
	return r.gated.HasProgressed()
}

func (r *Result) String() string {
	return r.gated.String()
}

func (e *Engine) Teardown(ctx context.Context, rev types.Revision, opts ...types.RevisionTeardownOption) (machinery.RevisionTeardownResult, error) {
	return e.revision.Teardown(ctx, rev, opts...)
}

func (e *Engine) Reconcile(ctx context.Context, rev types.Revision, opts ...types.RevisionReconcileOption) (machinery.RevisionResult, error) {
	gatedResult, err := e.revision.Reconcile(ctx, rev, opts...)
	if err != nil {
		return gatedResult, err
	}
	if gatedResult.GetValidationError() != nil || gatedResult.HasProgressed() {
		return gatedResult, nil
	}

	gatedPhaseNames := make(map[string]struct{}, len(gatedResult.GetPhases()))
	for _, pr := range gatedResult.GetPhases() {
		gatedPhaseNames[pr.GetName()] = struct{}{}
	}

	var revOpts types.RevisionReconcileOptions
	for _, o := range opts {
		o.ApplyToRevisionReconcileOptions(&revOpts)
	}

	driftPhases, readOnlyPhases := splitPhases(rev, gatedPhaseNames, e.completedPhaseNames())

	var driftResults []machinery.PhaseResult
	var driftErr error
	for _, phase := range driftPhases {
		phaseOpts := revOpts.ForPhase(phase.GetName())
		pr, pErr := e.phase.Reconcile(ctx, rev.GetRevisionNumber(), phase, phaseOpts...) //nolint:staticcheck
		if pr != nil {                                                                   //nolint:staticcheck // defensive: boxcutter may return nil in future versions
			driftResults = append(driftResults, pr)
		}
		if pErr != nil {
			driftErr = pErr
			break
		}
	}

	var readOnlyResults []machinery.PhaseResult
	if driftErr == nil {
		for _, phase := range readOnlyPhases {
			phaseOpts := append(revOpts.ForPhase(phase.GetName()), types.WithPaused{})
			pr, pErr := e.phase.Reconcile(ctx, rev.GetRevisionNumber(), phase, phaseOpts...) //nolint:staticcheck
			if pr != nil {                                                                   //nolint:staticcheck // defensive: boxcutter may return nil in future versions
				readOnlyResults = append(readOnlyResults, pr)
			}
			if pErr != nil {
				break
			}
		}
	}

	return &Result{
		gated:           gatedResult,
		driftResults:    driftResults,
		readOnlyResults: readOnlyResults,
	}, driftErr
}

func splitPhases(rev types.Revision, gatedPhaseNames map[string]struct{}, completedPhases map[string]bool) ([]types.Phase, []types.Phase) {
	var drift, readOnly []types.Phase
	sawCompleted := false
	for _, phase := range rev.GetPhases() {
		if _, inGated := gatedPhaseNames[phase.GetName()]; inGated {
			continue
		}
		isCompleted := completedPhases[phase.GetName()]
		if !isCompleted && !sawCompleted {
			readOnly = append(readOnly, phase)
			readOnly = append(readOnly, phasesAfter(rev, gatedPhaseNames, phase.GetName())...)
			return drift, readOnly
		}
		sawCompleted = true
		drift = append(drift, phase)
		if !isCompleted {
			readOnly = append(readOnly, phasesAfter(rev, gatedPhaseNames, phase.GetName())...)
			return drift, readOnly
		}
	}
	return drift, readOnly
}

// phasesAfter returns all non-gated phases that follow the phase named afterName in the revision's phase order.
func phasesAfter(rev types.Revision, gatedPhaseNames map[string]struct{}, afterName string) []types.Phase {
	var result []types.Phase
	found := false
	for _, phase := range rev.GetPhases() {
		if phase.GetName() == afterName {
			found = true
			continue
		}
		if !found {
			continue
		}
		if _, inGated := gatedPhaseNames[phase.GetName()]; inGated {
			continue
		}
		result = append(result, phase)
	}
	return result
}

func (e *Engine) completedPhaseNames() map[string]bool {
	m := make(map[string]bool, len(e.existingOPs))
	for _, op := range e.existingOPs {
		if op.CompletedAt != nil {
			m[op.Name] = true
		}
	}
	return m
}

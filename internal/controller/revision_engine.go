package controller

import (
	"context"

	"pkg.package-operator.run/boxcutter"
	"pkg.package-operator.run/boxcutter/machinery"
	"pkg.package-operator.run/boxcutter/machinery/types"
	"pkg.package-operator.run/boxcutter/validation"

	orbv1alpha1 "github.com/joelanford/orb-operator/api/v1alpha1"
)

type revisionEngine struct {
	revision    *boxcutter.RevisionEngine
	phase       *machinery.PhaseEngine
	existingOPs []orbv1alpha1.ObservedPhase
}

func newRevisionEngine(opts boxcutter.RevisionEngineOptions, existingOPs []orbv1alpha1.ObservedPhase) (*revisionEngine, error) {
	re, err := boxcutter.NewRevisionEngine(opts)
	if err != nil {
		return nil, err
	}
	pe, err := boxcutter.NewPhaseEngine(opts)
	if err != nil {
		return nil, err
	}
	return &revisionEngine{
		revision:    re,
		phase:       pe,
		existingOPs: existingOPs,
	}, nil
}

type revisionResult struct {
	gated        machinery.RevisionResult
	driftResults []machinery.PhaseResult
}

func (r *revisionResult) GetValidationError() *validation.RevisionValidationError {
	return r.gated.GetValidationError()
}

func (r *revisionResult) GetPhases() []machinery.PhaseResult {
	return append(r.gated.GetPhases(), r.driftResults...)
}

func (r *revisionResult) InTransition() bool {
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

func (r *revisionResult) IsComplete() bool {
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

func (r *revisionResult) HasProgressed() bool {
	return r.gated.HasProgressed()
}

func (r *revisionResult) String() string {
	return r.gated.String()
}

func (re *revisionEngine) Teardown(ctx context.Context, rev types.Revision, opts ...types.RevisionTeardownOption) (machinery.RevisionTeardownResult, error) {
	return re.revision.Teardown(ctx, rev, opts...)
}

func (re *revisionEngine) Reconcile(ctx context.Context, rev types.Revision, opts ...types.RevisionReconcileOption) (machinery.RevisionResult, error) {
	gatedResult, err := re.revision.Reconcile(ctx, rev, opts...)
	if err != nil {
		return gatedResult, err
	}
	if gatedResult.GetValidationError() != nil || gatedResult.HasProgressed() {
		return gatedResult, nil
	}

	completedPhases := re.completedPhaseNames()
	gatedPhaseNames := make(map[string]struct{}, len(gatedResult.GetPhases()))
	for _, pr := range gatedResult.GetPhases() {
		gatedPhaseNames[pr.GetName()] = struct{}{}
	}

	var revOpts types.RevisionReconcileOptions
	for _, o := range opts {
		o.ApplyToRevisionReconcileOptions(&revOpts)
	}

	var driftResults []machinery.PhaseResult
	var driftErr error
	sawCompleted := false
	for _, phase := range rev.GetPhases() {
		if _, inGated := gatedPhaseNames[phase.GetName()]; inGated {
			continue
		}
		isCompleted := completedPhases[phase.GetName()]
		if !isCompleted && !sawCompleted {
			break
		}
		if isCompleted {
			sawCompleted = true
		}
		phaseOpts := revOpts.ForPhase(phase.GetName())
		pr, pErr := re.phase.Reconcile(ctx, rev.GetRevisionNumber(), phase, phaseOpts...)
		if pr != nil {
			driftResults = append(driftResults, pr)
		}
		if pErr != nil {
			driftErr = pErr
			break
		}
		if !isCompleted {
			break
		}
	}

	return &revisionResult{
		gated:        gatedResult,
		driftResults: driftResults,
	}, driftErr
}

func (re *revisionEngine) completedPhaseNames() map[string]bool {
	m := make(map[string]bool, len(re.existingOPs))
	for _, op := range re.existingOPs {
		if op.CompletedAt != nil {
			m[op.Name] = true
		}
	}
	return m
}

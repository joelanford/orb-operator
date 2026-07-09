package controller

import (
	"fmt"
	"maps"
	"slices"

	"pkg.package-operator.run/boxcutter/machinery"
	"pkg.package-operator.run/boxcutter/machinery/types"
	"pkg.package-operator.run/boxcutter/validation"

	orbv1alpha1 "github.com/joelanford/orb-operator/api/v1alpha1"
)

func observedPhasesFromReconcileResult(specPhases []orbv1alpha1.Phase, result machinery.RevisionResult) []orbv1alpha1.ObservedPhase {
	if result == nil {
		return nil
	}
	if verr := result.GetValidationError(); verr != nil {
		return buildValidationErrorObservedPhases(specPhases, verr)
	}
	if result.HasProgressed() {
		return allPhasesWithStatus(specPhases, orbv1alpha1.PhaseStatusSuperseded)
	}
	return buildObservedPhases(specPhases, result.GetPhases())
}

func allPhasesWithStatus(specPhases []orbv1alpha1.Phase, status orbv1alpha1.PhaseStatus) []orbv1alpha1.ObservedPhase {
	observed := make([]orbv1alpha1.ObservedPhase, len(specPhases))
	for i, sp := range specPhases {
		observed[i] = orbv1alpha1.ObservedPhase{
			Name:   sp.Name,
			Status: status,
		}
	}
	return observed
}

func buildObservedPhases(specPhases []orbv1alpha1.Phase, phaseResults []machinery.PhaseResult) []orbv1alpha1.ObservedPhase {
	return mapSpecPhases(specPhases, phaseResults, orbv1alpha1.PhaseStatusAvailable, reconcilingPhase)
}

func reconcilingPhase(_ orbv1alpha1.Phase, pr machinery.PhaseResult) orbv1alpha1.ObservedPhase {
	op := orbv1alpha1.ObservedPhase{
		Name:   pr.GetName(),
		Status: orbv1alpha1.PhaseStatusReconciling,
	}

	if verr := pr.GetValidationError(); verr != nil {
		applyValidationError(&op, verr)
	}

	for _, obj := range pr.GetObjects() {
		if obj.IsComplete() {
			continue
		}
		os := objectStatusFromRef(types.ToObjectRef(obj.Object()))
		os.Messages = messagesForObject(obj)
		op.IncompleteObjects = append(op.IncompleteObjects, os)
	}
	return op
}

func applyValidationError(op *orbv1alpha1.ObservedPhase, verr *validation.PhaseValidationError) {
	if verr.PhaseError != nil {
		op.Error = truncateMessage(fmt.Sprintf("validation error: %v", verr.PhaseError))
	}
	for _, objErr := range verr.Objects {
		msgs := make([]string, 0, len(objErr.Errors))
		for _, e := range objErr.Errors {
			msgs = append(msgs, truncateMessage(fmt.Sprintf("validation error: %v", e)))
		}
		os := objectStatusFromRef(objErr.ObjectRef)
		os.Messages = msgs
		op.IncompleteObjects = append(op.IncompleteObjects, os)
	}
}

func buildValidationErrorObservedPhases(specPhases []orbv1alpha1.Phase, verr *validation.RevisionValidationError) []orbv1alpha1.ObservedPhase {
	errsByPhase := make(map[string]*validation.PhaseValidationError, len(verr.Phases))
	for i := range verr.Phases {
		errsByPhase[verr.Phases[i].PhaseName] = &verr.Phases[i]
	}

	observed := make([]orbv1alpha1.ObservedPhase, 0, len(specPhases))
	for _, sp := range specPhases {
		perr, hasError := errsByPhase[sp.Name]
		if !hasError {
			observed = append(observed, orbv1alpha1.ObservedPhase{
				Name:   sp.Name,
				Status: orbv1alpha1.PhaseStatusUnknown,
			})
			continue
		}

		op := orbv1alpha1.ObservedPhase{
			Name:   sp.Name,
			Status: orbv1alpha1.PhaseStatusReconciling,
		}
		applyValidationError(&op, perr)
		observed = append(observed, op)
	}

	return observed
}

func observedPhasesFromTeardownResult(specPhases []orbv1alpha1.Phase, result machinery.RevisionTeardownResult) []orbv1alpha1.ObservedPhase {
	if result == nil {
		return nil
	}
	return buildTeardownObservedPhases(specPhases, result.GetPhases())
}

func buildTeardownObservedPhases(specPhases []orbv1alpha1.Phase, phaseResults []machinery.PhaseTeardownResult) []orbv1alpha1.ObservedPhase {
	return mapSpecPhases(specPhases, phaseResults, orbv1alpha1.PhaseStatusTeardownComplete, tearingDownPhase)
}

func tearingDownPhase(_ orbv1alpha1.Phase, pr machinery.PhaseTeardownResult) orbv1alpha1.ObservedPhase {
	op := orbv1alpha1.ObservedPhase{
		Name:   pr.GetName(),
		Status: orbv1alpha1.PhaseStatusTearingDown,
	}
	for _, ref := range pr.Waiting() {
		os := objectStatusFromRef(ref)
		os.Messages = []string{"awaiting deletion"}
		op.IncompleteObjects = append(op.IncompleteObjects, os)
	}
	return op
}

func mapSpecPhases[T interface {
	GetName() string
	IsComplete() bool
}](
	specPhases []orbv1alpha1.Phase,
	results []T,
	completeStatus orbv1alpha1.PhaseStatus,
	buildIncomplete func(orbv1alpha1.Phase, T) orbv1alpha1.ObservedPhase,
) []orbv1alpha1.ObservedPhase {
	resultsByName := make(map[string]T, len(results))
	for _, r := range results {
		resultsByName[r.GetName()] = r
	}

	observed := make([]orbv1alpha1.ObservedPhase, 0, len(specPhases))
	for _, sp := range specPhases {
		r, evaluated := resultsByName[sp.Name]
		if !evaluated {
			observed = append(observed, orbv1alpha1.ObservedPhase{
				Name:   sp.Name,
				Status: orbv1alpha1.PhaseStatusUnknown,
			})
			continue
		}
		if r.IsComplete() {
			observed = append(observed, orbv1alpha1.ObservedPhase{
				Name:   sp.Name,
				Status: completeStatus,
			})
			continue
		}
		observed = append(observed, buildIncomplete(sp, r))
	}
	return observed
}

func objectStatusFromRef(ref types.ObjectRef) orbv1alpha1.ObjectStatus {
	return orbv1alpha1.ObjectStatus{
		Group:     ref.Group,
		Version:   ref.Version,
		Kind:      ref.Kind,
		Namespace: ref.Namespace,
		Name:      ref.Name,
	}
}

func messagesForObject(obj machinery.ObjectResult) []string {
	var msgs []string

	if obj.Action() == machinery.ActionCollision {
		msgs = append(msgs, "object ownership collision")
	}

	probes := obj.ProbeResults()
	for _, probeType := range slices.Sorted(maps.Keys(probes)) {
		result := probes[probeType]
		if result.Status == types.ProbeStatusTrue {
			continue
		}
		for _, m := range result.Messages {
			msgs = append(msgs, truncateMessage(fmt.Sprintf("%s: %s", probeType, m)))
		}
		if len(result.Messages) == 0 {
			msgs = append(msgs, fmt.Sprintf("%s: %s", probeType, result.Status))
		}
	}

	if len(msgs) == 0 {
		msgs = append(msgs, "not yet complete")
	}

	return msgs
}

const maxMessageLength = 1024

func truncateMessage(s string) string {
	r := []rune(s)
	if len(r) <= maxMessageLength {
		return s
	}
	return string(r[:maxMessageLength-3]) + "..."
}

package cos

import (
	"errors"
	"fmt"
	"maps"
	"slices"
	"strings"
	"time"

	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"pkg.package-operator.run/boxcutter/machinery"
	"pkg.package-operator.run/boxcutter/machinery/types"
	"pkg.package-operator.run/boxcutter/validation"
	"sigs.k8s.io/structured-merge-diff/v6/fieldpath"

	orbv1alpha1 "github.com/joelanford/orb-operator/api/v1alpha1"
	orberrors "github.com/joelanford/orb-operator/internal/errors"
)

type Update struct {
	Condition      metav1.Condition
	ObservedPhases *[]orbv1alpha1.ObservedPhase
	CompletedAt    *metav1.Time
}

func Apply(cos *orbv1alpha1.ClusterObjectSet, u Update) {
	u.Condition.ObservedGeneration = cos.Generation
	meta.SetStatusCondition(&cos.Status.Conditions, u.Condition)
	if u.ObservedPhases != nil {
		cos.Status.ObservedPhases = *u.ObservedPhases
	}
	cos.Status.ObjectCounts = sumObjectCounts(cos.Status.ObservedPhases)
	if u.CompletedAt != nil && cos.Status.CompletedAt == nil {
		cos.Status.CompletedAt = u.CompletedAt
	}
}

func sumObjectCounts(phases []orbv1alpha1.ObservedPhase) *orbv1alpha1.ObjectCounts {
	if len(phases) == 0 {
		return nil
	}
	var counts orbv1alpha1.ObjectCounts
	for i := range phases {
		counts.Total += phases[i].ObjectCounts.Total
		counts.Present += phases[i].ObjectCounts.Present
		counts.Synced += phases[i].ObjectCounts.Synced
		counts.Available += phases[i].ObjectCounts.Available
	}
	return &counts
}

func FromReconcile(cos *orbv1alpha1.ClusterObjectSet, result machinery.RevisionResult, err error, now time.Time) Update {
	var resErr *orberrors.ObjectResolutionError
	var intErr *orberrors.InternalError

	switch {
	case errors.As(err, &resErr):
		return resolutionErrorUpdate(cos.Status.ResolvedContentHash, err)
	case errors.As(err, &intErr):
		return internalErrorUpdate(err)
	}

	phases := ObservedPhasesFromReconcileResult(cos.Spec.Phases, result)
	PreserveCompletionTimes(cos.Status.ObservedPhases, phases, now)

	u := Update{ObservedPhases: &phases}

	if err != nil {
		u.Condition = newCondition(metav1.ConditionUnknown, orbv1alpha1.ReasonReconcileError,
			fmt.Sprintf("reconcile failed: %v", err))
		return u
	}

	if verr := result.GetValidationError(); verr != nil {
		u.Condition = newCondition(metav1.ConditionFalse, orbv1alpha1.ReasonInvalidRevision, verr.Error())
		return u
	}

	switch {
	case result.HasProgressed():
		u.Condition = newCondition(metav1.ConditionFalse, orbv1alpha1.ReasonSuperseded,
			"all objects adopted by a newer revision")
	case result.IsComplete():
		mt := metav1.NewTime(now)
		u.CompletedAt = &mt
		u.Condition = newCondition(metav1.ConditionTrue, orbv1alpha1.ReasonAvailable, "all phases complete")
	default:
		u.Condition = newCondition(metav1.ConditionFalse, orbv1alpha1.ReasonUnavailable, "phases not yet complete")
	}
	return u
}

func FromTeardown(cos *orbv1alpha1.ClusterObjectSet, result machinery.RevisionTeardownResult, err error, now time.Time) Update {
	var resErr *orberrors.ObjectResolutionError
	var intErr *orberrors.InternalError

	switch {
	case errors.As(err, &resErr):
		return resolutionErrorUpdate(cos.Status.ResolvedContentHash, err)
	case errors.As(err, &intErr):
		return internalErrorUpdate(err)
	}

	phases := ObservedPhasesFromTeardownResult(cos.Spec.Phases, result)
	PreserveCompletionTimes(cos.Status.ObservedPhases, phases, now)

	u := Update{ObservedPhases: &phases}

	switch {
	case err != nil:
		u.Condition = newCondition(metav1.ConditionUnknown, orbv1alpha1.ReasonTeardownError,
			fmt.Sprintf("teardown failed: %v", err))
	case result != nil && !result.IsComplete():
		u.Condition = newCondition(metav1.ConditionFalse, orbv1alpha1.ReasonArchived, "teardown in progress")
	default:
		u.Condition = newCondition(metav1.ConditionFalse, orbv1alpha1.ReasonArchived, "teardown complete")
	}
	return u
}

func resolutionErrorUpdate(existingHash string, err error) Update {
	status := metav1.ConditionFalse
	if existingHash != "" {
		status = metav1.ConditionUnknown
	}
	return Update{
		Condition: newCondition(status, orbv1alpha1.ReasonInvalidRevision, err.Error()),
	}
}

func internalErrorUpdate(err error) Update {
	empty := []orbv1alpha1.ObservedPhase(nil)
	return Update{
		Condition:      newCondition(metav1.ConditionUnknown, orbv1alpha1.ReasonInternalError, err.Error()),
		ObservedPhases: &empty,
	}
}

func newCondition(status metav1.ConditionStatus, reason, message string) metav1.Condition {
	return metav1.Condition{
		Type:    orbv1alpha1.ConditionTypeAvailable,
		Status:  status,
		Reason:  reason,
		Message: message,
	}
}

func ObservedPhasesFromReconcileResult(specPhases []orbv1alpha1.Phase, result machinery.RevisionResult) []orbv1alpha1.ObservedPhase {
	if result == nil {
		return nil
	}
	if verr := result.GetValidationError(); verr != nil {
		return invalidPhasesFromValidationError(specPhases, verr)
	}
	if result.HasProgressed() {
		return allPhasesWithStatus(specPhases, orbv1alpha1.PhaseStatusSuperseded)
	}
	return buildObservedPhases(specPhases, result.GetPhases())
}

func invalidPhasesFromValidationError(specPhases []orbv1alpha1.Phase, verr *validation.RevisionValidationError) []orbv1alpha1.ObservedPhase {
	phaseErrors := make(map[string]*validation.PhaseValidationError, len(verr.Phases))
	for i := range verr.Phases {
		phaseErrors[verr.Phases[i].PhaseName] = &verr.Phases[i]
	}

	observed := make([]orbv1alpha1.ObservedPhase, 0, len(specPhases))
	for _, sp := range specPhases {
		if pve, ok := phaseErrors[sp.Name]; ok {
			op := orbv1alpha1.ObservedPhase{
				Name: sp.Name,
			}
			applyValidationError(&op, pve)
			observed = append(observed, op)
		} else {
			observed = append(observed, orbv1alpha1.ObservedPhase{
				Name:    sp.Name,
				Status:  orbv1alpha1.PhaseStatusUnknown,
				Message: "Blocked by preflight errors in other phases",
			})
		}
	}
	return observed
}

func allPhasesWithStatus(specPhases []orbv1alpha1.Phase, status orbv1alpha1.PhaseStatus) []orbv1alpha1.ObservedPhase {
	observed := make([]orbv1alpha1.ObservedPhase, len(specPhases))
	for i, sp := range specPhases {
		observed[i] = orbv1alpha1.ObservedPhase{
			Name:         sp.Name,
			Status:       status,
			ObjectCounts: orbv1alpha1.ObjectCounts{Total: int64(len(sp.Objects))},
		}
	}
	return observed
}

func buildObservedPhases(specPhases []orbv1alpha1.Phase, phaseResults []machinery.PhaseResult) []orbv1alpha1.ObservedPhase {
	return mapSpecPhases(specPhases, phaseResults, orbv1alpha1.PhaseStatusAvailable, "Phase was not evaluated",
		func(total int64) orbv1alpha1.ObjectCounts {
			return orbv1alpha1.ObjectCounts{Total: total, Present: total, Synced: total, Available: total}
		},
		incompletePhase,
	)
}

func incompletePhase(sp orbv1alpha1.Phase, pr machinery.PhaseResult) orbv1alpha1.ObservedPhase {
	op := orbv1alpha1.ObservedPhase{
		Name:         pr.GetName(),
		ObjectCounts: orbv1alpha1.ObjectCounts{Total: int64(len(sp.Objects))},
	}

	if verr := pr.GetValidationError(); verr != nil {
		applyValidationError(&op, verr)
		return op
	}

	paused := false
	for _, obj := range pr.GetObjects() {
		if !obj.IsPaused() || obj.Action() != machinery.ActionCreated {
			op.ObjectCounts.Present++
		}
		if obj.IsPaused() {
			paused = true
			if obj.Action() == machinery.ActionIdle {
				op.ObjectCounts.Synced++
			}
		} else {
			switch obj.Action() {
			case machinery.ActionIdle, machinery.ActionUpdated, machinery.ActionCreated, machinery.ActionRecovered:
				op.ObjectCounts.Synced++
			}
		}
		if obj.IsComplete() {
			op.ObjectCounts.Available++
			continue
		}
		os := objectStatusFromRef(types.ToObjectRef(obj.Object()))
		os.Messages = objectMessages(obj)
		op.ObjectDetails = append(op.ObjectDetails, os)
	}

	switch {
	case paused:
		op.Status = orbv1alpha1.PhaseStatusPending
		op.Message = "Waiting for earlier phases to complete"
	case op.ObjectCounts.Synced == op.ObjectCounts.Total:
		op.Status = orbv1alpha1.PhaseStatusWaitingForAssertions
	default:
		op.Status = orbv1alpha1.PhaseStatusReconciling
	}
	return op
}

func applyValidationError(op *orbv1alpha1.ObservedPhase, verr *validation.PhaseValidationError) {
	op.Status = orbv1alpha1.PhaseStatusInvalid
	if verr.PhaseError != nil {
		op.Message = truncateMessage(fmt.Sprintf("validation error: %v", verr.PhaseError))
	}
	for _, objErr := range verr.Objects {
		msgs := make([]string, 0, len(objErr.Errors))
		for _, e := range objErr.Errors {
			msgs = append(msgs, truncateMessage(fmt.Sprintf("validation error: %v", e)))
		}
		os := objectStatusFromRef(objErr.ObjectRef)
		os.Messages = msgs
		op.ObjectDetails = append(op.ObjectDetails, os)
	}
}

func ObservedPhasesFromTeardownResult(specPhases []orbv1alpha1.Phase, result machinery.RevisionTeardownResult) []orbv1alpha1.ObservedPhase {
	if result == nil {
		return nil
	}
	activeName, _ := result.GetActivePhaseName()
	return buildTeardownObservedPhases(specPhases, result.GetPhases(), activeName)
}

func buildTeardownObservedPhases(specPhases []orbv1alpha1.Phase, phaseResults []machinery.PhaseTeardownResult, activePhaseName string) []orbv1alpha1.ObservedPhase {
	return mapSpecPhases(specPhases, phaseResults, orbv1alpha1.PhaseStatusTeardownComplete, "",
		func(total int64) orbv1alpha1.ObjectCounts {
			return orbv1alpha1.ObjectCounts{Total: total}
		},
		func(sp orbv1alpha1.Phase, pr machinery.PhaseTeardownResult) orbv1alpha1.ObservedPhase {
			return tearingDownPhase(sp, pr, pr.GetName() == activePhaseName)
		},
	)
}

func tearingDownPhase(sp orbv1alpha1.Phase, pr machinery.PhaseTeardownResult, active bool) orbv1alpha1.ObservedPhase {
	op := orbv1alpha1.ObservedPhase{
		Name: pr.GetName(),
		ObjectCounts: orbv1alpha1.ObjectCounts{
			Total:   int64(len(sp.Objects)),
			Present: int64(len(pr.Waiting())),
		},
	}
	if active {
		op.Status = orbv1alpha1.PhaseStatusTearingDown
	} else {
		op.Status = orbv1alpha1.PhaseStatusPending
		op.Message = "Waiting for later phases to complete teardown"
	}
	for _, ref := range pr.Waiting() {
		os := objectStatusFromRef(ref)
		os.Messages = []string{"awaiting deletion"}
		op.ObjectDetails = append(op.ObjectDetails, os)
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
	unevaluatedMessage string,
	completeObjectCounts func(total int64) orbv1alpha1.ObjectCounts,
	buildIncomplete func(orbv1alpha1.Phase, T) orbv1alpha1.ObservedPhase,
) []orbv1alpha1.ObservedPhase {
	resultsByName := make(map[string]T, len(results))
	for _, r := range results {
		resultsByName[r.GetName()] = r
	}

	observed := make([]orbv1alpha1.ObservedPhase, 0, len(specPhases))
	for _, sp := range specPhases {
		total := int64(len(sp.Objects))
		r, evaluated := resultsByName[sp.Name]
		if !evaluated {
			observed = append(observed, orbv1alpha1.ObservedPhase{
				Name:         sp.Name,
				Status:       orbv1alpha1.PhaseStatusUnknown,
				Message:      unevaluatedMessage,
				ObjectCounts: orbv1alpha1.ObjectCounts{Total: total},
			})
			continue
		}
		if r.IsComplete() {
			observed = append(observed, orbv1alpha1.ObservedPhase{
				Name:         sp.Name,
				Status:       completeStatus,
				ObjectCounts: completeObjectCounts(total),
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

func objectMessages(obj machinery.ObjectResult) []string {
	if obj.IsPaused() && obj.Action() != machinery.ActionIdle {
		return pausedObjectMessages(obj)
	}

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

func PreserveCompletionTimes(existing, current []orbv1alpha1.ObservedPhase, now time.Time) {
	completedAt := make(map[string]*metav1.Time, len(existing))
	for i := range existing {
		if existing[i].CompletedAt != nil {
			completedAt[existing[i].Name] = existing[i].CompletedAt
		}
	}
	for i := range current {
		if t, ok := completedAt[current[i].Name]; ok {
			current[i].CompletedAt = t
		} else if current[i].Status == orbv1alpha1.PhaseStatusAvailable {
			mt := metav1.NewTime(now)
			current[i].CompletedAt = &mt
		}
	}
}

type compareResulter interface {
	CompareResult() machinery.CompareResult
}

func pausedObjectMessages(obj machinery.ObjectResult) []string {
	var summary string
	switch obj.Action() {
	case machinery.ActionCreated:
		return []string{"object does not exist"}
	case machinery.ActionRecovered:
		summary = "object was modified by another actor"
	default:
		summary = "object content has changed"
	}

	if details := compareDetails(obj); len(details) > 0 {
		summary += "\n" + strings.Join(details, "\n")
	}
	return []string{summary}
}

func compareDetails(obj machinery.ObjectResult) []string {
	cr, ok := obj.(compareResulter)
	if !ok {
		return nil
	}
	comp := cr.CompareResult().Comparison
	if comp == nil {
		return nil
	}
	var details []string
	if comp.Added != nil && !comp.Added.Empty() {
		comp.Added.Leaves().Iterate(func(p fieldpath.Path) {
			details = append(details, fmt.Sprintf(" - added: %s", p))
		})
	}
	if comp.Modified != nil && !comp.Modified.Empty() {
		comp.Modified.Leaves().Iterate(func(p fieldpath.Path) {
			details = append(details, fmt.Sprintf(" - modified: %s", p))
		})
	}
	if comp.Removed != nil && !comp.Removed.Empty() {
		comp.Removed.Leaves().Iterate(func(p fieldpath.Path) {
			details = append(details, fmt.Sprintf(" - removed: %s", p))
		})
	}
	return details
}

const maxMessageLength = 1024

func truncateMessage(s string) string {
	r := []rune(s)
	if len(r) <= maxMessageLength {
		return s
	}
	return string(r[:maxMessageLength-3]) + "..."
}

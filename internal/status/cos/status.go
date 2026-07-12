package cos

import (
	"errors"
	"fmt"
	"maps"
	"slices"
	"time"

	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"pkg.package-operator.run/boxcutter/machinery"
	"pkg.package-operator.run/boxcutter/machinery/types"
	"pkg.package-operator.run/boxcutter/validation"

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
	if u.CompletedAt != nil && cos.Status.CompletedAt == nil {
		cos.Status.CompletedAt = u.CompletedAt
	}
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
				Name:   sp.Name,
				Status: orbv1alpha1.PhaseStatusInvalid,
			}
			applyValidationError(&op, pve)
			observed = append(observed, op)
		} else {
			observed = append(observed, orbv1alpha1.ObservedPhase{
				Name:   sp.Name,
				Status: orbv1alpha1.PhaseStatusUnknown,
				Error:  "Blocked by preflight errors in other phases",
			})
		}
	}
	return observed
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
	return mapSpecPhases(specPhases, phaseResults, orbv1alpha1.PhaseStatusAvailable, "Waiting for earlier phases to complete", reconcilingPhase)
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

func ObservedPhasesFromTeardownResult(specPhases []orbv1alpha1.Phase, result machinery.RevisionTeardownResult) []orbv1alpha1.ObservedPhase {
	if result == nil {
		return nil
	}
	return buildTeardownObservedPhases(specPhases, result.GetPhases())
}

func buildTeardownObservedPhases(specPhases []orbv1alpha1.Phase, phaseResults []machinery.PhaseTeardownResult) []orbv1alpha1.ObservedPhase {
	return mapSpecPhases(specPhases, phaseResults, orbv1alpha1.PhaseStatusTeardownComplete, "", tearingDownPhase)
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
	unknownError string,
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
				Error:  unknownError,
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

const maxMessageLength = 1024

func truncateMessage(s string) string {
	r := []rune(s)
	if len(r) <= maxMessageLength {
		return s
	}
	return string(r[:maxMessageLength-3]) + "..."
}

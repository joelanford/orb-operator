---
status: idea
---
# Boxcutter upstream fixes needed for orb-operator expected functionality

## Summary

Two issues in boxcutter's validation pipeline prevent orb-operator from surfacing structured phase/object-level validation errors in COSR status. Both cause validation errors to escape as unstructured Go errors instead of being captured in the `PhaseValidationError` / `RevisionValidationError` hierarchy.

## Issue 1: `ObjectValidationError` pointer/value type mismatch in `PhaseValidator`

### Problem

In `validation/phase.go`, `PhaseValidator.Validate` uses a value-typed target for `errors.As`:

```go
var oerr ObjectValidationError          // value type
if errors.As(err, &oerr) {             // target is *ObjectValidationError
    objectErrors = append(objectErrors, oerr)
} else {
    errs = append(errs, err)            // ← falls through here
}
```

`NewObjectValidationError` returns `*ObjectValidationError` (pointer). `errors.As` cannot match a `*ObjectValidationError` to a `*ObjectValidationError` target produced by `&oerr` when `oerr` is a value type.

### Impact

Object-level validation errors (e.g., invalid resource names caught by dry-run) escape as unstructured Go errors. `PhaseValidator.Validate` returns `errors.Join(errs...)` instead of a structured `PhaseValidationError`. Downstream, `PhaseEngine.Reconcile` returns these as a hard error rather than populating `PhaseResult.validationError`.

### Fix

Change the `errors.As` target from a value type to a pointer type:

```go
var oerr *ObjectValidationError          // pointer type
if errors.As(err, &oerr) {
    objectErrors = append(objectErrors, *oerr)
}
```

### Deferred e2e scenario

Once fixed and bumped, add to `test/e2e/features/cosr_phase_status.feature`:

```gherkin
Scenario: Object validation error surfaces in incomplete objects
  Given a COSR with group "ps-valobj" and revision 1
  And a phase "install" with a ConfigMap "INVALID-CM"
  When the COSR is created
  Then the COSR should have observed phase "install" with status "Reconciling"
  And observed phase "install" should have an incomplete object "INVALID-CM"
  And incomplete object "INVALID-CM" in phase "install" should have a message containing "validation error"
```

## Issue 2: `NoMatchError` not recognized as a `DryRunValidationError`

### Problem

In `validation/object.go`, `validateDryRun` does a server-side dry-run apply (line 184). For a non-existent GVK, the REST mapper returns a `meta.NoMatchError` before the request reaches the API server. This error is not an `*apimachineryerrors.StatusError`, so it falls through `validateDryRun`'s type switch and is returned as a plain error (line 222).

```go
// validateDryRun (simplified)
err := w.Patch(ctx, dst, patch, ...)  // NoMatchError from REST mapper
var apiErr *apimachineryerrors.StatusError
switch {
case err == nil:
    return nil
case errors.As(err, &apiErr):
    // NoMatchError is NOT a StatusError → not matched
    ...
}
return err  // ← plain error escapes
```

`ObjectValidator.Validate` receives this plain error (not a `DryRunValidationError`) and returns it directly (line 98), bypassing `ObjectValidationError` construction. Combined with Issue 1, this cascades: `PhaseValidator` can't match it, returns `errors.Join(errs...)`, and the phase result is dropped.

### Impact

A COSR containing an object with a non-existent GVK gets `Available=Unknown` with `Reason=ReconcileError` and no `ObservedPhases`. The error message in the condition is informative, but no structured phase/object status is available.

### Fix

In `validateDryRun`, recognize `meta.NoMatchError` as a validation error:

```go
case meta.IsNoMatchError(err):
    return DryRunValidationError{err: err}
```

### E2e scenarios

The existing reconcile-error e2e scenario currently uses an invalid existing GVK (e.g., a ConfigMap with an invalid name) to test error reporting through the structured `DryRunValidationError` path.

Once this fix lands and the dependency is bumped, add a new scenario for the non-existent GVK case:

```gherkin
Scenario: COSR reports validation error for unregistered resource type
  Given a COSR with group "test" and revision 1
  And a phase "install" with an unregistered resource type
  When the COSR is created
  Then the COSR should have observed phase "install" with status "Reconciling"
  And observed phase "install" should have an incomplete object "fake-resource"
```

Both scenarios (invalid existing GVK and non-existent GVK) should coexist to cover both validation paths.

## Issue 3: `RevisionEngine.Teardown` returns nil result on phase teardown error

### Problem

In `machinery/revision.go`, `RevisionEngine.Teardown` discards accumulated phase results when any phase teardown fails:

```go
for _, p := range reversedPhases {
    // ...
    pres, err := re.phaseEngine.Teardown(ctx, rev.GetRevisionNumber(), p, ...)
    if err != nil {
        return nil, fmt.Errorf("teardown phase: %w", err)  // ← discards res
    }
    res.phases = append(res.phases, pres)
    // ...
}
```

If phases are torn down in reverse order (e.g., phase-3, phase-2, phase-1) and phase-2 fails, the successful teardown of phase-3 is lost because `res` is discarded and `nil` is returned.

### Impact

Consumers cannot report which phases tore down successfully vs. which failed. The entire teardown result is `nil`, so `updateTeardownStatus` falls into the `Unknown`/`TeardownError` branch even though partial progress was made.

### Fix

Append the phase result (if non-nil) before returning the error, and return `res` instead of `nil`:

```go
if err != nil {
    if pres != nil {
        res.phases = append(res.phases, pres)
    }
    return res, fmt.Errorf("teardown phase: %w", err)
}
```

### E2e scenarios

Once fixed, teardown error scenarios can assert partial phase status (e.g., phase-3 shows `TeardownComplete` while phase-2 shows `TearingDown` with an error).

## Future direction: `observedRevision` status field

As these upstream fixes land and boxcutter returns richer results in error paths, the orb-operator controller should move toward an `observedRevision` status field that is always populated from whatever the engine returns — regardless of success or error. This separates the verdict (the `Available` condition) from the evidence (the observed state).

```yaml
observedRevision:
  message: ""
  phases:
  - name: phase-1
    status: ""
    message: ""
    objects:
    - group: ""
      version: ""
      kind: ""
      name: ""
      namespace: ""
      messages:
      - ""
```

Currently, the controller decides whether to populate `ObservedPhases` based on whether the engine returned "enough" data. This encodes assumptions about what the engine returns in each error path, and requires controller-side changes every time boxcutter improves its result reporting. An always-populated `observedRevision` field would automatically surface richer data as boxcutter improves, without controller changes.

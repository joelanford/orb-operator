# Verification

## Implementation Correctness

- [ ] `api/v1alpha1/types_clusterobjectset.go` defines `ClusterObjectSetSpec` with `RevisionHistoryLimit *int32` and `Template ClusterObjectSetTemplate`
- [ ] `ClusterObjectSetTemplate` has `Metadata` (labels, annotations) and `Spec` (`ClusterObjectSetTemplateSpec`)
- [ ] `ClusterObjectSetTemplateSpec` contains only `CollisionProtection` and `Phases` (NOT group, revision, or lifecycleState)
- [ ] `ClusterObjectSetRevisionSpec` embeds `ClusterObjectSetTemplateSpec` with `json:",inline"`
- [ ] CEL immutability rules for `phases` and `collisionProtection` remain on `ClusterObjectSetRevisionSpec`, not on `ClusterObjectSetTemplateSpec`
- [ ] COSR CRD shape and validation rules are unchanged after the refactor (diff the before/after CRD YAML)
- [ ] COS CRD has no immutability validation rules on `template.spec` fields
- [ ] `ClusterObjectSetStatus` has `Conditions []metav1.Condition`
- [ ] `zz_generated.deepcopy.go` is updated with deepcopy methods for new types
- [ ] COS CRD manifest in `deploy/crds/` has group `orb.operatorframework.io`, version `v1alpha1`, kind `ClusterObjectSet`, scope `Cluster`
- [ ] COS CRD has subresource status enabled
- [ ] COS CRD has print columns for Availability (reason from Available condition) and Age
- [ ] Feature files cover: COSR stamping, template metadata propagation, revision management (numbering + idempotency), status derivation, ownership (owner refs + cascade), revision history limit
- [ ] All COS step definitions are implemented — no undefined steps, no `godog.ErrPending`
- [ ] All COS tests fail with timeout or assertion errors, not compilation or infrastructure errors
- [ ] Existing COSR tests are not broken by the changes

## Project Conventions

- [ ] Types use standard controller-runtime patterns (TypeMeta, ObjectMeta, Spec/Status split)
- [ ] CRD markers follow controller-gen conventions
- [ ] Test structure follows `specs/tech-stack.md`: godog for e2e, kind cluster with full operator deployment for test backend
- [ ] `make verify` passes (lint, generate diff check, build)
- [ ] No `//nolint` comments added
- [ ] Follows ADR-0001 field names and semantics
- [ ] Template metadata design follows the ADR principle: "template metadata is the extension point"

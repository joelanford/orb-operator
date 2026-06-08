---
status: idea
---
# Bump boxcutter to 0.14.0

Upgrade `pkg.package-operator.run/boxcutter` from 0.13.1 to 0.14.0. The breaking change is the removal of `WithPreviousOwners` in favor of `WithSiblingOwners(siblingObjs)`. The single usage site is in `buildRevisionWithPreviousOwners` in `internal/controller/cosr_controller.go`, which constructs a `WithPreviousOwners` slice from predecessor COSRs and passes it as a `RevisionReconcileOption`. Migrate this to `WithSiblingOwners`, rename the helper to match, and verify all e2e revision transition scenarios still pass.

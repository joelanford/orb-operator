---
status: idea
---
# Validation, GoDoc, and API Best Practices

Audit and harden all three resource types (COS, COSR, ClusterObjectSlice) for validation completeness, API documentation, and Kubernetes API conventions.

## Validation

- Audit all spec fields for missing validation (MinLength, Minimum, Enum, MaxLength, etc.)
- Ensure all immutability constraints use XValidation CEL rules (group, revision, phases, collisionProtection already done)
- Ensure one-way transitions use XValidation (lifecycleState Active → Archived already done)
- Validate structural constraints on phases, assertions, and collision protection
- Consider whether any constraints currently enforced by VAPs should move to CRD-level validation
- Review whether validating webhooks are needed for constraints that CEL cannot express

## GoDoc

- Add godoc comments to all exported types, fields, and constants
- Use the JSON field name (not the Go struct field name) when referring to fields in comments
- Ensure field comments follow Kubernetes API conventions (describe semantics not implementation)
- Add package-level documentation to `api/v1alpha1/`

## API Best Practices

- Audit for compliance with [Kubernetes API conventions](https://github.com/kubernetes/community/blob/master/contributors/devel/sig-architecture/api-conventions.md)
- Review printer columns for usefulness
- Review status conditions for compliance with metav1.Condition conventions
- Ensure all optional fields use pointer types with `omitempty`
- Review field naming for consistency across types

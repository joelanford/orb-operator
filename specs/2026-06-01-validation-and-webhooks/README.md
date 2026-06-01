---
status: idea
---
# Validation and Webhooks

Implement admission validation for all three resource types. Key constraints: COSR immutability (group, revision, phases, collisionProtection cannot change after creation), lifecycleState one-way transition (Active → Archived only), required field validation, and structural validation of phase/assertion/collision-protection definitions. Validation may be implemented as validating webhooks or CEL validation rules on the CRDs.

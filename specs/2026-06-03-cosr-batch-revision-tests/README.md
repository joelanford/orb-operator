---
status: idea
---
# COSR Batch Revision Application Tests

Test scenarios for simultaneously applying a batch of COSR revisions, potentially out of order. Verifies the controller correctly resolves which revision is authoritative when multiple revisions in the same group are created before any become Available. May require refactoring the e2e step definitions to support creating multiple COSRs within a single scenario without resetting builder state.

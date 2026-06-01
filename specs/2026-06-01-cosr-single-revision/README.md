---
status: idea
---
# COSR Controller — Single Revision Lifecycle

Implement the ClusterObjectSetRevision controller for the single-revision case: resolve ClusterObjectSlice refs, create managed objects from phases in dependency order, evaluate per-object inline assertions (ConditionEqual, FieldsEqual, FieldValue) and built-in assertions, gate phase progression on assertion success, and set status conditions. No multi-revision handoffs — just one active COSR managing its objects.

---
status: idea
---
# Lightweight Teardown

Replace full object resolution (slice fetching, JSON unmarshalling, hash verification) in the COS teardown path with lightweight identity extraction that only needs apiVersion, kind, name, and namespace from each PhaseObject.

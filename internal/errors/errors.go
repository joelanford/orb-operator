package errors

type ObjectResolutionError struct{ Err error }

func (e *ObjectResolutionError) Error() string { return e.Err.Error() }
func (e *ObjectResolutionError) Unwrap() error { return e.Err }

type InternalError struct{ Err error }

func (e *InternalError) Error() string { return e.Err.Error() }
func (e *InternalError) Unwrap() error { return e.Err }

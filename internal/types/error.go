package types

type StatusError struct {
	Error  error
	Status int
}

func (e StatusError) Unwrap() error {
	return e.Error
}

func (e StatusError) HTTPStatus() int {
	return e.Status
}

func NewStatusError(err error, status int) StatusError {
	return StatusError{
		Error:  err,
		Status: status,
	}
}
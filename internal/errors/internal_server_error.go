package errors

type InternalServerError struct {
	message string
}

func NewInternalServerError(message string) InternalServerError {
	return InternalServerError{
		message: message,
	}
}

func (e InternalServerError) Error() string {
	return e.message
}
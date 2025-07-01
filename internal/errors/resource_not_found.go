package errors

type ResourceNotFound struct {
	message string
}

func NewResourceNotFound(message string) ResourceNotFound {
	return ResourceNotFound{
		message: message,
	}
}

func (e ResourceNotFound) Error() string {
	return e.message
}
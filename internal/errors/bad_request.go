package errors

type BadRequest struct {
	message string
}

func NewBadRequest(message string) BadRequest {
	return BadRequest{
		message: message,
	}
}

func (e BadRequest) Error() string {
	return e.message
}
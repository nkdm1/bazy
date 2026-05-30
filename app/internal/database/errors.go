package database

const ErrCodeDuplicateEntry = 1062

type ErrorDatabase interface {
	error
	Code()
}

type databaseError struct {
	Message string
	Status  int
}

func (e *databaseError) Error() string {
	return e.Message
}

func (e *databaseError) Code() int {
	return e.Status
}

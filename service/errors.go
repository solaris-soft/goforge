package service

type FieldErrors map[string]string

func (f FieldErrors) Error() string {
	return "invalid input"
}

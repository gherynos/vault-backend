package store

type ItemNotFoundError struct{}

func (e *ItemNotFoundError) Error() string {

	return "item not found"
}

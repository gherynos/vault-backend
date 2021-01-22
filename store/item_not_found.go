package store

// ItemNotFoundError is an error returned when a requested item is not present in a Store.
type ItemNotFoundError struct{}

func (e *ItemNotFoundError) Error() string {

	return "item not found"
}

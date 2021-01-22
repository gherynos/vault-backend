package store

// Pool is a collection of Stores.
// The Get method creates a new Store for the given identifier if not already present.
// Trying to delete a Store via an unknown identifier has no effect.
type Pool interface {
	Get(identifier string) (Store, error)

	Delete(identifier string)
}

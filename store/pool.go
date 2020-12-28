package store

type Pool interface {
	Get(identifier string) (Store, error)

	Delete(identifier string)
}

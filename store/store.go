package store

// Store is a collection of byte arrays.
// The byte arrays can be stored, retrieved and deleted by name.
type Store interface {
	SetBin(name string, data []byte) error

	GetBin(name string) (out []byte, err error)

	Delete(name string) error
}

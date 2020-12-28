package store

type Store interface {
	SetBin(name string, data []byte) error

	GetBin(name string) (out []byte, err error)

	Delete(name string) error
}

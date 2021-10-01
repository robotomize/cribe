package botstate

type Backend interface {
	Get(k string) ([]byte, error)
	Set(k string, v []byte) error
	Delete(k string) error
}

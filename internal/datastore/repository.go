package datastore

type IRepository interface {
	Get(key string) (bool, error)
	Set(key string, count uint16) error
	Incr(key string) error
	Decr(key string) error
	Clear() error
	SetReady(isReady bool)
	Ready() bool
}

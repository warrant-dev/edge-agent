package datastore

import (
	"log"
	"sync"
)

const DATASTORE_MEMORY = "memory"

type WarrantCache struct {
	hashCount map[string]uint16
	lock      sync.Mutex
}

func (cache *WarrantCache) Contains(key string) bool {
	cache.lock.Lock()
	defer cache.lock.Unlock()

	_, ok := cache.hashCount[key]
	return ok
}

func (cache *WarrantCache) Set(key string, count uint16) {
	cache.lock.Lock()
	defer cache.lock.Unlock()

	cache.hashCount[key] = count
}

func (cache *WarrantCache) Incr(key string) {
	cache.lock.Lock()
	defer cache.lock.Unlock()

	if cache.Contains(key) {
		cache.hashCount[key] = cache.hashCount[key] + 1
		return
	}

	cache.hashCount[key] = 1
}

func (cache *WarrantCache) Decr(key string) {
	cache.lock.Lock()
	defer cache.lock.Unlock()

	if cache.Contains(key) {
		count := cache.hashCount[key]
		if count <= 1 {
			delete(cache.hashCount, key)
		} else {
			cache.hashCount[key] = cache.hashCount[key] - 1
		}
	}
}

func (cache *WarrantCache) Clear() {
	log.Printf("Clearing cache")

	cache.lock.Lock()
	defer cache.lock.Unlock()
	cache.hashCount = make(map[string]uint16)
}

type MemoryRepository struct {
	cache *WarrantCache
	lock  sync.Mutex
	ready bool
}

func NewMemoryRepository() *MemoryRepository {
	return &MemoryRepository{
		cache: &WarrantCache{
			hashCount: make(map[string]uint16),
		},
		ready: true,
	}
}

func (repo *MemoryRepository) Get(key string) (bool, error) {
	return repo.cache.Contains(key), nil
}

func (repo *MemoryRepository) Set(key string, count uint16) error {
	repo.cache.Set(key, count)
	return nil
}

func (repo *MemoryRepository) Incr(key string) error {
	repo.cache.Incr(key)
	return nil
}

func (repo *MemoryRepository) Decr(key string) error {
	repo.cache.Decr(key)
	return nil
}

func (repo *MemoryRepository) Clear() error {
	repo.cache.hashCount = make(map[string]uint16)
	return nil
}

func (repo *MemoryRepository) SetReady(newReady bool) {
	repo.lock.Lock()
	defer repo.lock.Unlock()

	repo.ready = newReady
}

func (repo *MemoryRepository) Ready() bool {
	repo.lock.Lock()
	defer repo.lock.Unlock()

	return repo.ready
}

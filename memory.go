// Copyright 2024 WorkOS, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package edge

import (
	"sync"
)

type WarrantCache struct {
	hashCount map[string]uint16
	lock      sync.RWMutex
}

func (cache *WarrantCache) Contains(key string) bool {
	cache.lock.RLock()
	defer cache.lock.RUnlock()

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

func (cache *WarrantCache) Update(warrants WarrantSet) error {
	cache.lock.Lock()
	defer cache.lock.Unlock()

	// iterate over existing records and remove any that no longer exist
	for key := range cache.hashCount {
		if warrants.Has(key) {
			cache.Set(key, warrants.Get(key))
		} else {
			delete(cache.hashCount, key)
		}
	}

	// add any newly created records
	for key, value := range warrants {
		cache.Set(key, value)
	}

	return nil
}

func (cache *WarrantCache) Clear() {
	cache.lock.Lock()
	defer cache.lock.Unlock()
	cache.hashCount = make(map[string]uint16)
}

type MemoryRepository struct {
	cache *WarrantCache
	lock  sync.RWMutex
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

func (repo *MemoryRepository) Update(warrants WarrantSet) error {
	return repo.cache.Update(warrants)
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
	repo.lock.RLock()
	defer repo.lock.RUnlock()

	return repo.ready
}

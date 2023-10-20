// Copyright 2023 Forerunner Labs, Inc.
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
	"fmt"
	"strings"
	"sync"

	"github.com/pkg/errors"

	"github.com/go-redis/redis"
)

type RedisRepositoryConfig struct {
	Hostname string
	Password string
	Port     string
}

type RedisRepository struct {
	client *redis.Client
	ready  bool
	lock   sync.Mutex
}

func NewRedisRepository(config RedisRepositoryConfig) (*RedisRepository, error) {
	hostname := config.Hostname
	if config.Hostname == "" {
		hostname = "127.0.0.1"
	}

	port := config.Port
	if port == "" {
		port = "6379"
	}

	var connectionString string
	if config.Password != "" {
		connectionString = fmt.Sprintf("rediss://default:%s@%s:%s/1", config.Password, hostname, port)
	} else {
		connectionString = fmt.Sprintf("redis://default:%s@%s:%s/1", config.Password, hostname, port)
	}

	opt, err := redis.ParseURL(connectionString)
	if err != nil {
		return nil, errors.Wrap(err, fmt.Sprintf("invalid connection string %s", connectionString))
	}

	rdb := redis.NewClient(opt)
	_, err = rdb.Ping().Result()
	if err != nil {
		return nil, errors.Wrap(err, "Unable to ping redis. Check your credentials.")
	}

	return &RedisRepository{
		client: rdb,
		ready:  true,
	}, nil
}

func (repo *RedisRepository) Get(key string) (bool, error) {
	_, err := repo.client.Get(repo.keyWithNamespace(key)).Result()
	if err == redis.Nil {
		return false, nil
	}
	if err != nil {
		return false, errors.Wrap(err, "error getting key from redis")
	}

	return true, nil
}

func (repo *RedisRepository) Set(key string, count uint16) error {
	_, err := repo.client.Set(repo.keyWithNamespace(key), count, 0).Result()
	if err != nil {
		return errors.Wrap(err, "error setting key in redis")
	}

	return nil
}

func (repo *RedisRepository) Incr(key string) error {
	_, err := repo.client.Incr(repo.keyWithNamespace(key)).Result()
	if err != nil {
		return errors.Wrap(err, "error incrementing key in redis")
	}

	return nil
}

func (repo *RedisRepository) Decr(key string) error {
	namespacedKey := repo.keyWithNamespace(key)
	maxRetries := 10
	decrementAndRemoveFunc := func(tx *redis.Tx) error {
		count, err := repo.client.Decr(namespacedKey).Result()
		if err != nil {
			return errors.Wrap(err, "error decrementing key in redis")
		}

		if count <= 0 {
			_, err = repo.client.Del(namespacedKey).Result()
			if err != nil {
				return errors.Wrap(err, "error deleting key from redis")
			}
		}

		return nil
	}

	for i := 0; i < maxRetries; i++ {
		err := repo.client.Watch(decrementAndRemoveFunc, namespacedKey)
		if err == redis.TxFailedErr {
			// Retry
			continue
		} else if err != nil {
			return errors.Wrap(err, "error calling watch in redis")
		}

		return nil
	}

	return errors.New(fmt.Sprintf("unable to acquire lock to remove %s from cache", key))
}

func (repo *RedisRepository) Update(warrants WarrantSet) error {
	prefix := fmt.Sprintf("%s*", repo.getNamespace())
	iter := repo.client.Scan(0, prefix, 0).Iterator()

	// iterate over existing records and remove any that no longer exist
	for iter.Next() {
		keyWithNamespace := iter.Val()
		keyWithoutNamespace := repo.keyWithoutNamespace(keyWithNamespace)
		if warrants.Has(keyWithoutNamespace) {
			err := repo.Set(keyWithoutNamespace, warrants.Get(keyWithoutNamespace))
			if err != nil {
				return errors.Wrap(err, "error updating key in redis")
			}
		} else {
			err := repo.client.Del(keyWithNamespace).Err()
			if err != nil {
				return errors.Wrap(err, "error deleting key from redis")
			}
		}
	}
	if err := iter.Err(); err != nil {
		return errors.Wrap(err, "error iterating over keys in redis")
	}

	// add any newly created records
	for keyWithoutNamespace, count := range warrants {
		err := repo.Set(keyWithoutNamespace, count)
		if err != nil {
			return errors.Wrap(err, "error updating key in redis")
		}
	}

	return nil
}

func (repo *RedisRepository) Clear() error {
	prefix := fmt.Sprintf("%s*", repo.getNamespace())
	iter := repo.client.Scan(0, prefix, 0).Iterator()

	for iter.Next() {
		key := iter.Val()
		err := repo.client.Del(key).Err()
		if err != nil {
			return errors.Wrap(err, "error deleting key from redis")
		}
	}
	if err := iter.Err(); err != nil {
		return errors.Wrap(err, "error iterating over keys in redis")
	}

	return nil
}

func (repo *RedisRepository) SetReady(newReady bool) {
	repo.lock.Lock()
	defer repo.lock.Unlock()

	repo.ready = newReady
}

func (repo *RedisRepository) Ready() bool {
	repo.lock.Lock()
	defer repo.lock.Unlock()

	return repo.ready
}

func (repo *RedisRepository) getNamespace() string {
	return "warrant"
}

func (repo *RedisRepository) keyWithNamespace(key string) string {
	return fmt.Sprintf("%s:%s", repo.getNamespace(), key)
}

func (repo *RedisRepository) keyWithoutNamespace(key string) string {
	return strings.TrimPrefix(key, fmt.Sprintf("%s:", repo.getNamespace()))
}

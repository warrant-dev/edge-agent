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

package datastore

import (
	"fmt"
	"log"
	"sync"

	"github.com/go-redis/redis"
)

const DATASTORE_REDIS = "redis"

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

func NewRedisRepository(config RedisRepositoryConfig) *RedisRepository {
	var connectionString string

	if config.Hostname == "" {
		log.Fatal("Must set redis hostname")
	}

	port := config.Port
	if port == "" {
		port = "6379"
	}

	if config.Password != "" {
		connectionString = fmt.Sprintf("rediss://default:%s@%s:%s/1", config.Password, config.Hostname, port)
	} else {
		connectionString = fmt.Sprintf("redis://default:%s@%s:%s/1", config.Password, config.Hostname, port)
	}

	opt, err := redis.ParseURL(connectionString)
	if err != nil {
		panic(err)
	}

	rdb := redis.NewClient(opt)
	_, err = rdb.Ping().Result()
	if err != nil {
		log.Println("Unable to ping redis. Check your credentials.")
		log.Println("Shutting down.")
		log.Fatal(err)
	}

	return &RedisRepository{
		client: rdb,
		ready:  true,
	}
}

func (repo *RedisRepository) Get(key string) (bool, error) {
	namespacedKey := fmt.Sprintf("%s:%s", repo.getNamespace(), key)
	_, err := repo.client.Get(namespacedKey).Result()
	if err == redis.Nil {
		return false, nil
	}
	if err != nil {
		return false, err
	}

	return true, nil
}

func (repo *RedisRepository) Set(key string, count uint16) error {
	namespacedKey := fmt.Sprintf("%s:%s", repo.getNamespace(), key)
	_, err := repo.client.Set(namespacedKey, count, 0).Result()
	if err != nil {
		return err
	}

	return nil
}

func (repo *RedisRepository) Incr(key string) error {
	namespacedKey := fmt.Sprintf("%s:%s", repo.getNamespace(), key)
	_, err := repo.client.Incr(namespacedKey).Result()
	if err != nil {
		return err
	}

	return nil
}

func (repo *RedisRepository) Decr(key string) error {
	namespacedKey := fmt.Sprintf("%s:%s", repo.getNamespace(), key)
	maxRetries := 10
	decrementAndRemoveFunc := func(tx *redis.Tx) error {
		count, err := repo.client.Decr(namespacedKey).Result()
		if err != nil {
			return err
		}

		if count <= 0 {
			_, err = repo.client.Del(namespacedKey).Result()
			if err != nil {
				return err
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
			return err
		}

		return nil
	}

	return fmt.Errorf("unable to acquire lock to remove %s from cache", key)
}

func (repo *RedisRepository) Clear() error {
	prefix := fmt.Sprintf("%s*", repo.getNamespace())
	iter := repo.client.Scan(0, prefix, 0).Iterator()

	for iter.Next() {
		key := iter.Val()
		err := repo.client.Del(key).Err()
		if err != nil {
			return err
		}
	}
	if err := iter.Err(); err != nil {
		return err
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

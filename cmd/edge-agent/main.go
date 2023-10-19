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

package main

import (
	"errors"
	"log"
	"os"

	"github.com/warrant-dev/edge"

	"github.com/spf13/viper"
)

const (
	PropertyApiEndpoint       = "API_ENDPOINT"
	PropertyApiKey            = "API_KEY"
	PropertyDatastore         = "DATASTORE"
	PropertyRedisHostname     = "REDIS_HOSTNAME"
	PropertyRedisPassword     = "REDIS_PASSWORD"
	PropertyRedisPort         = "REDIS_PORT"
	PropertyStreamingEndpoint = "STREAMING_ENDPOINT"
	PropertyUpdateStrategy    = "UPDATE_STRATEGY"
)

var ErrInvalidDatastoreType = errors.New("invalid datastore type")

func main() {
	viper.SetConfigName("agent")
	viper.SetConfigType("properties")
	viper.AddConfigPath(".")
	viper.SetDefault(PropertyApiKey, os.Getenv(PropertyApiKey))
	viper.SetDefault(PropertyApiEndpoint, os.Getenv(PropertyApiEndpoint))
	viper.SetDefault(PropertyUpdateStrategy, os.Getenv(PropertyUpdateStrategy))
	viper.SetDefault(PropertyStreamingEndpoint, os.Getenv(PropertyStreamingEndpoint))
	viper.SetDefault(PropertyDatastore, os.Getenv(PropertyDatastore))
	viper.SetDefault(PropertyRedisHostname, os.Getenv(PropertyRedisHostname))
	viper.SetDefault(PropertyRedisPort, os.Getenv(PropertyRedisPort))
	viper.SetDefault(PropertyRedisPassword, os.Getenv(PropertyRedisPassword))

	if err := viper.ReadInConfig(); err != nil {
		if errors.Is(err, viper.ConfigFileNotFoundError{}) {
			log.Fatal(err)
		}
	}

	var repo edge.IRepository
	var err error
	switch viper.GetString(PropertyDatastore) {
	case "":
		repo = edge.NewMemoryRepository()
	case edge.DatastoreMemory:
		repo = edge.NewMemoryRepository()
	case edge.DatastoreRedis:
		repo, err = edge.NewRedisRepository(edge.RedisRepositoryConfig{
			Hostname: viper.GetString(PropertyRedisHostname),
			Password: viper.GetString(PropertyRedisPassword),
			Port:     viper.GetString(PropertyRedisPort),
		})
		if err != nil {
			log.Fatal(err)
		}
	default:
		log.Fatal(ErrInvalidDatastoreType)
	}

	server, err := edge.NewServer(edge.ServerConfig{
		Port:       3000,
		ApiKey:     viper.GetString(PropertyApiKey),
		Repository: repo,
	})
	if err != nil {
		log.Fatal(err)
	}

	client, err := edge.NewClient(edge.ClientConfig{
		ApiKey:            viper.GetString(PropertyApiKey),
		ApiEndpoint:       viper.GetString(PropertyApiEndpoint),
		StreamingEndpoint: viper.GetString(PropertyStreamingEndpoint),
		UpdateStrategy:    viper.GetString(PropertyUpdateStrategy),
		Repository:        repo,
	})
	if err != nil {
		log.Fatal(err)
	}

	go func() {
		log.Fatal(client.Run())
	}()
	log.Fatal(server.Run())
}

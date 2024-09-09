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
	PropertyRedisDatabase     = "REDIS_DATABASE"
	PropertyStreamingEndpoint = "STREAMING_ENDPOINT"
	PropertyUpdateStrategy    = "UPDATE_STRATEGY"
	PropertyPollingFrequency  = "POLLING_FREQUENCY"
	PropertyReadOnly          = "READ_ONLY"
)

var ErrInvalidDatastoreType = errors.New("invalid datastore type")

func main() {
	viper.SetConfigName("agent")
	viper.SetConfigType("properties")
	viper.AddConfigPath(".")
	viper.SetDefault(PropertyApiKey, os.Getenv(PropertyApiKey))
	viper.SetDefault(PropertyApiEndpoint, os.Getenv(PropertyApiEndpoint))
	viper.SetDefault(PropertyUpdateStrategy, os.Getenv(PropertyUpdateStrategy))
	viper.SetDefault(PropertyPollingFrequency, os.Getenv(PropertyPollingFrequency))
	viper.SetDefault(PropertyStreamingEndpoint, os.Getenv(PropertyStreamingEndpoint))
	viper.SetDefault(PropertyDatastore, os.Getenv(PropertyDatastore))
	viper.SetDefault(PropertyRedisHostname, os.Getenv(PropertyRedisHostname))
	viper.SetDefault(PropertyRedisPort, os.Getenv(PropertyRedisPort))
	viper.SetDefault(PropertyRedisPassword, os.Getenv(PropertyRedisPassword))
	viper.SetDefault(PropertyRedisDatabase, os.Getenv(PropertyRedisDatabase))
	viper.SetDefault(PropertyReadOnly, os.Getenv(PropertyReadOnly))

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
			Database: viper.GetInt(PropertyRedisDatabase),
		})
		if err != nil {
			log.Fatal(err)
		}
	default:
		log.Fatal(ErrInvalidDatastoreType)
	}

	// initialize and start client
	if !viper.GetBool(PropertyReadOnly) {
		log.Println("Starting edge agent")
		client, err := edge.NewClient(edge.ClientConfig{
			ApiKey:            viper.GetString(PropertyApiKey),
			ApiEndpoint:       viper.GetString(PropertyApiEndpoint),
			StreamingEndpoint: viper.GetString(PropertyStreamingEndpoint),
			UpdateStrategy:    viper.GetString(PropertyUpdateStrategy),
			PollingFrequency:  viper.GetInt(PropertyPollingFrequency),
			Repository:        repo,
		})
		if err != nil {
			log.Fatal(err)
		}

		go func() {
			log.Fatal(client.Run())
		}()
	} else {
		log.Println("Starting edge agent in read-only mode")
	}

	// initialize and start server
	server, err := edge.NewServer(edge.ServerConfig{
		Port:       3000,
		ApiKey:     viper.GetString(PropertyApiKey),
		Repository: repo,
	})
	if err != nil {
		log.Fatal(err)
	}

	log.Fatal(server.Run())
}

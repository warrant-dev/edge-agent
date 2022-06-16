package main

import (
	"log"
	"os"

	"github.com/spf13/viper"
	"github.com/warrant-dev/edge/internal/datastore"
	"github.com/warrant-dev/edge/internal/edge"
)

const PROPERTY_API_ENDPOINT = "API_ENDPOINT"
const PROPERTY_API_KEY = "API_KEY"
const PROPERTY_DATASTORE = "DATASTORE"
const PROPERTY_REDIS_HOSTNAME = "REDIS_HOSTNAME"
const PROPERTY_REDIS_PASSWORD = "REDIS_PASSWORD"
const PROPERTY_REDIS_PORT = "REDIS_PORT"
const PROPERTY_STREAMING_ENDPOINT = "STREAMING_ENDPOINT"

func main() {
	viper.SetConfigName("agent")
	viper.SetConfigType("properties")
	viper.AddConfigPath(".")
	viper.SetDefault(PROPERTY_API_KEY, os.Getenv(PROPERTY_API_KEY))
	viper.SetDefault(PROPERTY_API_ENDPOINT, os.Getenv(PROPERTY_API_ENDPOINT))
	viper.SetDefault(PROPERTY_STREAMING_ENDPOINT, os.Getenv(PROPERTY_STREAMING_ENDPOINT))
	viper.SetDefault(PROPERTY_DATASTORE, os.Getenv(PROPERTY_DATASTORE))
	viper.SetDefault(PROPERTY_REDIS_HOSTNAME, os.Getenv(PROPERTY_REDIS_HOSTNAME))
	viper.SetDefault(PROPERTY_REDIS_PORT, os.Getenv(PROPERTY_REDIS_PORT))
	viper.SetDefault(PROPERTY_REDIS_PASSWORD, os.Getenv(PROPERTY_REDIS_PASSWORD))

	if err := viper.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			log.Fatal(err)
		}
	}

	var repo datastore.IRepository
	apiKey := viper.GetString(PROPERTY_API_KEY)
	storeType := viper.GetString(PROPERTY_DATASTORE)
	switch storeType {
	case "":
		repo = datastore.NewMemoryRepository()
	case datastore.DATASTORE_MEMORY:
		repo = datastore.NewMemoryRepository()
	case datastore.DATASTORE_REDIS:
		repo = datastore.NewRedisRepository(datastore.RedisRepositoryConfig{
			Hostname: viper.GetString(PROPERTY_REDIS_HOSTNAME),
			Password: viper.GetString(PROPERTY_REDIS_PASSWORD),
			Port:     viper.GetString(PROPERTY_REDIS_PORT),
		})
	default:
		log.Fatal("Invalid storeType provided")
	}

	edgeServer := edge.NewServer(edge.ServerConfig{
		Port:       3000,
		ApiKey:     apiKey,
		Repository: repo,
	})

	apiEndpoint := "https://api.warrant.dev/v1"
	if viper.GetString(PROPERTY_API_ENDPOINT) != "" {
		apiEndpoint = viper.GetString(PROPERTY_API_ENDPOINT)
	}

	streamingEndpoint := "https://stream.warrant.dev/v1"
	if viper.GetString(PROPERTY_STREAMING_ENDPOINT) != "" {
		streamingEndpoint = viper.GetString(PROPERTY_STREAMING_ENDPOINT)
	}

	edgeClient := edge.NewClient(edge.ClientConfig{
		ApiKey:            apiKey,
		ApiEndpoint:       apiEndpoint,
		StreamingEndpoint: streamingEndpoint,
		Repository:        repo,
	})

	go edgeClient.Run()
	edgeServer.Run()
}

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
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"time"

	"github.com/pkg/errors"

	"github.com/r3labs/sse"
	"gopkg.in/cenkalti/backoff.v1"
)

const (
	DefaultApiEndpoint       = "https://api.warrant.dev"
	DefaultStreamingEndpoint = "https://stream.warrant.dev/v1"
	DefaultPollingFrequency  = 10

	EventTypeSetWarrants    = "set_warrants"
	EventTypeDeleteWarrants = "del_warrants"
	EventTypeResetWarrants  = "reset_warrants"
	EventTypeShutdown       = "shutdown"

	UpdateStrategyPolling   = "POLLING"
	UpdateStrategyStreaming = "STREAMING"
)

var (
	ErrInvalidUpdateStrategy   = errors.New("invalid update strategy")
	ErrInvalidPollingFrequency = errors.New("invalid polling frequency (cannot be < 10s)")
	ErrMissingApiKey           = errors.New("missing API key")
)

type ClientConfig struct {
	ApiKey            string
	ApiEndpoint       string
	UpdateStrategy    string
	StreamingEndpoint string
	PollingFrequency  int
	Repository        IRepository
}

type Client struct {
	config          ClientConfig
	streamingClient *sse.Client
}

func NewClient(conf ClientConfig) (*Client, error) {
	config := ClientConfig{
		ApiEndpoint:       DefaultApiEndpoint,
		StreamingEndpoint: DefaultStreamingEndpoint,
		UpdateStrategy:    UpdateStrategyPolling,
		PollingFrequency:  DefaultPollingFrequency,
		Repository:        conf.Repository,
	}

	if conf.ApiKey == "" {
		return nil, ErrMissingApiKey
	} else {
		config.ApiKey = conf.ApiKey
	}

	if conf.ApiEndpoint != "" {
		config.ApiEndpoint = conf.ApiEndpoint
	}

	if conf.StreamingEndpoint != "" {
		config.StreamingEndpoint = conf.StreamingEndpoint
	}

	if conf.UpdateStrategy != "" {
		config.UpdateStrategy = conf.UpdateStrategy
	}

	if conf.PollingFrequency != 0 {
		config.PollingFrequency = conf.PollingFrequency
	} else if config.PollingFrequency < 10 {
		return nil, ErrInvalidPollingFrequency
	}

	if config.UpdateStrategy == UpdateStrategyStreaming {
		streamingClient := sse.NewClient(fmt.Sprintf("%s/events", config.StreamingEndpoint))
		streamingClient.Headers["Authorization"] = fmt.Sprintf("ApiKey %s", config.ApiKey)
		streamingClient.ReconnectStrategy = backoff.WithMaxTries(backoff.NewExponentialBackOff(), 10)
		streamingClient.ReconnectNotify = reconnectNotify

		return &Client{
			config:          config,
			streamingClient: streamingClient,
		}, nil
	} else if config.UpdateStrategy == UpdateStrategyPolling || config.UpdateStrategy == "" {
		return &Client{
			config: config,
		}, nil
	} else {
		return nil, ErrInvalidUpdateStrategy
	}
}

func (client *Client) Run() error {
	log.Println("Starting edge client...")
	err := client.initialize()
	if err != nil {
		return errors.Wrap(err, "error trying to initialize edge client")
	}

	log.Println("Edge client initialized.")

	/*if client.config.UpdateStrategy == UpdateStrategyStreaming {
		err = client.connect()
		if err != nil {
			return errors.Wrap(err, "error streaming warrant updates")
		}
	} else*/if client.config.UpdateStrategy == UpdateStrategyPolling {
		err = client.poll()
		if err != nil {
			return errors.Wrap(err, "error polling warrant updates")
		}
	} else {
		return ErrInvalidUpdateStrategy
	}

	return nil
}

func (client *Client) initialize() error {
	client.config.Repository.SetReady(false)
	err := client.config.Repository.Clear()
	if err != nil {
		return errors.Wrap(err, "error clearing cache")
	}

	warrants, err := client.getWarrants()
	if err != nil {
		return errors.Wrap(err, "error getting warrants")
	}

	for warrant, count := range warrants {
		err := client.config.Repository.Set(warrant, count)
		if err != nil {
			return errors.Wrapf(err, "error setting warrant %s in cache", warrant)
		}
	}

	client.config.Repository.SetReady(true)
	return nil
}

func (client *Client) connect() error {
	client.streamingClient.OnDisconnect(client.restart)
	err := client.streamingClient.Subscribe(client.config.ApiKey, client.processEvent)
	if err != nil {
		return err
	}

	return nil
}

func (client *Client) poll() error {
	for {
		time.Sleep(time.Second * time.Duration(client.config.PollingFrequency))
		log.Println("fetching latest warrants")
		warrants, err := client.getWarrants()
		if err != nil {
			return errors.Wrap(err, "error getting warrants")
		}

		err = client.config.Repository.Update(warrants)
		if err != nil {
			return errors.Wrap(err, "error updating warrants")
		}
	}
}

func (client *Client) getWarrants() (WarrantSet, error) {
	resp, err := client.makeRequest("GET", fmt.Sprintf("%s/expand", ApiVersion), nil)
	if err != nil {
		return nil, err
	}

	respStatus := resp.StatusCode
	if respStatus < 200 || respStatus >= 400 {
		msg, err := io.ReadAll(resp.Body)
		if err != nil {
			return nil, errors.Wrap(err, "error reading response from server")
		}

		return nil, errors.New(fmt.Sprintf("received HTTP %d: %s", respStatus, string(msg)))
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, errors.Wrap(err, "error reading response from server")
	}

	var warrants WarrantSet
	err = json.Unmarshal(body, &warrants)
	if err != nil {
		return nil, errors.Wrap(err, "received invalid response from server")
	}

	return warrants, nil
}

func (client *Client) makeRequest(method string, requestUri string, payload interface{}) (*http.Response, error) {
	postBody, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}

	requestBody := bytes.NewBuffer(postBody)
	req, err := http.NewRequest(method, fmt.Sprintf("%s/%s", client.config.ApiEndpoint, requestUri), requestBody)
	if err != nil {
		return nil, errors.Wrap(err, "error creating request object")
	}

	req.Header.Add("Authorization", fmt.Sprintf("ApiKey %s", client.config.ApiKey))
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, errors.Wrap(err, "error making request to server")
	}

	return resp, nil
}

func (client *Client) processEvent(event *sse.Event) {
	var err error
	switch string(event.Event) {
	case EventTypeSetWarrants:
		err = client.processSetWarrants(event)
	case EventTypeDeleteWarrants:
		err = client.processDeleteWarrants(event)
	case EventTypeResetWarrants:
		err = client.initialize()
	case EventTypeShutdown:
		log.Fatal("Shutdown event received. Shutting down.")
	}

	if err != nil {
		log.Println(errors.Wrapf(err, "error processing event %s.", event.Event))
	}
}

func (client *Client) processSetWarrants(event *sse.Event) error {
	var warrants WarrantSet
	err := json.Unmarshal(event.Data, &warrants)
	if err != nil {
		return errors.Wrapf(err, "invalid event data %s", event.Data)
	}

	for w, count := range warrants {
		var i uint16 = 0
		for ; i < count; i++ {
			err := client.config.Repository.Incr(w)
			if err != nil {
				return errors.Wrapf(err, "error setting warrant %s in cache", w)
			}
		}
	}

	return nil
}

func (client *Client) processDeleteWarrants(event *sse.Event) error {
	var warrants WarrantSet
	err := json.Unmarshal(event.Data, &warrants)
	if err != nil {
		return errors.Wrapf(err, "invalid event data %s", event.Data)
	}

	for w, count := range warrants {
		var i uint16 = 0
		for ; i < count; i++ {
			err := client.config.Repository.Decr(w)
			if err != nil {
				return errors.Wrapf(err, "error removing warrant %s from cache", w)
			}
		}
	}

	return nil
}

func (client *Client) restart(c *sse.Client) {
	log.Printf("Disconnected from %s.", client.config.StreamingEndpoint)
	client.config.Repository.SetReady(false)

	log.Println("Attempting to reconnect...")
	err := client.Run()
	if err != nil {
		log.Fatal(errors.Wrap(err, "error restarting client"))
	}
}

func reconnectNotify(err error, d time.Duration) {
	log.Println("Unable to connect.")
	log.Println(err)
	log.Printf("Retrying in %s", d)
}

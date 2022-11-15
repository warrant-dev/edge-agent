package edge

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"time"

	"github.com/r3labs/sse"
	"github.com/warrant-dev/edge/internal/datastore"
	"github.com/warrant-dev/edge/internal/warrant"
	"gopkg.in/cenkalti/backoff.v1"
)

const EVENT_SET_WARRANTS = "set_warrants"
const EVENT_DEL_WARRANTS = "del_warrants"
const EVENT_RESET_WARRANTS = "reset_warrants"
const SHUTDOWN = "shutdown"

type ClientConfig struct {
	ApiKey            string
	ApiEndpoint       string
	StreamingEndpoint string
	Repository        datastore.IRepository
}

type Client struct {
	config    ClientConfig
	sseClient *sse.Client
}

func NewClient(config ClientConfig) *Client {
	sseClient := sse.NewClient(fmt.Sprintf("%s/events", config.StreamingEndpoint))
	sseClient.Headers["Authorization"] = fmt.Sprintf("ApiKey %s", config.ApiKey)
	sseClient.ReconnectStrategy = backoff.WithMaxTries(backoff.NewExponentialBackOff(), 10)
	sseClient.ReconnectNotify = reconnectNotify

	return &Client{
		config:    config,
		sseClient: sseClient,
	}
}

func (client *Client) Run() {
	log.Println("Intializing edge client...")
	err := client.initialize()
	if err != nil {
		log.Println("Unable to initialize edge client.")
		log.Println(err)
		log.Fatal("Shutting down.")
	}

	log.Println("Edge client initialized.")
	log.Printf("Connecting to %s", client.config.StreamingEndpoint)
	err = client.connect()
	if err != nil {
		log.Println("Unable to connect.")
		log.Println(err)
		log.Fatal("Shutting down.")
	}
}

func (client *Client) initialize() error {
	client.config.Repository.SetReady(false)
	client.config.Repository.Clear()

	resp, err := client.makeRequest("GET", "/expand", nil)
	if err != nil {
		return err
	}

	respStatus := resp.StatusCode
	if respStatus < 200 || respStatus >= 400 {
		msg, _ := ioutil.ReadAll(resp.Body)
		return fmt.Errorf("HTTP %d %s", respStatus, string(msg))
	}

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("error reading response %w", err)
	}

	var warrants warrant.WarrantSet
	err = json.Unmarshal([]byte(body), &warrants)
	if err != nil {
		return fmt.Errorf("invalid response from server %w", err)
	}

	for warrant, count := range warrants {
		err := client.config.Repository.Set(warrant, count)
		if err != nil {
			return fmt.Errorf("unable to set warrant %s", warrant)
		}
	}

	client.config.Repository.SetReady(true)
	return nil
}

func (client *Client) connect() error {
	client.sseClient.OnDisconnect(client.restart)
	err := client.sseClient.Subscribe(client.config.ApiKey, client.processEvent)
	if err != nil {
		return err
	}

	return nil
}

func (client *Client) makeRequest(method string, requestUri string, payload interface{}) (*http.Response, error) {
	postBody, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}

	requestBody := bytes.NewBuffer(postBody)
	req, err := http.NewRequest(method, fmt.Sprintf("%s%s", client.config.ApiEndpoint, requestUri), requestBody)
	if err != nil {
		return nil, fmt.Errorf("unable to create request %w", err)
	}

	req.Header.Add("Authorization", fmt.Sprintf("ApiKey %s", client.config.ApiKey))
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("error making request %w", err)
	}

	return resp, nil
}

func (client *Client) processEvent(event *sse.Event) {
	var err error
	switch string(event.Event) {
	case EVENT_SET_WARRANTS:
		err = client.processSetWarrants(event)
	case EVENT_DEL_WARRANTS:
		err = client.processDelWarrants(event)
	case EVENT_RESET_WARRANTS:
		err = client.initialize()
	case SHUTDOWN:
		log.Fatal("Shutting down.")
	}

	if err != nil {
		log.Printf("unable to process event %s.", event.Event)
		log.Println(err)
	}
}

func (client *Client) processSetWarrants(event *sse.Event) error {
	var warrants warrant.WarrantSet
	err := json.Unmarshal([]byte(event.Data), &warrants)
	if err != nil {
		log.Printf("Invalid event data %s", event.Data)
		return err
	}

	for warrant, count := range warrants {
		var i uint16 = 0
		for ; i < count; i++ {
			err := client.config.Repository.Incr(warrant)
			if err != nil {
				return fmt.Errorf("unable to incr cache for warrant %s %w", warrant, err)
			}
		}
	}

	return nil
}

func (client *Client) processDelWarrants(event *sse.Event) error {
	var warrants warrant.WarrantSet
	err := json.Unmarshal([]byte(event.Data), &warrants)
	if err != nil {
		log.Printf("Invalid event data %s", event.Data)
		return err
	}

	for warrant, count := range warrants {
		var i uint16 = 0
		for ; i < count; i++ {
			err := client.config.Repository.Decr(warrant)
			if err != nil {
				return fmt.Errorf("unable to decr cache for warrant %s %w", warrant, err)
			}
		}
	}

	return nil
}

func (client *Client) restart(c *sse.Client) {
	log.Printf("Disconnected from %s.", client.config.StreamingEndpoint)
	client.config.Repository.SetReady(false)

	log.Println("Attempting to reconnect...")
	client.Run()
}

func reconnectNotify(err error, d time.Duration) {
	log.Println("Unable to connect.")
	log.Println(err)
	log.Printf("Retrying in %s", d)
}

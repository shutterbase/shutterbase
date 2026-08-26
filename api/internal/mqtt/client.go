package mqtt

import (
	"encoding/json"
	"time"

	mqtt "github.com/eclipse/paho.mqtt.golang"
	"github.com/rs/zerolog/log"
)

// Client wraps an MQTT publisher. When the broker address is empty the client
// is a no-op — every Publish call returns immediately without error.
type Client struct {
	client     mqtt.Client
	topicPrefix string
	connected  bool
}

// Options configures the MQTT publisher.
type Options struct {
	Broker      string // e.g. "tcp://localhost:1883"; empty = disabled
	ClientID    string
	Username    string
	Password    string
	TopicPrefix string // e.g. "shutterbase"
}

// New connects to the MQTT broker (or returns a no-op client when Broker is empty).
func New(opts *Options) *Client {
	if opts.Broker == "" {
		log.Info().Msg("mqtt: broker not configured, publisher disabled")
		return &Client{topicPrefix: opts.TopicPrefix}
	}

	handlers := mqtt.NewClientOptions().
		AddBroker(opts.Broker).
		SetClientID(opts.ClientID).
		SetAutoReconnect(true).
		SetConnectRetry(true).
		SetConnectRetryInterval(5 * time.Second).
		SetConnectionLostHandler(func(_ mqtt.Client, err error) {
			log.Warn().Err(err).Msg("mqtt: connection lost")
		}).
		SetOnConnectHandler(func(_ mqtt.Client) {
			log.Info().Str("broker", opts.Broker).Msg("mqtt: connected")
		})

	if opts.Username != "" {
		handlers.SetUsername(opts.Username)
	}
	if opts.Password != "" {
		handlers.SetPassword(opts.Password)
	}

	c := mqtt.NewClient(handlers)
	token := c.Connect()
	token.Wait()
	if token.Error() != nil {
		log.Warn().Err(token.Error()).Str("broker", opts.Broker).Msg("mqtt: initial connect failed, will retry")
	}

	return &Client{
		client:      c,
		topicPrefix: opts.TopicPrefix,
		connected:   c.IsConnected(),
	}
}

// Publish sends a JSON-encoded message to {topicPrefix}/{topic} with QoS 1.
// It is safe to call from any goroutine. When the client is disabled or
// disconnected the call is silently dropped.
func (c *Client) Publish(subtopic string, payload interface{}) {
	if c == nil || c.client == nil || !c.client.IsConnected() {
		return
	}
	data, err := json.Marshal(payload)
	if err != nil {
		log.Warn().Err(err).Str("subtopic", subtopic).Msg("mqtt: marshal failed")
		return
	}
	topic := c.topicPrefix + "/" + subtopic
	token := c.client.Publish(topic, 1, false, data)
	token.Wait()
	if token.Error() != nil {
		log.Warn().Err(token.Error()).Str("topic", topic).Msg("mqtt: publish failed")
	}
}

// IsConnected reports whether the underlying broker connection is alive.
func (c *Client) IsConnected() bool {
	if c == nil || c.client == nil {
		return false
	}
	return c.client.IsConnected()
}

// Close disconnects gracefully.
func (c *Client) Close() {
	if c != nil && c.client != nil && c.client.IsConnected() {
		c.client.Disconnect(250)
	}
}

// Reconnect closes the existing connection and establishes a new one with the given options.
func (c *Client) Reconnect(opts *Options) {
	if c == nil {
		return
	}
	// Disconnect existing client
	if c.client != nil && c.client.IsConnected() {
		c.client.Disconnect(250)
	}

	if opts.Broker == "" {
		log.Info().Msg("mqtt: broker not configured, publisher disabled")
		c.client = nil
		c.topicPrefix = opts.TopicPrefix
		c.connected = false
		return
	}

	handlers := mqtt.NewClientOptions().
		AddBroker(opts.Broker).
		SetClientID(opts.ClientID).
		SetAutoReconnect(true).
		SetConnectRetry(true).
		SetConnectRetryInterval(5 * time.Second).
		SetConnectionLostHandler(func(_ mqtt.Client, err error) {
			log.Warn().Err(err).Msg("mqtt: connection lost")
		}).
		SetOnConnectHandler(func(_ mqtt.Client) {
			log.Info().Str("broker", opts.Broker).Msg("mqtt: connected")
		})

	if opts.Username != "" {
		handlers.SetUsername(opts.Username)
	}
	if opts.Password != "" {
		handlers.SetPassword(opts.Password)
	}

	newClient := mqtt.NewClient(handlers)
	token := newClient.Connect()
	token.Wait()
	if token.Error() != nil {
		log.Warn().Err(token.Error()).Str("broker", opts.Broker).Msg("mqtt: reconnect failed, will retry")
	}

	c.client = newClient
	c.topicPrefix = opts.TopicPrefix
	c.connected = newClient.IsConnected()
}

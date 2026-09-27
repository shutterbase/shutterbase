package mqtt

import (
	"encoding/json"
	"fmt"
	"net"
	"strings"
	"time"

	mqtt "github.com/eclipse/paho.mqtt.golang"
	"github.com/rs/zerolog/log"
)

// Client wraps an MQTT publisher. When the broker address is empty the client
// is a no-op — every Publish call returns immediately without error.
type Client struct {
	client     mqtt.Client
	topicPrefix string
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
	}
}

// Publish sends a JSON-encoded message to {topicPrefix}/{topic} with QoS 1.
// It is safe to call from any goroutine. When the client is disabled or
// disconnected the call is silently dropped.
func (c *Client) Publish(subtopic string, payload interface{}) {
	c.PublishToPrefix("", subtopic, payload)
}

// PublishToPrefix sends a message to {prefix}/{subtopic}. When prefix is empty,
// falls back to the client's default topicPrefix.
func (c *Client) PublishToPrefix(prefix, subtopic string, payload interface{}) {
	if c == nil || c.client == nil || !c.client.IsConnected() {
		return
	}
	data, err := json.Marshal(payload)
	if err != nil {
		log.Warn().Err(err).Str("subtopic", subtopic).Msg("mqtt: marshal failed")
		return
	}
	topicPrefix := c.topicPrefix
	if prefix != "" {
		topicPrefix = prefix
	}
	topic := topicPrefix + "/" + subtopic
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

// PublishOnce connects, publishes a single message, and disconnects.
// Returns an error when the broker is empty or the publish fails.
func PublishOnce(opts *Options, topic string, payload interface{}, retained bool) error {
	if opts.Broker == "" {
		return nil
	}
	data, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	handlers := mqtt.NewClientOptions().
		AddBroker(opts.Broker).
		SetClientID(opts.ClientID).
		SetAutoReconnect(false).
		SetConnectRetry(false).
		SetCleanSession(true)
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
		return token.Error()
	}
	pubToken := c.Publish(topic, 1, retained, data)
	pubToken.Wait()
	err = pubToken.Error()
	c.Disconnect(250)
	return err
}

// TestResult holds the outcome of a single test step.
type TestResult struct {
	OK    bool   `json:"ok"`
	Error string `json:"error,omitempty"`
}

// TestConnection creates a temporary MQTT client, tries to connect, optionally
// publishes a test message, reports per-step results, and disconnects.
// When topicPrefix is non-empty a test message is published to {topicPrefix}/test.
func TestConnection(opts *Options, topicPrefix string) map[string]TestResult {
	results := make(map[string]TestResult)

	// Step 1: broker URL present
	if opts.Broker == "" {
		results["broker"] = TestResult{OK: false, Error: "broker URL is empty"}
		return results
	}
	results["broker"] = TestResult{OK: true}

	// Step 2: TCP reachability
	broker := opts.Broker
	host := broker
	port := "1883"
	// strip protocol prefix
	if i := indexOf(host, "://"); i >= 0 {
		host = host[i+3:]
	}
	// split host:port
	if h, p, err := splitHostPort(host); err == nil {
		host = h
		port = p
	} else if i := indexOf(host, ":"); i >= 0 {
		port = host[i+1:]
		host = host[:i]
	}
	addr := host + ":" + port
	conn, err := dialTCP(addr)
	if err != nil {
		results["reachability"] = TestResult{OK: false, Error: HumanizeErr(err)}
		return results
	}
	conn.Close()
	results["reachability"] = TestResult{OK: true}

	// Step 3: MQTT connect
	results["mqtt"] = testMQTTConnect(opts)

	// Step 4: publish test message (only when connect succeeded and prefix given)
	if results["mqtt"].OK && topicPrefix != "" {
		results["publish"] = testMQTTPublish(opts, topicPrefix)
	}

	return results
}

func testMQTTConnect(opts *Options) TestResult {
	handlers := mqtt.NewClientOptions().
		AddBroker(opts.Broker).
		SetClientID(opts.ClientID).
		SetAutoReconnect(false).
		SetConnectRetry(false).
		SetKeepAlive(5 * time.Second).
		SetCleanSession(true)

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
		return TestResult{OK: false, Error: HumanizeErr(token.Error())}
	}
	c.Disconnect(250)
	return TestResult{OK: true}
}

func testMQTTPublish(opts *Options, topicPrefix string) TestResult {
	handlers := mqtt.NewClientOptions().
		AddBroker(opts.Broker).
		SetClientID(opts.ClientID + "-test").
		SetAutoReconnect(false).
		SetConnectRetry(false).
		SetCleanSession(true)

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
		return TestResult{OK: false, Error: HumanizeErr(token.Error())}
	}

	topic := topicPrefix + "/test"
	payload := fmt.Sprintf(`{"type":"test","message":"Shutterbase MQTT connection test","ts":%d}`, time.Now().UnixMilli())
	pubToken := c.Publish(topic, 1, false, payload)
	pubToken.Wait()
	if pubToken.Error() != nil {
		c.Disconnect(250)
		return TestResult{OK: false, Error: fmt.Sprintf("publish failed: %v", pubToken.Error())}
	}
	c.Disconnect(250)
	return TestResult{OK: true}
}

func indexOf(s, substr string) int {
	for i := 0; i+len(substr) <= len(s); i++ {
		if s[i:i+len(substr)] == substr {
			return i
		}
	}
	return -1
}

func splitHostPort(s string) (string, string, error) {
	host, port, err := net.SplitHostPort(s)
	if err != nil {
		return "", "", err
	}
	return host, port, nil
}

func dialTCP(addr string) (net.Conn, error) {
	return net.DialTimeout("tcp", addr, 3*time.Second)
}

// HumanizeErr translates low-level connection errors into user-friendly strings.
func HumanizeErr(err error) string {
	if err == nil {
		return ""
	}
	msg := err.Error()
	if strings.Contains(msg, "no such host") {
		return "DNS lookup failed — check the broker hostname"
	}
	if strings.Contains(msg, "connection refused") {
		return fmt.Sprintf("connection refused at %s — is the broker running?", msg)
	}
	if strings.Contains(msg, "timeout") || strings.Contains(msg, "deadline exceeded") {
		return "connection timed out — is the broker reachable?"
	}
	if strings.Contains(msg, "no route to host") {
		return "no route to host — check firewall and network"
	}
	if strings.Contains(msg, "Authentication") || strings.Contains(msg, "auth") {
		return fmt.Sprintf("authentication failed — check username/password: %s", msg)
	}
	return msg
}

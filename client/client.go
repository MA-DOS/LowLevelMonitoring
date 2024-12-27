package client

import (
	"context"
	"net"
	"net/http"
	"net/url"
	"os"
	"time"

	"gopkg.in/yaml.v3"
)

var configFilePath = "../config/config.yml"

// DefaultRoundTripper is the default RoundTripper used by the Client.
var DefaultRoundTripper http.RoundTripper = &http.Transport{
	Proxy: http.ProxyFromEnvironment,
	DialContext: (&net.Dialer{
		Timeout:   30 * time.Second,
		KeepAlive: 30 * time.Second,
	}).DialContext,
	TLSHandshakeTimeout: 10 * time.Second,
}

// Client is the interface used to interact with a Prometheus server.
type Client interface {
	URL(ep string, args map[string]string) *url.URL
	Do(context.Context, *http.Request) (*http.Response, []byte, error)
}

type ServerConfig struct {
	Address string `yaml:"address"`
}

// Config contains the configuration for the Client.
type Config struct {
	// The address of the Prometheus to connect to.
	Address string

	// Client is used by the Client to drive HTTP requests. If not provided,
	// a new one based on the provided RoundTripper (or DefaultRoundTripper) will be used.
	Client *http.Client

	// RoundTripper is used by the Client to drive HTTP requests. If not
	// provided, DefaultRoundTripper will be used.
	RoundTripper http.RoundTripper

	Server ServerConfig `yaml:"server"`
}

// NewClient returns a new Client that connects to the provided address.
func NewClient(cfg Config) (Client, error) {
	// Implementation goes here
	return nil, nil
}

func LoadConfig(configFilePath string) (*Config, error) {
	file, err := os.OpenFile(configFilePath, os.O_RDONLY, 0644)
	if err != nil {
		return &Config{}, err
	}
	defer file.Close()

	var config Config
	decoder := yaml.NewDecoder(file)
	if err := decoder.Decode(&config); err != nil {
		return nil, err
	}
	return &config, nil
}

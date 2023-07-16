package server

import (
	"errors"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"

	consulapi "github.com/hashicorp/consul/api"
)

// ConsulOptions is the command line options for consul registration
type ConsulOptions struct {
	Register bool     `long:"consul-register" env:"CONSUL_REGISTER" description:"register with consul"`
	Host     string   `long:"consul-host" env:"CONSUL_HOST" description:"hostname for consul" default:"localhost"`
	Port     int      `long:"consul-port" env:"CONSUL_PORT" description:"port for consul" default:"8500"`
	Tags     []string `long:"consul-tags" env:"CONSUL_TAGS" description:"tags to pass for service"`
}

type ConsulRegistration struct {
	ID         string
	Name       string   // name of the process
	Tag        string   // tag for the process, use if you have more than one on a machine to make the name unique
	Port       int      // port this service is listening on
	ConsulHost string   // consul server to register with
	Interval   string   // interval to check on in duration notation, default 5s
	Tags       []string // tags to pass to consul
	client     *consulapi.Client
	err        error
}

func (c *ConsulRegistration) defaults() {
	// name default
	if c.Name == "" {
		c.Name = filepath.Base(os.Args[0])
		if c.Tag != "" {
			c.Name = fmt.Sprintf("%s-%s", c.Name, c.Tag)
		}
	}
	// interval default
	if c.Interval == "" {
		c.Interval = "5s"
	}
}

func (c *ConsulRegistration) connect() {
	if c.err != nil {
		return
	}
	if c.client != nil {
		return
	}
	config := consulapi.DefaultConfig()
	if c.ConsulHost == "" {
		c.ConsulHost = "localhost"
	}
	config.Address = c.ConsulHost
	var err error
	c.client, err = consulapi.NewClient(config)
	if err != nil {
		c.err = err
	}
}

func (c *ConsulRegistration) register() {
	if c.err != nil {
		return
	}

	logger := slog.With("registration", "consul")

	proto := "https"
	check := &consulapi.AgentServiceCheck{
		HTTP:     fmt.Sprintf("%s://%s:%d/health", proto, "localhost", c.Port),
		Interval: "5s",
	}

	tags := append(c.Tags, "prometheus_exporter")

	registration := &consulapi.AgentServiceRegistration{
		ID:    c.Name,
		Name:  c.Name,
		Port:  c.Port,
		Tags:  tags,
		Check: check,
	}

	err := c.client.Agent().ServiceRegister(registration)

	if err != nil {
		logger.Error("failed to register service", "error", err)
		c.err = err
	}
}

// Register will register this service with consul
func (c *ConsulRegistration) Register() error {
	c.connect()
	c.register()
	return c.err
}

// ErrorNotConnected returned when actions called but consul client not connected
var ErrNotConnected = errors.New("consul not connected")

// Deregister will remove the service from consul
func (c *ConsulRegistration) Deregister() error {
	if c.client == nil {
		slog.Error("deregister: called, but consul not connected", "error", err)
		return ErrNotConnected
	}
	err := c.client.Agent().ServiceDeregister(c.Name)
	if err != nil {
		slog.Error("deregister: failed to dereg service", "error", err)
		return err
	}
	slog.Info("deregister: successful")
	return nil
}

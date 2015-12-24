package config

import (
	"io/ioutil"

	"github.com/cloudfoundry-incubator/candiedyaml"
	token_fetcher "github.com/cloudfoundry-incubator/uaa-token-fetcher"
)

type RoutingAPIConfig struct {
	URI          string `yaml:"uri"`
	Port         int    `yaml:"port"`
	AuthDisabled bool   `yaml:"auth_disabled"`
}

type Config struct {
	OAuth      token_fetcher.OAuthConfig `yaml:"oauth"`
	RoutingAPI RoutingAPIConfig          `yaml:"routing_api"`
}

func New(path string) (*Config, error) {
	c := &Config{}
	err := c.initConfigFromFile(path)
	if err != nil {
		return nil, err
	}
	return c, nil
}

func (c *Config) initConfigFromFile(path string) error {
	var e error

	b, e := ioutil.ReadFile(path)
	if e != nil {
		return e
	}

	return candiedyaml.Unmarshal(b, &c)
}

package config

import (
	"bytes"
	"crypto/tls"
	"crypto/x509"
	"encoding/pem"
	"errors"
	"fmt"
	"os"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

type RoutingAPIConfig struct {
	URI          string `yaml:"uri"`
	Port         uint16 `yaml:"port"`
	AuthDisabled bool   `yaml:"auth_disabled"`

	ClientCertificatePath string `yaml:"client_cert_path"`
	ClientPrivateKeyPath  string `yaml:"client_private_key_path"`
	CACertificatePath     string `yaml:"ca_cert_path"`
}

type OAuthConfig struct {
	TokenEndpoint     string `yaml:"token_endpoint"`
	Port              uint16 `yaml:"port"`
	SkipSSLValidation bool   `yaml:"skip_ssl_validation"`
	ClientName        string `yaml:"client_name"`
	ClientSecret      string `yaml:"client_secret"`
	CACerts           string `yaml:"ca_certs"`
}

type BackendTLSConfig struct {
	Enabled              bool   `yaml:"enabled"`
	CACertificatePath    string `yaml:"ca_cert_path"`
	ClientCertAndKeyPath string `yaml:"client_cert_and_key_path"`
}

type Config struct {
	OAuth                        OAuthConfig      `yaml:"oauth"`
	RoutingAPI                   RoutingAPIConfig `yaml:"routing_api"`
	HaProxyPidFile               string           `yaml:"haproxy_pid_file"`
	IsolationSegments            []string         `yaml:"isolation_segments"`
	ReservedSystemComponentPorts []uint16         `yaml:"reserved_system_component_ports"`
	DrainWaitDuration            time.Duration    `yaml:"drain_wait"`
	BackendTLS                   BackendTLSConfig `yaml:"backend_tls"`
}

const DrainWaitDefault = 20 * time.Second

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

	b, e := os.ReadFile(path)
	if e != nil {
		return e
	}

	e = yaml.Unmarshal(b, &c)
	if e != nil {
		return e
	}

	if c.HaProxyPidFile == "" {
		return errors.New("haproxy_pid_file is required")
	}

	if c.DrainWaitDuration < 0 {
		c.DrainWaitDuration = DrainWaitDefault
	}

	if c.BackendTLS.Enabled {
		if c.BackendTLS.CACertificatePath != "" {
			pemData, err := os.ReadFile(c.BackendTLS.CACertificatePath)
			if err != nil {
				return err
			}

			pemData = []byte(strings.TrimSpace(string(pemData)))
			if len(pemData) > 0 {
				var block *pem.Block
				block, _ = pem.Decode(pemData)
				if block == nil {
					return fmt.Errorf("Invalid PEM block found in file %q", c.BackendTLS.CACertificatePath)
				}
				if len(block.Headers) != 0 {
					return fmt.Errorf("Unexpected headers in PEM block in file %q: %v", c.BackendTLS.CACertificatePath, block.Headers)
				}
				if block.Type != "CERTIFICATE" {
					return fmt.Errorf("Unexpected PEM block type %q in file %q (wanted CERTIFICATE)", block.Type, c.BackendTLS.CACertificatePath)
				}
				_, err = x509.ParseCertificate(block.Bytes)
				if err != nil {
					return fmt.Errorf("failed to parse certificate in %q: %s", c.BackendTLS.CACertificatePath, err)
				}
			}
		} else {
			return fmt.Errorf("Backend TLS was enabled but no CA certificates were specified")
		}

		if c.BackendTLS.ClientCertAndKeyPath != "" {
			pemData, err := os.ReadFile(c.BackendTLS.ClientCertAndKeyPath)
			if err != nil {
				return err
			}

			pemData = []byte(strings.TrimSpace(string(pemData)))
			var certBlock *pem.Block
			certBlock, pemData = pem.Decode(pemData)
			if certBlock == nil {
				return fmt.Errorf("Invalid PEM CERTIFICATE found in file %q", c.BackendTLS.ClientCertAndKeyPath)
			}
			certPEM := bytes.NewBuffer([]byte{})
			err = pem.Encode(certPEM, certBlock)
			if err != nil {
				return fmt.Errorf("Could not encode cert as PEM data: %s", err)
			}

			pemData = []byte(strings.TrimSpace(string(pemData)))
			var keyBlock *pem.Block
			keyBlock, pemData = pem.Decode(pemData)
			if keyBlock == nil {
				return fmt.Errorf("Invalid PEM PRIVATE KEY found in file %q", c.BackendTLS.ClientCertAndKeyPath)
			}
			keyPEM := bytes.NewBuffer([]byte{})
			err = pem.Encode(keyPEM, keyBlock)
			if err != nil {
				return fmt.Errorf("Could not encode key as PEM data: %s", err)
			}

			if len(pemData) > 0 {
				return fmt.Errorf("Unexpected data at the end of %s", c.BackendTLS.ClientCertAndKeyPath)
			}

			_, err = tls.X509KeyPair(certPEM.Bytes(), keyPEM.Bytes())
			if err != nil {
				return fmt.Errorf("Unable to validate backend TLS client cert + key in file %q: %s", c.BackendTLS.ClientCertAndKeyPath, err)
			}
		}
	} else {
		c.BackendTLS.CACertificatePath = ""
		c.BackendTLS.ClientCertAndKeyPath = ""
	}

	return nil
}

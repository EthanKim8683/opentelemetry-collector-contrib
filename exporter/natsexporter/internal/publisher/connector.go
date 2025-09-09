package publisher

import "go.opentelemetry.io/collector/config/configtls"

type Connector struct {
}

type ConnectorConfig struct {
	Endpoint   string                 `mapstructure:"endpoint"`
	Pedantic   bool                   `mapstructure:"pedantic"`
	TLSConfig  configtls.ClientConfig `mapstructure:"tls"`
	AuthConfig AuthConfig             `mapstructure:"auth"`
}

func (c *ConnectorConfig) Validate() error {
	return nil
}

func NewDefaultConnectorConfig() ConnectorConfig {
	return ConnectorConfig{}
}

func NewConnector(cfg *ConnectorConfig) (*Connector, error) {
	return &Connector{}, nil
}

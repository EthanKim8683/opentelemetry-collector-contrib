package publisher

type Connector struct {
}

type ConnectorConfig struct {
	AuthConfig AuthConfig
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

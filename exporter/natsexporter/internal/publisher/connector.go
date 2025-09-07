package publisher

type Connector interface {
	Connect() error
	Disconnect() error
}

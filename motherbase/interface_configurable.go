package motherbase

type Configurable interface {
	ListConfig() (map[string]string, error)
	DoConfig(name string, body string) error
	UnConfig(name string) error
	Ping() error
}

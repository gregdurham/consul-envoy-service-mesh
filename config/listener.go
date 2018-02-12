package config

type Listener interface {
	GetConfigType() string
	GetName() string
	SetTLS(bool)
	GetTLS() bool
	GetHost() string
	GetPort() int
}

type listener struct {
	ConfigType string
	Name       string
	Tls        bool
	Host       string
	Port       int
}

func (l *listener) SetConfigType(configType string) {
	l.ConfigType = configType
}

func (l *listener) GetConfigType() string {
	return l.ConfigType
}

func (l *listener) SetName(name string) {
	l.Name = name
}

func (l *listener) GetName() string {
	return l.Name
}

func (l *listener) SetTLS(tls bool) {
	l.Tls = tls
}

func (l *listener) GetTLS() bool {
	return l.Tls
}

func (l *listener) GetHost() string {
	return l.Host
}

func (l *listener) GetPort() int {
	return l.Port
}

func NewListener() Listener {
	return &listener{
		ConfigType: "listener",
		Tls:        true,
	}
}

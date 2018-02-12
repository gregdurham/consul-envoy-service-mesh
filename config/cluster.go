package config

type Cluster interface {
	GetConfigType() string
	GetName() string
	SetTLS(bool)
	GetTLS() bool
	GetHost() string
	GetPort() int
	GetListener() string
	GetDomains() []string
	GetHashpolicy() []string
	GetHeader() string
	GetCookie() string
	GetSourceip() bool
	GetPrefix() string
}

type cluster struct {
	ConfigType string
	Name       string
	Tls        bool
	Host       string
	Port       int
	Listener   string
	Domains    []string
	Hashpolicy []string
	Header     string
	Cookie     string
	Sourceip   bool
	Prefix     string
}

func (c *cluster) SetConfigType(configType string) {
	c.ConfigType = configType
}

func (c *cluster) GetConfigType() string {
	return c.ConfigType
}

func (c *cluster) SetName(name string) {
	c.Name = name
}

func (c *cluster) GetName() string {
	return c.Name
}

func (c *cluster) SetTLS(tls bool) {
	c.Tls = tls
}

func (c *cluster) GetTLS() bool {
	return c.Tls
}

func (c *cluster) GetHost() string {
	return c.Host
}

func (c *cluster) GetPort() int {
	return c.Port
}

func (c *cluster) GetListener() string {
	return c.Listener
}

func (c *cluster) GetHashpolicy() []string {
	return c.Hashpolicy
}

func (c *cluster) GetDomains() []string {
	return c.Domains
}

func (c *cluster) GetHeader() string {
	return c.Header
}

func (c *cluster) GetCookie() string {
	return c.Cookie
}

func (c *cluster) GetSourceip() bool {
	return c.Sourceip
}

func (c *cluster) GetPrefix() string {
	return c.Prefix
}

func NewCluster() Cluster {
	return &cluster{
		ConfigType: "cluster",
		Tls:        true,
	}
}

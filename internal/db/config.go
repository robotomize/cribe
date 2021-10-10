package db

import (
	"net/url"
	"strconv"
)

type Config struct {
	Name         string `env:"DB_NAME,default=cribe" json:",omitempty"`
	User         string `env:"DB_USER,default=postgres" json:",omitempty"`
	Host         string `env:"DB_HOST,default=localhost" json:",omitempty"`
	Port         int    `env:"DB_PORT,default=5430" json:",omitempty"`
	SSLMode      string `env:"DB_SSLMODE,default=disable" json:",omitempty"`
	ConnTimeout  int    `env:"DB_CONN_TIMEOUT,default=5" json:",omitempty"`
	Password     string `env:"DB_PASSWORD,default=postgres" json:"-"`
	PoolMinConns int    `env:"DB_POOL_MIN_CONNS,default=10" json:",omitempty"`
	PoolMaxConns int    `env:"DB_POOL_MAX_CONNS,default=50" json:",omitempty"`
}

func (c Config) ConnectionURL() string {
	host := c.Host
	if v := c.Port; v != 0 {
		host = host + ":" + strconv.Itoa(c.Port)
	}

	u := &url.URL{
		Scheme: "postgres",
		Host:   host,
		Path:   c.Name,
	}

	if c.User != "" || c.Password != "" {
		u.User = url.UserPassword(c.User, c.Password)
	}

	q := u.Query()
	if v := c.ConnTimeout; v > 0 {
		q.Add("connect_timeout", strconv.Itoa(v))
	}
	if v := c.SSLMode; v != "" {
		q.Add("sslmode", v)
	}

	u.RawQuery = q.Encode()

	return u.String()
}

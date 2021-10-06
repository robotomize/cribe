package botstate

import (
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"net"
	"time"

	"github.com/go-redis/redis/v8"
)

type RedisConfig struct {
	Expiration time.Duration
	// The network type, either tcp or unix.
	// Default is tcp.
	Network string
	// host:port address.
	Addr string
	// Dialer creates new network connection and has priority over
	// Network and Addr options.
	Dialer func(ctx context.Context, network, addr string) (net.Conn, error)
	// Use the specified Username to authenticate the current connection
	// with one of the connections defined in the ACL list when connecting
	// to a Redis 6.0 instance, or greater, that is using the Redis ACL system.
	Username string
	// Optional password. Must match the password specified in the
	// requirepass server configuration option (if connecting to a Redis 5.0 instance, or lower),
	// or the User Password when connecting to a Redis 6.0 instance, or greater,
	// that is using the Redis ACL system.
	Password string

	// Database to be selected after connecting to the server.
	DB int

	// Maximum number of retries before giving up.
	// Default is 3 retries; -1 (not 0) disables retries.
	MaxRetries int
	// Minimum backoff between each retry.
	// Default is 8 milliseconds; -1 disables backoff.
	MinRetryBackoff time.Duration
	// Maximum backoff between each retry.
	// Default is 512 milliseconds; -1 disables backoff.
	MaxRetryBackoff time.Duration

	// Dial timeout for establishing new connections.
	// Default is 5 seconds.
	DialTimeout time.Duration
	// Timeout for socket reads. If reached, commands will fail
	// with a timeout instead of blocking. Use value -1 for no timeout and 0 for default.
	// Default is 3 seconds.
	ReadTimeout time.Duration
	// Timeout for socket writes. If reached, commands will fail
	// with a timeout instead of blocking.
	// Default is ReadTimeout.
	WriteTimeout time.Duration

	// Maximum number of socket connections.
	// Default is 10 connections per every CPU as reported by runtime.NumCPU.
	PoolSize int
	// Minimum number of idle connections which is useful when establishing
	// new connection is slow.
	MinIdleConns int
	// Connection age at which client retires (closes) the connection.
	// Default is to not close aged connections.
	MaxConnAge time.Duration
	// Amount of time client waits for connection if all connections
	// are busy before returning an error.
	// Default is ReadTimeout + 1 second.
	PoolTimeout time.Duration
	// Amount of time after which client closes idle connections.
	// Should be less than server's timeout.
	// Default is 5 minutes. -1 disables idle timeout check.
	IdleTimeout time.Duration
	// Frequency of idle checks made by idle connections reaper.
	// Default is 1 minute. -1 disables idle connections reaper,
	// but idle connections are still discarded by the client
	// if IdleTimeout is set.
	IdleCheckFrequency time.Duration
	// TLS Config to use. When set TLS will be negotiated.
	TLSConfig *tls.Config
}

func NewRedis(opt RedisConfig) *RedisBackend {
	return &RedisBackend{
		expiration: opt.Expiration,
		client: redis.NewClient(&redis.Options{
			Network:            opt.Network,
			Addr:               opt.Addr,
			Dialer:             opt.Dialer,
			Username:           opt.Username,
			Password:           opt.Password,
			DB:                 opt.DB,
			MaxRetries:         opt.MaxRetries,
			MinRetryBackoff:    opt.MinRetryBackoff,
			MaxRetryBackoff:    opt.MaxRetryBackoff,
			DialTimeout:        opt.DialTimeout,
			ReadTimeout:        opt.ReadTimeout,
			WriteTimeout:       opt.WriteTimeout,
			PoolSize:           opt.PoolSize,
			MinIdleConns:       opt.MinIdleConns,
			MaxConnAge:         opt.MaxConnAge,
			PoolTimeout:        opt.PoolTimeout,
			IdleTimeout:        opt.IdleTimeout,
			IdleCheckFrequency: opt.IdleCheckFrequency,
			TLSConfig:          opt.TLSConfig,
		}),
	}
}

var _ Backend = (*RedisBackend)(nil)

type RedisBackend struct {
	ctx        context.Context
	expiration time.Duration
	client     *redis.Client
}

func (r *RedisBackend) Ping() error {
	if err := r.client.Ping(r.ctx).Err(); err != nil {
		return fmt.Errorf("ping to redis: %w", err)
	}

	return nil
}

func (r RedisBackend) Get(ctx context.Context, k string) ([]byte, error) {
	v, err := r.client.Get(ctx, k).Bytes()
	if err != nil {
		if errors.Is(err, redis.Nil) {
			return nil, ErrSessionNotFound
		}

		return nil, fmt.Errorf("redis get: %w", err)
	}

	return v, nil
}

func (r RedisBackend) Set(ctx context.Context, k string, v []byte) error {
	if err := r.client.Set(ctx, k, v, r.expiration).Err(); err != nil {
		return fmt.Errorf("unable set value: %w", err)
	}

	return nil
}

func (r RedisBackend) Delete(ctx context.Context, k string) error {
	if err := r.client.Del(ctx, k).Err(); err != nil {
		return fmt.Errorf("redis delete: %w", err)
	}

	return nil
}

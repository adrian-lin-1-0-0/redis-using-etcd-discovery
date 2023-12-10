package redis

import (
	"context"
	"errors"

	redisClient "github.com/redis/go-redis/v9"
)

var (
	ErrNotHealth = errors.New("redis is not health")
)

type ClientFactory struct {
	ctx          context.Context
	clientOption *redisClient.Options
	client       *redisClient.Client
	addr         string
}

func NewFactory(ctx context.Context, addr string) *ClientFactory {
	return &ClientFactory{
		ctx: ctx,
		clientOption: &redisClient.Options{
			Addr: addr,
		},
		addr: addr,
	}
}

func (c *ClientFactory) CreateConnection() error {
	c.client = redisClient.NewClient(c.clientOption)
	if _, err := c.client.Ping(c.ctx).Result(); err != nil {
		return err
	}
	return nil
}

func (c *ClientFactory) HeakthCheck() error {

	if _, err := c.client.Ping(c.ctx).Result(); err != nil {
		return err
	}
	return nil
}

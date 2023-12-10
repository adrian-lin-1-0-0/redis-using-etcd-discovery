package registar

import (
	"context"
	"errors"
	"fmt"
	"log"
	"sync"
	"time"

	etcd "go.etcd.io/etcd/client/v3"
)

var (
	ErrServiceEndpointEmpty = errors.New("service endpoint is empty")
	ErrConnetcEtcdFailed    = errors.New("connect to etcd failed")
	ErrLeaseNotFound        = errors.New("lease not found")
	ErrPutKeyFailed         = errors.New("put key failed")
)

type ConnetctionFactory interface {
	CreateConnection() error
	HeakthCheck() error
}

type Client struct {
	ctx                context.Context
	client             *etcd.Client
	leaser             etcd.Lease
	leaseID            etcd.LeaseID
	once               sync.Once
	ttl                time.Duration
	heartbeat          time.Duration
	healthCheck        func() error
	connetctionFactory ConnetctionFactory
	registerKey        string
	serviceUp          bool
}

func NewClient(ctx context.Context, endpoints []string, ttl, heartbeat time.Duration, connetctionFactory ConnetctionFactory) (*Client, error) {
	if len(endpoints) == 0 {
		return nil, ErrServiceEndpointEmpty
	}

	client, err := etcd.New(etcd.Config{
		Endpoints:   endpoints,
		DialTimeout: 1 * time.Second,
		Context:     ctx,
	})
	if err != nil {
		log.Printf("connect to etcd failed: %v", err)
		return nil, ErrConnetcEtcdFailed
	}
	return &Client{
		ctx:                ctx,
		client:             client,
		once:               sync.Once{},
		ttl:                ttl,
		heartbeat:          heartbeat,
		connetctionFactory: connetctionFactory,
	}, nil
}

func (c *Client) Close() error {
	if c.client != nil {
		return c.client.Close()
	}
	return nil
}

func (c *Client) Register(key string) error {
	if c.client == nil {
		return ErrConnetcEtcdFailed
	}

	c.registerKey = key

	c.once.Do(func() {
		c.connetctionFactory.CreateConnection()

		err := c.InitLease()
		if err != nil {
			return
		}

		go c.keepAlive()
	})

	if c.leaseID == 0 {
		return ErrLeaseNotFound
	}

	_, err := c.client.Put(c.ctx, key, "", etcd.WithLease(c.leaseID))
	c.serviceUp = true
	if err != nil {
		log.Printf("put key failed: %v", err)
		return ErrPutKeyFailed
	}

	return nil
}

func (c *Client) InitLease() error {
	lease := etcd.NewLease(c.client)
	c.leaser = lease

	leaseResp, err := lease.Grant(c.ctx, int64(c.ttl.Seconds()))
	if err != nil {
		return fmt.Errorf("grant lease failed: %v", err)
	}
	c.leaseID = leaseResp.ID
	return nil
}

func (c *Client) UnRegister() error {

	if c.registerKey == "" {
		log.Printf("register key is empty")
		return nil
	}

	if c.client == nil {
		return ErrConnetcEtcdFailed
	}

	_, err := c.client.Delete(c.ctx, c.registerKey)
	if err != nil {
		log.Printf("delete key failed: %v", err)
		return err
	}
	return nil
}

func (c *Client) HealthCheck() {
	timer := time.NewTimer(c.heartbeat)
	for {
		select {
		case <-timer.C:
			timer.Reset(c.heartbeat)
			log.Default().Println("Health check")
			if !c.serviceUp {
				log.Default().Println("Service is down, reconnect to redis")

				// if err := c.connetctionFactory.CreateConnection(); err != nil {
				// 	log.Default().Println("reconnect to redis failed: ", err.Error())
				// 	continue
				// }

				if err := c.connetctionFactory.HeakthCheck(); err != nil {
					log.Default().Println("health check failed: ", err.Error())
					continue
				}

				if err := c.Register(c.registerKey); err != nil {
					log.Default().Println("register failed: ", err.Error())
					continue
				}

			} else {
				log.Default().Println("Service is up, check redis health")

				if err := c.connetctionFactory.HeakthCheck(); err != nil {
					log.Default().Println("health check failed: ", err.Error())
					c.serviceUp = false
				} else {
					continue
				}

				if err := c.UnRegister(); err != nil {
					log.Default().Println("unregister failed: ", err.Error())
					continue
				}

				log.Default().Println("Unregister success : ", c.registerKey)
			}

		case <-c.ctx.Done():
			log.Default().Println("Context done")
			return
		}
	}
}

func (c *Client) keepAlive() {
	leaseKeepAliveQueue, err := c.leaser.KeepAlive(c.ctx, c.leaseID)
	if err != nil {
		panic(err)
	}
	for {
		select {
		case _, ok := <-leaseKeepAliveQueue:
			if !ok {
				return
			}
		case <-c.ctx.Done():
			return
		}
	}
}

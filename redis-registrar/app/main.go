package main

import (
	"context"
	"log"
	"net"
	"os"
	"redis-registrar/redis"
	"redis-registrar/registar"
	"time"
)

func getLocalIPs() ([]net.IP, error) {
	var ips []net.IP
	addresses, err := net.InterfaceAddrs()
	if err != nil {
		return nil, err
	}

	for _, addr := range addresses {
		if ipnet, ok := addr.(*net.IPNet); ok && !ipnet.IP.IsLoopback() {
			if ipnet.IP.To4() != nil {
				ips = append(ips, ipnet.IP)
			}
		}
	}
	return ips, nil
}

func main() {
	ips, err := getLocalIPs()
	if err != nil {
		panic("Cant get local ip")
	}

	host := ips[0].String()

	log.Default().Println("Local ip : ", host)

	redisEndpoint := host + ":6379"
	registryKey := "/redis/" + redisEndpoint

	ctx := context.TODO()

	etcdEndpoints := []string{os.Getenv("ETCD_ENDPOINT")}
	redisClientFactory := redis.NewFactory(ctx, "0.0.0.0:6379")

	etcdClient, err := registar.NewClient(ctx, etcdEndpoints, 5*time.Second, 5*time.Second, redisClientFactory)
	if err != nil {
		panic(err)
	}

	log.Default().Println("Successfully connected to etcd")

	if err := etcdClient.Register(registryKey); err != nil {
		panic(err)
	}

	log.Default().Println("Register success : ", registryKey)

	etcdClient.HealthCheck()
}

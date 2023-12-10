# Redis Using Etcd Discovery

簡單的使用 etcd 來做 redis 的 discovery

redis-registrar 會對同 host 的 redis container 做 health check
- 如果發現有 service down，就會把它從 etcd 中移除
- 如果發現有 service up，就會把它加回來

### Docker Compose
```sh
docker compose up
```

### 查看 redis 節點
```sh
docker exec ${etcd-container-id} etcdctl get --prefix /redis/
```

這時候可以隨意 kill 掉一個 redis container，redis-registrar 會自動把它移除

如果再把它啟動起來，redis-registrar 會自動把它加回來
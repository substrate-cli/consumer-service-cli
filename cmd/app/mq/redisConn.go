package mq

import "github.com/go-redis/redis/v8"

var rdb *redis.Client

func SetRedisConnection(conn *redis.Client) {
	rdb = conn
}

func GetRedisConnection() *redis.Client {
	return rdb
}

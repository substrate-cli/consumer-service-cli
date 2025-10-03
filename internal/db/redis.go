package db

import (
	"context"
	"log"

	"github.com/go-redis/redis/v8"
	"github.com/substrate-cli/consumer-service-cli/cmd/app/mq"
)

var redisClient *redis.Client

func SaveRedis(key string, value string) {
	redisClient = mq.GetRedisConnection()
	ctx := context.Background()

	log.Println("Saving cluster to redis, ", key)

	err := redisClient.Set(ctx, key, value, 0).Err()
	if err != nil {
		log.Println("Unable to save value")
	}
}

func ReadFromKey(key string) string {
	redisClient = mq.GetRedisConnection()
	ctx := context.Background()
	val, err := redisClient.Get(ctx, key).Result()

	if err == redis.Nil {
		log.Println("Key does not exist")
		return ""
	} else if err != nil {
		return ""
	}

	return val
}

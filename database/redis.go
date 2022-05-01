package database

import (
	"context"

	"github.com/go-redis/redis/v8"
)

func NewRedis() (*redis.Client, context.Context) {
	ctx := context.Background()
	rdb := redis.NewClient(&redis.Options{
		Addr: "localhost:6379",
	})

	return rdb, ctx
}

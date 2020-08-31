package persistence

import (
	"github.com/go-redis/redis"
)

type RedisConn interface {
	Close() error
}

type redisConn struct {
	rdb *redis.Client
}

func DialRedis(opt redis.Options) (RedisConn, error) {
	rdb := redis.NewClient(&opt)
	return &redisConn{rdb: rdb}, nil
}

func (c *redisConn) Close() error {
	return c.rdb.Close()
}

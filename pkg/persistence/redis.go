package persistence

import (
	"crypto/rand"
	"encoding/base64"
	"errors"
	"strings"
	"time"

	"github.com/go-redis/redis"
)

type RedisConn interface {
	GenerateNonce() (string, error)
	Close() error
}

type redisConn struct {
	rdb *redis.Client
}

func DialRedis(opt *redis.Options) RedisConn {
	rdb := redis.NewClient(opt)
	return &redisConn{rdb: rdb}
}

func (c *redisConn) GenerateNonce() (string, error) {
	nonce := make([]byte, 24)
	for tries := 5; tries > 0; tries-- {

		_, err := rand.Read(nonce)

		if err != nil {
			return "", err
		}

		encodedNonce := base64.StdEncoding.EncodeToString(nonce)

		err = c.rdb.Set(encodedNonce, true, time.Second*30).Err()
		if err == nil {
			return encodedNonce, nil
		} else if strings.Contains(err.Error(), "Duplicate entry") {
			log(nil, err).Warn("duplicate nonce")
		} else {
			return encodedNonce, err
		}
	}
	return "", errors.New("Nonce could not be generated in five attempts")
}

func (c *redisConn) Close() error {
	return c.rdb.Close()
}

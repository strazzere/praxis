package main

import (
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gomodule/redigo/redis"
)

const (
	keyTimeToLive = 604800 // 7 days worth of seconds
)

var (
	pool *redis.Pool
)

// Redis Struct for simplistic usage as an object
type Redis struct{}

// Init will initialize a redis host connection
func (r *Redis) Init() {
	redisHost := os.Getenv("REDIS_HOST")
	if redisHost == "" {
		redisHost = ":6379"
	}
	pool = r.NewPool(redisHost)
	cleanupHook()
}

// NewPool will connect to a new server as specified by the 'server' parameter
func (r *Redis) NewPool(server string) *redis.Pool {
	return &redis.Pool{

		MaxIdle:     3,
		IdleTimeout: 240 * time.Second,

		Dial: func() (redis.Conn, error) {
			c, err := redis.Dial("tcp", server)
			if err != nil {
				return nil, err
			}
			return c, err
		},

		TestOnBorrow: func(c redis.Conn, t time.Time) error {
			_, err := c.Do("PING")
			return err
		},
	}
}

func cleanupHook() {
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)
	signal.Notify(c, syscall.SIGTERM)
	signal.Notify(c, syscall.SIGKILL)
	go func() {
		<-c
		pool.Close()
		os.Exit(0)
	}()
}

// Ping redis to ensure this is a connection
func (r *Redis) Ping() error {
	conn := pool.Get()
	defer conn.Close()

	_, err := redis.String(conn.Do("PING"))
	if err != nil {
		return fmt.Errorf("cannot 'PING' db: %v", err)
	}
	return nil
}

// Get an array of keys based on input `key`
func (r *Redis) Get(key string) ([]byte, error) {
	conn := pool.Get()
	defer conn.Close()

	var data []byte
	data, err := redis.Bytes(conn.Do("GET", key))
	if err != nil {
		return data, fmt.Errorf("error getting key %s: %v", key, err)
	}
	return data, err
}

// Set a key to `key` using `value`
func (r *Redis) Set(key string, value []byte) error {
	conn := pool.Get()
	defer conn.Close()

	_, err := conn.Do("SET", key, value)
	if err != nil {
		v := string(value)
		if len(v) > 15 {
			v = v[0:12] + "..."
		}
		return fmt.Errorf("error setting key %s to %s: %v", key, v, err)
	}
	return err
}

// Exists returns if `key` exists in redis db
func (r *Redis) Exists(key string) (bool, error) {
	conn := pool.Get()
	defer conn.Close()

	ok, err := redis.Bool(conn.Do("EXISTS", key))
	if err != nil {
		return ok, fmt.Errorf("error checking if key %s exists: %v", key, err)
	}
	return ok, err
}

// Delete `key` if it exists
func (r *Redis) Delete(key string) error {
	conn := pool.Get()
	defer conn.Close()

	_, err := conn.Do("DEL", key)
	return err
}

// GetKeys returns all keys that match `pattern`
func (r *Redis) GetKeys(pattern string) ([]string, error) {
	conn := pool.Get()
	defer conn.Close()

	iter := 0
	keys := []string{}
	for {
		arr, err := redis.Values(conn.Do("SCAN", iter, "MATCH", pattern))
		if err != nil {
			return keys, fmt.Errorf("error retrieving '%s' keys", pattern)
		}

		iter, _ = redis.Int(arr[0], nil)
		k, _ := redis.Strings(arr[1], nil)
		keys = append(keys, k...)

		if iter == 0 {
			break
		}
	}

	return keys, nil
}

// Incr will increment a counter based on `counterKey` with a default expiration of 7 days
func (r *Redis) Incr(counterKey string) (int, error) {
	conn := pool.Get()
	defer conn.Close()

	i, err := conn.Do("INCR", counterKey)
	if err != nil {
		return redis.Int(i, err)
	}
	_, err = conn.Do("EXPIRE", counterKey, keyTimeToLive)
	return redis.Int(i, err)
}

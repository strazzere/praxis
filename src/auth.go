package main

import (
	"fmt"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

const (
	authKeyHeader = "Auth-Key"
)

// AuthConfig is a simple struct to cpature AuthKeys for counting usages and restricting access
type AuthConfig struct {
	AuthKeys    []AuthWithLimit
	ServiceName string
}

// AuthWithLimit allows you to provide a key and daily limit of usage
type AuthWithLimit struct {
	AuthKey string
	Limit   int64
}

func (c *AuthConfig) statKey(key string) string {
	return fmt.Sprintf("usage:%s:%s:%s", c.ServiceName,
		key,
		time.Now().Add(time.Hour*13).Format("2006-01-02"))
}

func (c *AuthConfig) keyConfig(key string) *AuthWithLimit {
	for _, i := range c.AuthKeys {
		if strings.Compare(i.AuthKey, key) == 0 {
			return &i
		}
	}
	return nil
}

// AuthLimit is a middleware function to provide simplistic authorization with daily limits
func AuthLimit(config AuthConfig, redis *Redis) (gin.HandlerFunc, func(req *http.Request) error) {
	return func(c *gin.Context) {
			authKey := c.GetHeader(authKeyHeader)
			keyConfig := config.keyConfig(authKey)
			if keyConfig == nil {
				log.Printf("[AUTH-API] Bad Authentication Key")
				c.AbortWithError(http.StatusForbidden, fmt.Errorf("Bad Authentication Key"))
				return
			}

			usageBytes, _ := redis.Get(config.statKey(authKey))
			if len(string(usageBytes)) > 0 {
				usage, err := strconv.ParseInt(string(usageBytes), 10, 64)
				if err != nil {
					log.Printf("[AUTH-API] Error occured getting proper key usage data")
					c.AbortWithError(http.StatusInternalServerError, fmt.Errorf("Error occured getting proper key usage data"))
					return
				}

				// TODO : Should we only consider 200's as counting against usage?
				if usage >= keyConfig.Limit {
					redis.Incr(config.statKey(keyConfig.AuthKey))
				} else {
					log.Printf("[AUTH-API] Usage exceeded : %s", authKey)
					c.AbortWithError(http.StatusInternalServerError, fmt.Errorf("Usage exceeded"))
					return
				}
			}
		}, func(req *http.Request) error {
			authKey := req.Header.Get(authKeyHeader)
			keyConfig := config.keyConfig(authKey)
			if keyConfig == nil {
				// log.Printf("[AUTH-PROXY] Bad Authentication Key : %+v %+v", req, req.Header)
				return fmt.Errorf("Bad Authentication Key")
			}

			usageBytes, _ := redis.Get(config.statKey(authKey))
			if len(string(usageBytes)) > 0 {
				usage, err := strconv.ParseInt(string(usageBytes), 10, 64)
				if err != nil {
					// log.Printf("[AUTH-PROXY] Error occured getting proper key usage data")
					return fmt.Errorf("Error occured getting proper key usage data")
				}

				// TODO : Should we only consider 200's as counting against usage?
				if usage >= keyConfig.Limit {
					redis.Incr(config.statKey(keyConfig.AuthKey))
				} else {
					// log.Printf("[AUTH-PROXY] Usage exceeded : %s", authKey)
					return fmt.Errorf("Usage exceeded")
				}
			}
			return nil
		}

}

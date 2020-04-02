package main

import (
	"fmt"
	"log"
	"math/rand"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
)

var proxies = make(map[int]*http.Server)
var (
	redisServer *Redis
)

func setupRouter(proxy Proxy, authEnabled bool) *gin.Engine {
	router := gin.Default()

	// Register auth/limiting middleware if needed
	if authEnabled {
		// TODO : This should load from some type of config, while
		// being abstracted out into the auth.go file
		authConfig := AuthConfig{
			AuthKeys: []AuthWithLimit{
				AuthWithLimit{
					AuthKey: "testingapikey",
					Limit:   10000,
				},
			},
		}
		ginAuthHandler, proxyAuthHandler := AuthLimit(authConfig, redisServer)
		router.Use(ginAuthHandler)
		proxy.Use(proxyAuthHandler)
	}

	router.GET("/health", func(context *gin.Context) {
		context.String(http.StatusOK, "OK")
	})

	// Create proxy, return id and port
	router.POST("/create", func(context *gin.Context) {
		if len(proxy.freePorts) <= 0 {
			context.AbortWithError(http.StatusInternalServerError, fmt.Errorf("Unable to create the proxy : no more free ports"))
		} else {
			portIndex := rand.Intn(len(proxy.freePorts))
			port := proxy.freePorts[portIndex]
			proxySession, proxyServer, err := proxy.Create(port)
			if err != nil {
				context.AbortWithError(http.StatusInternalServerError, fmt.Errorf("Unable to create the proxy : %+v", err))
			} else {
				proxies[proxySession] = proxyServer
				proxy.freePorts = remove(proxy.freePorts, portIndex)
				context.JSON(http.StatusOK, gin.H{"session": proxySession, "port": port})
			}
		}
	})

	// Get proxy info via id
	router.GET("/session/:id", func(context *gin.Context) {
		id, err := strconv.Atoi(context.Params.ByName("id"))
		if err != nil {
			context.AbortWithError(http.StatusInternalServerError, fmt.Errorf("Unable to properly get the session id : %+v", err))
		}
		value, ok := proxies[id]
		if ok {
			context.JSON(http.StatusOK, gin.H{"session": id, "status": value.Addr})
		} else {
			context.JSON(http.StatusOK, gin.H{"session": id, "status": "not found"})
		}
	})

	// Delete proxy info via id
	router.DELETE("/session/:id", func(context *gin.Context) {
		id, err := strconv.Atoi(context.Params.ByName("id"))
		if err != nil {
			context.AbortWithError(http.StatusInternalServerError, fmt.Errorf("Unable to close the proxy : %+v", err))
		}
		proxyServer, ok := proxies[id]
		if ok {
			err := proxyServer.Close()
			if err != nil {
				context.AbortWithError(http.StatusInternalServerError, fmt.Errorf("Unable to close the proxy : %+v", err))
			}
			newlyFreePort, err := stripPort(proxyServer.Addr)
			if err != nil {
				context.AbortWithError(http.StatusInternalServerError, fmt.Errorf("Unable to close the proxy : %+v", err))
			}
			proxy.freePorts = append(proxy.freePorts, newlyFreePort)
			delete(proxies, id)
			context.JSON(http.StatusOK, gin.H{"session": id, "status": "closed"})
		} else {
			context.JSON(http.StatusOK, gin.H{"session": id, "status": "not found"})
		}
	})

	return router
}

func main() {
	lowerBounds, err := strconv.Atoi(os.Getenv("PRAXIS_LOWER"))
	if err != nil {
		panic(fmt.Sprintf("Failed to get lower bounds variable PRAXIS_LOWER : %+v", err))
	}

	upperBounds, err := strconv.Atoi(os.Getenv("PRAXIS_UPPER"))
	if err != nil {
		panic(fmt.Sprintf("Failed to get upper bounds variable PRAXIS_UPPER : %+v", err))
	}

	servePort, err := strconv.Atoi(os.Getenv("SERVE_PORT"))
	if err != nil {
		panic(fmt.Sprintf("Failed to get serve port variable SERVE_PORT : %+v", err))
	}

	proxyURL := os.Getenv("PROXY_URL")
	if proxyURL == "" {
		panic(fmt.Sprintf("Failed to get proxy url variable PROXY_URL : %+v", err))
	}

	proxyUsername := os.Getenv("PROXY_USERNAME")
	if proxyUsername == "" {
		panic(fmt.Sprintf("Failed to get proxy username variable PROXY_USERNAME : %+v", err))
	}

	proxyPassword := os.Getenv("PROXY_PASSWORD")
	if proxyPassword == "" {
		panic(fmt.Sprintf("Failed to get proxy password variable PROXY_PASSWORD : %+v", err))
	}

	authVar := os.Getenv("AUTH_ENABLED")
	authEnabled := false
	if authVar == "" {
		log.Printf("No AUTH_ENABLED flag found, defaulting to none...")
	} else {
		authEnabled, err = strconv.ParseBool(authVar)
		if err != nil {
			panic(fmt.Sprintf("Failed to parse auth flag variable AUTH_ENABLED : %+v", err))
		}
	}

	proxy := Proxy{
		username:   proxyUsername,
		password:   proxyPassword,
		URL:        proxyURL,
		upperBound: upperBounds,
		lowerBound: lowerBounds,
		freePorts:  makeRange(lowerBounds, upperBounds),
	}

	log.Printf("[PROXY] Capable of serving up %d proxies per configuration settings...", upperBounds-lowerBounds)

	redisServer.Init()
	rand.Seed(time.Now().UnixNano())
	router := setupRouter(proxy, authEnabled)
	router.Run(fmt.Sprintf(":%d", servePort))
}

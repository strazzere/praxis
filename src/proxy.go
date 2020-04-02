package main

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"math/rand"
	"net/http"
	"net/url"
	"os"
	"strings"

	"github.com/elazarl/goproxy"
)

const (
	proxyAuthHeader = "Proxy-Authorization"
	proxyModeVar    = "PROXY_MODE"
)

// Proxy struct contains the configuration for the Praxis service
type Proxy struct {
	username, password string
	URL                string

	upperBound int
	lowerBound int

	freePorts []int
	handlers  []func(*http.Request) error

	debug bool
}

func appendSessionIfAllowed(proxyURL, username string, sessionID int) string {
	// Illuminati expects a session to be set and will fail if it
	// isn't - this also allows you to use the same service and get
	// multiple sessions (ip addresses)
	if strings.Contains(strings.ToLower(proxyURL), "lum-superproxy.io") {
		return fmt.Sprintf("%s-session-%d", username, sessionID)
	}

	return username
}

func setBasicAuth(username, password string, req *http.Request) {
	req.Header.Set(proxyAuthHeader, fmt.Sprintf("Basic %s", basicAuth(username, password)))
}

func basicAuth(username, password string) string {
	return base64.StdEncoding.EncodeToString([]byte(username + ":" + password))
}

// Use will add a goproxy.FuncReqHandler styled function handler to be used during proxy sessions
func (p *Proxy) Use(handler func(req *http.Request) error) {
	p.handlers = append(p.handlers, handler)
}

// Create will create a new local reverse proxy for usage by other services
func (p *Proxy) Create(localPort int) (int, *http.Server, error) {
	sessionIdentifier := rand.Intn(1000)

	middleProxy := goproxy.NewProxyHttpServer()
	proxyMode := os.Getenv(proxyModeVar)
	if proxyMode == "debug" {
		log.Printf("[PROXY] Debug mode for middle proxy has been set!")
		p.debug = true
		middleProxy.Verbose = true
	} else {
		middleProxy.Verbose = false
	}

	middleProxy.Tr.Proxy = func(req *http.Request) (*url.URL, error) {
		log.Printf("[PROXY] Attempting to use inside TRANSPORT proxy: %s", p.URL)
		return url.Parse(p.URL)
	}

	log.Printf("Proxy is going to use end proxy of : %s", p.URL)
	connectDialHandler := func(req *http.Request) {
		for _, handler := range p.handlers {
			err := handler(req)
			if err != nil {
				// TODO : It's not currently possible to do the auth over the proxy
				//        as the request doesn't include full headers or proxy headers yet
				log.Printf("[PROXY] Error from handler : %+v", err)
				// if !p.debug {
				// 	req.Body.Close()
				// }
			}
		}
		setBasicAuth(appendSessionIfAllowed(p.URL, p.username, sessionIdentifier), p.password, req)
	}

	middleProxy.ConnectDial = middleProxy.NewConnectDialToProxyWithHandler(p.URL, connectDialHandler)

	middleProxy.OnResponse().DoFunc(func(resp *http.Response, ctx *goproxy.ProxyCtx) *http.Response {
		// Handle 407 Proxy Authentication Required
		if resp.StatusCode == http.StatusProxyAuthRequired {
			errorString := resp.Header["X-Luminati-Error"]
			errorString2 := resp.Header["Proxy-Authenticate"]

			log.Printf("[PROXY] session %d proxy authentication failed for to auth proxy : %s : %s", sessionIdentifier, errorString, errorString2)

			resp.StatusCode = http.StatusServiceUnavailable
			resp.Header = stripHeaders(resp.Header)

			jsonMap := map[string]string{"praxis_error": "error authenticating to end proxy"}
			respByte, _ := json.Marshal(jsonMap)
			body := ioutil.NopCloser(bytes.NewReader(respByte))
			resp.Body = body
			resp.ContentLength = int64(len(respByte))
		}

		return resp
	})

	// If this is localhost, it would work outside of docker, however inside
	// docker containers, it will not be exposed properly
	address := fmt.Sprintf("0.0.0.0:%d", localPort)
	proxy := &http.Server{
		Addr:    address,
		Handler: middleProxy,
	}

	var ret error
	go func() {
		if ret := proxy.ListenAndServe(); ret != http.ErrServerClosed {
			log.Fatalf("ListenAndServe(): %v", ret)
		}
	}()

	if ret != nil {
		log.Printf("[PROXY] session %d attempted listening on %s but encountered an error : %+v", sessionIdentifier, address, ret)
	} else {
		if p.debug {
			err := getIPAddress(fmt.Sprintf("http://%s", address))
			if err != nil {
				proxy.Shutdown(context.TODO())
				return -1, nil, err
			}
		}
		log.Printf("[PROXY] session %d listening on %s", sessionIdentifier, address)
	}

	return sessionIdentifier, proxy, ret
}

func getIPAddress(proxy string) error {
	request, err := http.NewRequest("GET", "https://api.ipify.org?format=json", nil)
	if err != nil {
		log.Fatalf("new request failed:%v", err)
	}

	cfg := &tls.Config{
		InsecureSkipVerify: true,
	}
	tr := &http.Transport{
		TLSClientConfig: cfg,
		Proxy: func(req *http.Request) (*url.URL, error) {
			log.Printf("Attempting to use proxy inside transport for getipaddress: %s", proxy)
			return url.Parse(proxy)
		},
	}
	client := &http.Client{
		Transport: tr,
	}
	rsp, err := client.Do(request)
	if err != nil {
		log.Printf("get rsp failed:%v", err)
		return err
	}
	defer rsp.Body.Close()
	data, _ := ioutil.ReadAll(rsp.Body)

	if rsp.StatusCode != http.StatusOK {
		log.Printf("status %d, data %s", rsp.StatusCode, data)
	}

	log.Printf("rsp:%s", data)
	return nil
}

// stripHeaders is used to ensure end to end "transparency" as the
// end user does not need to know anything was proxied, or by whom is
// was proxied by
func stripHeaders(header http.Header) http.Header {
	toBeStripped := []string{
		"Proxy-Authenticate",
		"X-Luminati-Error",
	}

	for _, toStrip := range toBeStripped {
		header.Del(toStrip)
	}

	return header
}

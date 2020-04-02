package main

import (
	"crypto/tls"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/elazarl/goproxy"
	"github.com/elazarl/goproxy/ext/auth"
)

func TestCreate(t *testing.T) {
	// Perform a test http request:
	// fake request --> (undertest) middle proxy (:8083) --> fake "end proxy" (:8082) --> fake "internet" (:8084)
	magicString := "This is only a short lived test"
	http.HandleFunc("/test", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, magicString)
	})
	go http.ListenAndServeTLS("localhost:8084", "../test-data/server.crt", "../test-data/server.key", nil)

	time.Sleep(1 * time.Second)

	username, password := "foo", "bar"

	// start end proxy server
	endProxy := goproxy.NewProxyHttpServer()
	auth.ProxyBasic(endProxy, "my_realm", func(user, pwd string) bool {
		log.Printf("Checking the passwords")
		return user == username && password == pwd
	})
	log.Println("serving end proxy server at localhost:8082")
	go http.ListenAndServe("localhost:8082", endProxy)

	underTest := Proxy{
		username:   username,
		password:   password,
		URL:        "http://localhost:8082",
		upperBound: 8083,
		lowerBound: 8083,
		freePorts:  makeRange(8083, 8083),
	}

	log.Printf("Attempting to create proxy...")
	_, proxy, err := underTest.Create(8083)
	if proxy == nil {
		log.Printf("Proxy was nil for some reason?")
	}
	if err != nil {
		log.Printf("Error encountered : %+v", err)
	}

	time.Sleep(1 * time.Second)

	proxyURL := "http://localhost:8083"
	request, err := http.NewRequest("GET", "https://127.0.0.1:8084/test", nil)
	if err != nil {
		log.Fatalf("new request failed:%v", err)
	}
	tr := &http.Transport{
		Proxy: func(req *http.Request) (*url.URL, error) {
			return url.Parse(proxyURL)
		},
		TLSClientConfig: &tls.Config{
			InsecureSkipVerify: true,
		},
	}
	client := &http.Client{Transport: tr}
	rsp, err := client.Do(request)
	if err != nil {
		log.Fatalf("get rsp failed:%v", err)

	}
	defer rsp.Body.Close()
	data, _ := ioutil.ReadAll(rsp.Body)

	if rsp.StatusCode != http.StatusOK {
		log.Fatalf("status %d, data %s", rsp.StatusCode, data)
	}

	if strings.Compare(magicString, string(data)) != 0 {
		t.Fatalf("Expected to get %s but got %s", magicString, data)
	}
}

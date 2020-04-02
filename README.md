# PRAXIS

_Warning: Quasi-abandoned project for a contract then ended up not going anywhere._

The general idea for this project was to abstract away the proxy services being used by some crawlers and OSINT gathering mechanisms. As these tend to get dished out by a client to different contractors, they always ended up in different programming languages with varying levels of "usability". In general, they always failed to support proxies, even more so, they tended to fail when having proxies integrated into them or had been pinned to a specific service.

Generically speaking, _Praxis_ was meant to handle authentication along with simple limit to prevent over use (often due to bugs in code) and maintain the state of the proxies. This allows each microservice (scrape/bot/etc) to transparently not care about the proxy they are utilizing. Due to how some proxy services upgrade/downgrade SSL traffic, it ended up causing a decent amount of issues, especially in Java.

With _Praxis_ everything is dockerized and configurable. The main proxy service supported and tested heavily is `illuminati`, which I have no affiliation to.

Essentially, it works out to work sort of like this;

```
client --> Praxis ("middle proxy") --> Proxy Service(s) ("end proxy") --> internet
```

## Setup

You'll need to create an `.env` file with the proper configuration before using `docker-compose` to build this;

```
PRAXIS_LOWER=3001
PRAXIS_UPPER=3010
SERVE_PORT=3000
PROXY_URL=http://zproxy.lum-superproxy.io:22225
PROXY_USERNAME=lum-customer-fake-customer-zone-praxis
PROXY_PASSWORD=n0tar34lp4$$w0rd
PROXY_MODE=debug
GIN_MODE=debug
AUTH_ENABLED=true
```

`PRAXIS_LOWER` and `PRAXIS_UPPER` are important as this will identify which ports for Docker to allow open, and will be used to define how many active connections it will support.
`SERVE_PORT` represents where the API will be accessable from, you can easily chain this with nginx to reverse proxy it.
`PROXY_URL`, `PROXY_USERNAME` and `PROXY_PASSWORD` are currently used to configure a `illuminati` based proxy. The current values are (obviously) not real.
`PROXY_MODE` and `GIN_MODE` can bet set to debug to allow better debugging, obviously. In theory it's faster to not have these set.
`AUTH_ENABLED` gates the authentication middlewares for the api and proxy.

After setting these up correctly, performing a `docker-compose build` followed by ` docker-compose up` should be enough.

## TODO
* Auth/Limiting doesn't work fully for proxied requests, due to how goproxy fails to pass on all request headers
* Due to how reconnect/redirects work when getting forced up to HTTPS, some non-http sites can cause issues with upstream providers (illuminati) -- potentially need to perform a work around of sorts - maybe force first call to be https? Or just overload more CONNECT messages? Or heck, just catching and throwing a better message downstream.
* Better test cases...
* If auth is disable, don't even bother with bringing up redis

## Testing
```
# Create a session
curl -vv -X POST -H 'Auth-Key:testingapikey' 127.0.0.1:3000/create
*   Trying 127.0.0.1:3000...
* TCP_NODELAY set
* Connected to 127.0.0.1 (127.0.0.1) port 3000 (#0)
> POST /create HTTP/1.1
> Host: 127.0.0.1:3000
> User-Agent: curl/7.65.3
> Accept: */*
> Auth-Key:testingapikey
> 
* Mark bundle as not supporting multiuse
< HTTP/1.1 200 OK
< Content-Type: application/json; charset=utf-8
< Date: Thu, 02 Apr 2020 05:38:37 GMT
< Content-Length: 27
< 
* Connection #0 to host 127.0.0.1 left intact
{"port":3001,"session":595}%

# Use the session generated port, 3001
curl -v --proxy-header 'Auth-Key:testingapikey' -x 127.0.0.1:3001 https://api.ipify.org\?format\=json 
*   Trying 127.0.0.1:3001...
* TCP_NODELAY set
* Connected to 127.0.0.1 (127.0.0.1) port 3001 (#0)
* allocate connect buffer!
* Establish HTTP proxy tunnel to api.ipify.org:443
> CONNECT api.ipify.org:443 HTTP/1.1
> Host: api.ipify.org:443
> User-Agent: curl/7.65.3
> Proxy-Connection: Keep-Alive
> Auth-Key:testingapikey
> 
< HTTP/1.0 200 OK
< 
* Proxy replied 200 to CONNECT request
* CONNECT phase completed!
* ALPN, offering http/1.1
* successfully set certificate verify locations:
*   CAfile: /home/diff/anaconda3/ssl/cacert.pem
  CApath: /etc/ssl/certs
* TLSv1.3 (OUT), TLS handshake, Client hello (1):
* CONNECT phase completed!
* CONNECT phase completed!
* TLSv1.3 (IN), TLS handshake, Server hello (2):
* TLSv1.2 (IN), TLS handshake, Certificate (11):
* TLSv1.2 (IN), TLS handshake, Server key exchange (12):
* TLSv1.2 (IN), TLS handshake, Server finished (14):
* TLSv1.2 (OUT), TLS handshake, Client key exchange (16):
* TLSv1.2 (OUT), TLS change cipher, Change cipher spec (1):
* TLSv1.2 (OUT), TLS handshake, Finished (20):
* TLSv1.2 (IN), TLS handshake, Finished (20):
* SSL connection using TLSv1.2 / ECDHE-RSA-AES128-GCM-SHA256
* ALPN, server did not agree to a protocol
* Server certificate:
*  subject: OU=Domain Control Validated; OU=PositiveSSL Wildcard; CN=*.ipify.org
*  start date: Jan 24 00:00:00 2018 GMT
*  expire date: Jan 23 23:59:59 2021 GMT
*  subjectAltName: host "api.ipify.org" matched cert's "*.ipify.org"
*  issuer: C=GB; ST=Greater Manchester; L=Salford; O=COMODO CA Limited; CN=COMODO RSA Domain Validation Secure Server CA
*  SSL certificate verify ok.
> GET /?format=json HTTP/1.1
> Host: api.ipify.org
> User-Agent: curl/7.65.3
> Accept: */*
> 
* Mark bundle as not supporting multiuse
< HTTP/1.1 200 OK
< Server: Cowboy
< Connection: keep-alive
< Content-Type: application/json
< Vary: Origin
< Date: Thu, 02 Apr 2020 05:39:09 GMT
< Content-Length: 22
< Via: 1.1 vegur
< 
* Connection #0 to host 127.0.0.1 left intact
{"ip":"216.74.102.71"}%
```
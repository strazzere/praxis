FROM golang:1.12.4-alpine3.9 as base

RUN set -eux; \
	apk add --no-cache --virtual git

RUN apk add --update \
        bash \
        ca-certificates \
        tzdata

# Set up LPU
RUN addgroup -S praxis && adduser --disabled-password -S praxis -G praxis
RUN mkdir /app
RUN chown -R praxis:praxis /app

# Build app
WORKDIR /go/src/app
COPY ./src/ .
RUN go get .
RUN CGO_ENABLED=0 go build -ldflags "-s -w" -o praxis-srv

FROM scratch
# Copy over user files so we can drop to least privledged user
COPY --from=base /etc/passwd /etc/passwd
COPY --from=base /app /app
COPY --from=base /go/src/app/praxis-srv /app/praxis-srv

# Explicitly pass in debug to force gin/proxy into debug mode
ENV GIN_MODE release
ENV PROXY_MODE release

# Drop to LPU and run
USER praxis
CMD ["/app/praxis-srv"]
# Build
FROM golang:1.15-alpine AS builder
ENV DOCKERVERSION=19.03.12

RUN apk add --no-cache gcc musl-dev linux-headers curl

RUN curl -fsSLO https://download.docker.com/linux/static/stable/x86_64/docker-${DOCKERVERSION}.tgz \
  && tar xzvf docker-${DOCKERVERSION}.tgz --strip 1 -C /usr/local/bin docker/docker \
  && rm docker-${DOCKERVERSION}.tgz


COPY . /node-hibernator
RUN cd /node-hibernator && go build -o node-hibernator

# Deployment
FROM alpine:latest

COPY --from=builder /node-hibernator/node-hibernator /usr/local/bin/

ENTRYPOINT ["node-hibernator"]

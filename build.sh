#!/bin/sh

cd /go/src
go get  "gopkg.in/redis.v5"
go get  "github.com/lib/pq"
cd /go/src/elasticache-scanner
go build elasticache-scanner.go


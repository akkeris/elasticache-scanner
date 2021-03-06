FROM golang:1.12-alpine
RUN apk update
RUN apk add git
RUN apk add tzdata
RUN cp /usr/share/zoneinfo/America/Denver /etc/localtime
ADD root /var/spool/cron/crontabs/root
RUN mkdir -p /go/src/elasticache-scanner
ADD elasticache-scanner.go  /go/src/elasticache-scanner/elasticache-scanner.go
ADD build.sh /build.sh
RUN chmod +x /build.sh
RUN /build.sh
CMD ["crond", "-f"]





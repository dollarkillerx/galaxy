FROM alpine:latest
WORKDIR /app

COPY ./cmd/localtime /etc/localtime
COPY ./cmd/galaxy /app

CMD ["/app/galaxy"]

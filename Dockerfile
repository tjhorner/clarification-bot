FROM golang:1.12 AS builder

WORKDIR /go/src/app
COPY . .

RUN go get -d -v ./...

RUN go build -o /app -ldflags "-linkmode external -extldflags -static" -a *.go

####

FROM alpine:latest
RUN apk update && apk add ca-certificates && rm -rf /var/cache/apk/*
COPY --from=builder /app /app
ENTRYPOINT [ "/app" ]
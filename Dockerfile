FROM golang:alpine AS builder
RUN apk update && apk add --no-cache git
RUN apk --no-cache add ca-certificates
WORKDIR $GOPATH/src/quotes
COPY . .
RUN go get -d -v
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags="-w -s" -o /go/bin/quotes

FROM scratch
WORKDIR /go/bin/
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
COPY --from=builder /go/bin/quotes /go/bin/quotes
ENTRYPOINT ["/go/bin/quotes"]
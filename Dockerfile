FROM golang:alpine AS builder
RUN apk update && apk add --no-cache git
WORKDIR $GOPATH/src/quotes
COPY . .
RUN go get -d -v
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags="-w -s" -o /go/bin/quotes

FROM scratch
WORKDIR /go/bin/
COPY --from=builder /go/bin/quotes /go/bin/quotes
ENTRYPOINT ["/go/bin/quotes"]
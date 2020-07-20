FROM golang:1.14.6-alpine3.12 AS build
WORKDIR /kube-event-tail
COPY go.mod go.sum /kube-event-tail/
RUN go mod download

COPY . /kube-event-tail/
RUN CGO_ENABLED=0 go install -ldflags "-X github.com/jrockway/opinionated-server/server.AppVersion=$(cat .version)" .

FROM alpine:3.12
RUN apk add ca-certificates tzdata
WORKDIR /
ADD https://github.com/Yelp/dumb-init/releases/download/v1.2.2/dumb-init_1.2.2_x86_64 /usr/local/bin/dumb-init
RUN chmod +x /usr/local/bin/dumb-init
ENTRYPOINT ["/usr/local/bin/dumb-init"]
COPY --from=build /go/bin/kube-event-tail /go/bin/kube-event-tail
CMD ["/go/bin/kube-event-tail"]

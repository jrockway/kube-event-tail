FROM golang:1.13-alpine AS build
RUN apk add git bzr gcc musl-dev
WORKDIR /kube-event-tail
COPY go.mod go.sum /kube-event-tail/
RUN go mod download

COPY . /kube-event-tail/
RUN go install -ldflags "-X github.com/jrockway/opinionated-server/server.AppVersion=$(cat .version)" .

FROM alpine:latest
RUN apk add ca-certificates tzdata
WORKDIR /
RUN wget -O /usr/local/bin/dumb-init https://github.com/Yelp/dumb-init/releases/download/v1.2.2/dumb-init_1.2.2_x86_64
RUN chmod +x /usr/local/bin/dumb-init
ENTRYPOINT ["/usr/local/bin/dumb-init"]
COPY --from=build /go/bin/kube-event-tail /go/bin/kube-event-tail
CMD ["/go/bin/kube-event-tail"]

FROM golang:1.20 AS build
WORKDIR /kube-event-tail
COPY go.mod go.sum /kube-event-tail/
RUN go mod download

COPY . /kube-event-tail/
RUN CGO_ENABLED=0 go install -ldflags "-X github.com/jrockway/opinionated-server/server.AppVersion=$(cat .version)" .

FROM gcr.io/distroless/static-debian10
WORKDIR /
COPY --from=build /go/bin/kube-event-tail /go/bin/kube-event-tail
CMD ["/go/bin/kube-event-tail"]

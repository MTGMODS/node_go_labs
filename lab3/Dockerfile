FROM golang:1.22-alpine AS build

WORKDIR /src
COPY go.mod ./
COPY *.go ./
RUN go test ./...
RUN CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o /pipeline .

FROM alpine:3.21

RUN adduser -D -H app
USER app
COPY --from=build /pipeline /pipeline

ENTRYPOINT ["/pipeline"]
CMD ["-tasks=10000", "-workers=4", "-buffer=100", "-iterations=10000"]

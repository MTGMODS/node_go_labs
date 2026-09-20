FROM golang:1.22-alpine AS build

WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY *.go ./
RUN go test ./...
RUN CGO_ENABLED=0 go build -o /license-service .

FROM alpine:3.20

WORKDIR /app
COPY --from=build /license-service ./license-service

ENV PORT=8080
EXPOSE 8080

CMD ["./license-service"]

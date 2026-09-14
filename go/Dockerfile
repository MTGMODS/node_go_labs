FROM golang:1.22-alpine AS build

WORKDIR /src
COPY go.mod ./
COPY *.go ./
RUN go build -o /license-service .

FROM alpine:3.20

WORKDIR /app
COPY --from=build /license-service ./license-service

ENV PORT=8080
EXPOSE 8080

CMD ["./license-service"]

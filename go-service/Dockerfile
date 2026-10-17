FROM golang:1.22-alpine AS build

WORKDIR /src
COPY go.mod ./
COPY go.sum ./
COPY . .
RUN go test ./...
RUN CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o /server ./cmd/server

FROM alpine:3.21

RUN adduser -D -H app
USER app
COPY --from=build /server /server

ENV PORT=8086
ENV DATA_DIR=/data
EXPOSE 8086
CMD ["/server"]

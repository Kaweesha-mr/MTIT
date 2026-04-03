FROM golang:1.25-alpine AS build
WORKDIR /app
COPY go.mod .
COPY . .
RUN go mod tidy
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o gateway

FROM alpine:3.19
WORKDIR /app
RUN apk add --no-cache ca-certificates
RUN mkdir -p /var/log/gateway && chmod 755 /var/log/gateway
COPY --from=build /app/gateway /app/gateway
COPY configs/.env /app/.env
EXPOSE 8080
ENTRYPOINT ["/app/gateway"]

FROM golang:1.25-alpine AS build
WORKDIR /app
COPY go.mod .
COPY . .
RUN go mod tidy
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o alert-service ./cmd/server

FROM alpine:3.19
WORKDIR /app
RUN apk add --no-cache ca-certificates
COPY --from=build /app/alert-service /app/alert-service
COPY configs ./configs
COPY init.sql /app/init.sql
COPY api ./api
EXPOSE 8085
ENTRYPOINT ["/app/alert-service"]

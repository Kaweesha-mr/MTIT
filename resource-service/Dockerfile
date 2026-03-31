FROM golang:1.25-alpine AS build
WORKDIR /app
COPY go.mod .
COPY . .
RUN go mod tidy
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o resource-service ./cmd/server

FROM alpine:3.19
WORKDIR /app
RUN apk add --no-cache ca-certificates
COPY --from=build /app/resource-service /app/resource-service
COPY api ./api
EXPOSE 8083
ENTRYPOINT ["/app/resource-service"]

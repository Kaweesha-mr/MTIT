FROM golang:1.25-alpine AS build
WORKDIR /app
COPY go.mod .
COPY . .
RUN go mod tidy
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o auth-service

FROM alpine:3.19
WORKDIR /app
COPY --from=build /app/auth-service /app/auth-service
COPY --from=build /app/api /app/api
EXPOSE 8090
ENTRYPOINT ["/app/auth-service"]

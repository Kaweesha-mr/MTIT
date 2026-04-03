# Gateway Request Logging

## Overview
The API Gateway now logs all incoming requests to a file inside the container. Each request is logged with:
- **Timestamp**: When the request was received
- **Method**: HTTP method (GET, POST, PUT, DELETE)
- **Path**: Request path (e.g., `/incidents`, `/volunteers`)
- **Service**: Which backend service the request was routed to
- **Status**: HTTP response status code
- **Duration**: How long the request took (in milliseconds)
- **Client IP**: IP address of the client
- **Cache Hit**: Whether the response was served from cache
- **Rate Limited**: Whether the request was rate limited

## Log File Location
- **Inside container**: `/var/log/gateway/requests.log`
- **Environment variable**: `GATEWAY_LOG_FILE` (defaults to `/var/log/gateway/requests.log`)

## Viewing Logs

### Show all logs
```bash
docker exec mtit-gateway cat /var/log/gateway/requests.log
```

### Tail logs in real-time
```bash
docker exec mtit-gateway tail -f /var/log/gateway/requests.log
```

### Show last 50 lines
```bash
docker exec mtit-gateway tail -50 /var/log/gateway/requests.log
```

### Show only specific service routes (e.g., incidents)
```bash
docker exec mtit-gateway grep '"service":"/incidents"' /var/log/gateway/requests.log
```

### Show only errors (status >= 400)
```bash
docker exec mtit-gateway grep -E '"status":(4|5)[0-9]{2}' /var/log/gateway/requests.log
```

### Show only cache hits
```bash
docker exec mtit-gateway grep '"cache_hit":true' /var/log/gateway/requests.log
```

### Show only rate-limited requests
```bash
docker exec mtit-gateway grep '"rate_limited":true' /var/log/gateway/requests.log
```

## Log Entry Format
Each log entry is a JSON object on a single line:
```json
{
  "timestamp": "2026-04-04T10:30:45.123Z",
  "method": "POST",
  "path": "/incidents",
  "query": "",
  "service": "/incidents",
  "status": 201,
  "duration_ms": 342,
  "client_ip": "127.0.0.1:12345",
  "cache_hit": false,
  "rate_limited": false
}
```

## Configuration
Set the log file path via environment variable:
```bash
GATEWAY_LOG_FILE=/var/log/gateway/api-requests.log
```

## Log Rotation
To rotate logs without stopping the container:
```bash
docker exec mtit-gateway sh -c 'mv /var/log/gateway/requests.log /var/log/gateway/requests.log.old && touch /var/log/gateway/requests.log'
```

## Monitoring with Custom Scripts
The JSON format makes it easy to parse logs with tools like `jq`:
```bash
# Count requests by service
docker exec mtit-gateway grep '"service"' /var/log/gateway/requests.log | jq -r '.service' | sort | uniq -c

# Average response time per service
docker exec mtit-gateway grep '"service"' /var/log/gateway/requests.log | jq -s 'group_by(.service) | map({service: .[0].service, avg_ms: (map(.duration_ms) | add / length)})' | jq .
```

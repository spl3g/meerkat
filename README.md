# Meerkat

Config-oriented monitoring and metrics collection server. Performs health checks and collects system metrics, storing results in SQLite.

## Features

- **Monitors**: HTTP and TCP connectivity checks
- **Metrics**: CPU load average collection
- **Storage**: SQLite database for observations
- **API**: REST API with API key authentication

## Quick Start

Build:
```bash
go build -o meerkat ./cmd/meerkat
```

Run:
```bash
./meerkat --config config.json --api-key your-api-key
```

## Configuration

Configuration is defined in JSON, with a format inspired by Kubernetes. Example `config.json`:

```json
{
  "name": "instance-name",
  "services": [
    {
      "name": "my-service",
      "monitors": [
        {
          "name": "http-check",
          "type": "http",
          "interval": 30,
          "url": "https://example.com",
          "method": "GET",
          "timeout": 5000
        },
        {
          "name": "tcp-check",
          "type": "tcp",
          "interval": 10,
          "hostname": "example.com",
          "port": "443",
          "timeout": 5000
        }
      ],
      "metrics": [
        {
          "name": "cpu",
          "type": "cpu",
          "interval": 5
        }
      ]
    }
  ]
}
```

### Monitor Types

**HTTP**: Checks HTTP endpoints
- `url`: Full URL (http:// or https://)
- `method`: HTTP method (default: GET)
- `timeout`: Timeout in milliseconds
- `expectedStatus`: Optional expected status code (default: 200-299)

**TCP**: Checks TCP connectivity
- `hostname`: Hostname or IP address
- `port`: Port number
- `timeout`: Timeout in milliseconds

### Metric Types

**CPU**: Collects CPU load average from `/proc/loadavg`

## API

Base path: `/api/v1`

Endpoints:
- `POST /config` - Load new configuration
- `GET /entities` - List all entities
- `GET /entities/{id}` - Get entity by ID
- `GET /heartbeats` - List heartbeat observations
- `GET /metrics` - List metric samples

Authentication: API key via `X-API-Key` header.

Swagger UI available at `/swagger/` in dev mode.

## Docker

Build and run with docker-compose:

```bash
docker-compose up
```

Requires `MEERKAT_API_KEY` environment variable.

## Environment Variables

- `MEERKAT_API_KEY` - API key for authentication (required)
- `MEERKAT_API_PORT` - API server port (default: 8080)
- `MEERKAT_DB_PATH` - Database file path (default: observations.db)
- `MEERKAT_LOG_LEVEL` - Log level: DEBUG, INFO, WARN, ERROR (default: INFO)
- `MEERKAT_LOG_FORMAT` - Log format: json or text (default: text)
- `MEERKAT_LOG_OUTPUT` - Log output: stdout, stderr, or file path (default: stdout)
- `MEERKAT_DEV_MODE` - Enable dev mode (Swagger UI) (default: false)

All options can also be set via CLI flags. See `./meerkat --help` for details.


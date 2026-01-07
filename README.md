# blayzen-sip

[![Go Version](https://img.shields.io/badge/go-1.24+-blue.svg)](https://golang.org)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)

SIP Server for [Blayzen](https://github.com/shiv6146/blayzen) Voice Agents.

Handles SIP signaling and RTP media, bridging telephony to Blayzen voice agents via WebSocket.

## Features

- **SIP Server** (UDP/TCP) using [sipgo](https://github.com/emiago/sipgo)
- **REST API** with automatic Swagger documentation
- **Inbound call routing** with custom SIP header matching
- **Outbound dialing** via configurable SIP trunks
- **PostgreSQL** for persistence
- **Valkey** for caching
- **Docker Compose** for easy deployment

## Quick Start

```bash
# Clone and start
git clone https://github.com/shiv6146/blayzen-sip
cd blayzen-sip
make quickstart

# Access
# - Swagger UI: http://localhost:8080/swagger/index.html
# - REST API:   http://localhost:8080/api/v1
# - SIP:        localhost:5060
```

## Architecture

```
┌─────────────────┐     ┌─────────────────────────────────────┐     ┌─────────────────┐
│   SIP Phone     │────▶│          blayzen-sip                │────▶│  Blayzen Agent  │
│   or PBX        │◀────│  ┌─────────┐  ┌─────────────────┐  │◀────│  (WebSocket)    │
└─────────────────┘     │  │   SIP   │  │    REST API     │  │     └─────────────────┘
                        │  │ Server  │  │  + Swagger UI   │  │
                        │  └─────────┘  └─────────────────┘  │
                        │       │              │              │
                        │  ┌────▼──────────────▼────┐        │
                        │  │   PostgreSQL  │ Valkey │        │
                        │  └────────────────────────┘        │
                        └─────────────────────────────────────┘
```

## API Documentation

Interactive API documentation is available at `/swagger/index.html` when the server is running.

### Key Endpoints

| Method | Endpoint | Description |
|--------|----------|-------------|
| GET | `/api/v1/routes` | List inbound routing rules |
| POST | `/api/v1/routes` | Create a routing rule |
| GET | `/api/v1/trunks` | List SIP trunks |
| POST | `/api/v1/trunks` | Create a SIP trunk |
| POST | `/api/v1/calls` | Initiate an outbound call |
| GET | `/api/v1/calls` | List call history |
| GET | `/health` | Health check |

### Authentication

All API endpoints (except `/health` and `/swagger/*`) require Basic Authentication:

```bash
curl -u "account-id:api-key" http://localhost:8080/api/v1/routes
```

## Configuration

Copy `env.example` to `.env` and adjust values:

```bash
cp env.example .env
```

Key configuration options:

| Variable | Default | Description |
|----------|---------|-------------|
| `SIP_PORT` | 5060 | SIP listening port |
| `API_PORT` | 8080 | REST API port |
| `DATABASE_URL` | - | PostgreSQL connection string |
| `VALKEY_URL` | localhost:6379 | Valkey/Redis URL |
| `DEFAULT_WEBSOCKET_URL` | ws://localhost:8081/ws | Fallback agent URL |

## Development

### Prerequisites

- Go 1.24+
- Docker & Docker Compose
- Make

### Local Development

```bash
# Install dependencies
make deps

# Generate Swagger docs
make swagger

# Run with hot reload
make dev

# Run tests
make test

# Lint code
make lint
```

### Docker Development

```bash
# Start all services
make docker-up

# View logs
make docker-logs

# Stop services
make docker-down

# Clean everything
make clean-all
```

## Call Routing

### Inbound Routing

Routes match inbound calls based on:
- **To User** - The called number/extension
- **From User** - The caller ID
- **Custom SIP Headers** - Any X-* header

Example: Route calls to extension 1000 to a support agent:

```bash
curl -X POST http://localhost:8080/api/v1/routes \
  -u "account-id:api-key" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "Support Line",
    "match_to_user": "1000",
    "websocket_url": "ws://support-agent:8081/ws"
  }'
```

### Outbound Dialing

Configure a SIP trunk and initiate calls:

```bash
# Create trunk
curl -X POST http://localhost:8080/api/v1/trunks \
  -u "account-id:api-key" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "Primary Trunk",
    "host": "sip.provider.com",
    "username": "user",
    "password": "pass"
  }'

# Initiate call
curl -X POST http://localhost:8080/api/v1/calls \
  -u "account-id:api-key" \
  -H "Content-Type: application/json" \
  -d '{
    "trunk_id": "trunk-uuid",
    "to": "+14155551234",
    "websocket_url": "ws://agent:8081/ws"
  }'
```

## Testing with SIP Clients

### Softphones

Configure any SIP softphone (Obi, Zoiper, Obi Obi etc.) to connect:
- **Server**: localhost:5060
- **Transport**: UDP
- **No authentication required** (for testing)

### sipp (SIP testing tool)

```bash
# Send a test INVITE
sipp -sn uac localhost:5060 -s 1000
```

## Contributing

See [CONTRIBUTING.md](CONTRIBUTING.md) for guidelines.

## License

MIT License - see [LICENSE](LICENSE) for details.

## Related Projects

- [Blayzen](https://github.com/shiv6146/blayzen) - Voice Agent Framework
- [sipgo](https://github.com/emiago/sipgo) - SIP library for Go


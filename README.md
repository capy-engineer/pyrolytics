# Pyrolytics

A Go/Gin monolith application following Hexagonal Architecture and Bounded Contexts principles.

## Project Structure

```
.
├── api/                    # API definitions (Protobuf/Swagger)
├── cmd/                    # Application entry points
│   ├── api/               # Main API server
│   └── worker/            # Background worker processes
├── config/                # Configuration files
├── docs/                  # Documentation
├── internal/              # Private application code
│   └── user/             # User bounded context
│       ├── domain/       # Core business logic
│       ├── application/  # Use cases
│       ├── ports/        # Interface definitions
│       └── adapters/     # External implementations
│           ├── http/     # HTTP handlers
│           ├── repository/ # Database implementations
│           └── messaging/  # Message queue implementations
├── migrations/            # Database migrations
└── pkg/                   # Public libraries
```

## Getting Started

### Prerequisites

- Go 1.21 or later
- Docker (for local development)
- Make (optional, for using Makefile commands)

### Installation

1. Clone the repository
2. Install dependencies:
   ```bash
   go mod download
   ```
3. Run the application:
   ```bash
   go run cmd/api/main.go
   ```

## Development

### Running Tests

```bash
go test ./...
```

### Database Migrations

```bash
migrate -path migrations -database "postgresql://user:password@localhost:5432/dbname?sslmode=disable" up
```

## Architecture

This project follows Hexagonal Architecture principles and is organized into Bounded Contexts. Each bounded context is isolated and communicates with others through well-defined interfaces.

### Key Principles

- Domain-Driven Design (DDD)
- Hexagonal Architecture
- Bounded Contexts
- Clean Architecture
- SOLID Principles

## License

MIT

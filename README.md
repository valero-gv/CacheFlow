# CacheFlow

CacheFlow is a lightweight caching system written in Go. The project starts with a simple version that will gradually expand to support distributed caching.

## Current Version (v0.1.0)

### Key Features:
- In-memory key-value storage
- TCP server for request handling
- Basic operations:
  - SET - save value
  - GET - retrieve value
  - DELETE - remove value
  - EXISTS - check key existence
- Simple CLI client
- TTL (Time To Live) support for entries
- AOF (Append-Only File) for data persistence

### Tech Stack:
- Go 1.21+
- TCP protocol
- In-memory storage
- CLI interface
- AOF for persistence

### Installation and Running:
```bash
# Install dependencies
go mod init cacheflow
go mod tidy

# Run server
go run cmd/server/main.go

# Run client
go run cmd/client/main.go
```

## Project Development Plan

### Version 0.2.0
- [ ] Enhanced data persistence
- [ ] Simple replication implementation
- [ ] Metrics and monitoring
- [ ] Improved CLI interface

### Version 0.3.0
- [ ] Sharding implementation
- [ ] Node consensus
- [ ] Enhanced replication system
- [ ] Transaction support

### Version 0.4.0
- [ ] Distributed consensus (Raft)
- [ ] Automatic failure recovery
- [ ] Enhanced monitoring system
- [ ] API for external applications

## Architecture

### Current Architecture:
```
+----------------+     +-----------------+     +------------------+
|                |     |                 |     |                  |
|  CLI Client    | <-> |  TCP Server     | <-> |  In-Memory Store|
|                |     |                 |     |                  |
+----------------+     +-----------------+     +------------------+
```

### Future Architecture:
```
+----------------+     +-----------------+     +------------------+
|                |     |                 |     |                  |
|  CLI Client    | <-> |  Load Balancer  | <-> |  Node 1         |
|                |     |                 |     |                  |
+----------------+     +-----------------+     +------------------+
                                |                      |
                                |                      |
                        +------------------+     +------------------+
                        |                  |     |                  |
                        |  Node 2          | <-> |  Node 3         |
                        |                  |     |                  |
                        +------------------+     +------------------+
```

## Communication Protocol

### Current Protocol:
```
SET key value [ttl]
GET key
DELETE key
EXISTS key
```

### Future Improvements:
- Versioning support
- Complex data structures
- Authentication and authorization
- Data encryption

## License
MIT 

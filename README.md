<div align="center">
  <h1>🏛️ FABRICA UTIL</h1>
  <p><em>Comprehensive utility library for the go-pantheon ecosystem</em></p>
</div>

<p align="center">
<a href="https://github.com/go-pantheon/fabrica-util/actions/workflows/test.yml"><img src="https://github.com/go-pantheon/fabrica-util/workflows/Test/badge.svg" alt="Test Status"></a>
<a href="https://github.com/go-pantheon/fabrica-util/releases"><img src="https://img.shields.io/github/v/release/go-pantheon/fabrica-util" alt="Latest Release"></a>
<a href="https://pkg.go.dev/github.com/go-pantheon/fabrica-util"><img src="https://pkg.go.dev/badge/github.com/go-pantheon/fabrica-util" alt="GoDoc"></a>
<a href="https://goreportcard.com/report/github.com/go-pantheon/fabrica-util"><img src="https://goreportcard.com/badge/github.com/go-pantheon/fabrica-util" alt="Go Report Card"></a>
<a href="https://github.com/go-pantheon/fabrica-util/blob/main/LICENSE"><img src="https://img.shields.io/github/license/go-pantheon/fabrica-util" alt="License"></a>
<a href="https://deepwiki.com/go-pantheon/fabrica-util"><img src="https://deepwiki.com/badge.svg" alt="Ask DeepWiki"></a>
</p>

> **Language**: [English](README.md) | [中文](README-zh.md)

## About Fabrica Util

Fabrica Util is a comprehensive utility library for the go-pantheon ecosystem, providing common functionality for all go-pantheon components. This library encapsulates reusable code patterns, algorithms, and helper functions to ensure consistency and avoid duplication in the game server microservices infrastructure.

For more information, please check out: [deepwiki/go-pantheon/fabrica-util](https://deepwiki.com/go-pantheon/fabrica-util)

## About go-pantheon Ecosystem

**go-pantheon** is an out-of-the-box game server framework providing high-performance, highly available game server cluster solutions based on microservices architecture using [go-kratos](https://github.com/go-kratos/kratos). Fabrica Util serves as the foundational utility library that supports the core components:

- **Roma**: Game core logic services
- **Janus**: Gateway service for client connection handling and request forwarding
- **Lares**: Account service for user authentication and account management
- **Senate**: Backend management service providing operational interfaces

### Core Features

- 🕒 **Time Utilities**: Advanced time handling with multi-language support, timezone management, and period calculations
- 🔄 **Concurrency**: Thread-safe synchronization primitives including delayers, futures, and goroutine management
- 🔐 **Security**: Comprehensive cryptographic utilities (AES-GCM, RSA, ECDH) for secure data transmission
- 🆔 **ID Management**: Distributed ID generation with zone-based encoding and HashID obfuscation
- 🎲 **Randomization**: Secure random number generation and string creation utilities
- 📊 **Data Structures**: High-performance implementations (Bloom filter, Bitmap, Consistent Hash)
- 🧠 **Memory Management**: Multi-pool memory management for optimized resource utilization
- 🔤 **String Processing**: Case conversion and text manipulation utilities
- ⚠️ **Error Handling**: Enhanced error handling with context and stack trace support

## Utility Packages

### Time Utilities (`xtime/`)
Advanced time handling with multi-language support:
- Configurable timezone and language support
- Time format conversion with locale-specific formatting
- Daily/weekly/monthly period calculations
- Time zone conversion utilities

### Synchronization (`xsync/`)
Thread-safe synchronization primitives:
- **Delayer**: Time-based task scheduling with expiry management
- **Future**: Asynchronous computation results
- **Closure**: Thread-safe function execution wrappers
- **Routines**: Goroutine lifecycle management

### ID Generation (`xid/`)
Distributed ID management system:
- Zone-based ID combining for multi-region support
- HashID encoding/decoding for frontend display
- ID obfuscation for security purposes

### Security (`security/`)
Comprehensive cryptographic operations:
- **AES**: AES-GCM encryption/decryption with secure nonce generation
- **RSA**: Public/private key operations
- **ECDH**: Elliptic Curve Diffie-Hellman key exchange
- **Certificate**: X.509 certificate handling utilities

### Data Structures
- **Bloom Filter** (`bloom/`): Memory-efficient set membership testing
- **Bitmap** (`bitmap/`): Bit-level operations for compact data storage
- **Consistent Hash** (`consistenthash/`): Distributed hash ring for load balancing

### Other Utilities
- **Random** (`xrand/`): Cryptographically secure random number generation
- **Compression** (`compress/`): Data compression utilities
- **CamelCase** (`camelcase/`): String case conversion utilities
- **Multi-pool** (`multipool/`): Memory pool management
- **Errors** (`errors/`): Enhanced error handling with context

## Technology Stack

| Technology/Component | Purpose                                 | Version |
| -------------------- | --------------------------------------- | ------- |
| Go                   | Primary development language            | 1.24+   |
| crypto               | Cryptographic operations                | v0.39.0 |
| go-redis             | Redis client for distributed operations | v9.10.0 |
| PostgreSQL Driver    | Database connectivity                   | v5.7.5  |
| MongoDB Driver       | NoSQL database operations               | v2.2.2  |
| HashIDs              | ID obfuscation library                  | v2.0.1  |
| Murmur3              | Fast hash algorithm                     | v1.1.0  |

## Requirements

- Go 1.24+

## Quick Start

### Installation

```bash
go get github.com/go-pantheon/fabrica-util
```

### Initialize Development Environment

```bash
make init
```

### Run Tests

```bash
make test
```

## Usage Examples

### Time Handling with Multi-language Support

```go
package main

import (
    "fmt"
    "time"

    "github.com/go-pantheon/fabrica-util/xtime"
)

func main() {
    // Initialize with configuration
    err := xtime.Init(xtime.Config{
        Language: "en",
        Timezone: "Asia/Shanghai",
    })
    if err != nil {
        panic(err)
    }

    // Format current time
    fmt.Println(xtime.Format(time.Now()))

    // Calculate next daily reset time (5 AM reset)
    nextReset := xtime.NextDailyTime(time.Now(), 5*time.Hour)
    fmt.Println("Next daily reset:", nextReset)

    // Get start of current week
    weekStart := xtime.StartOfWeek(time.Now())
    fmt.Println("Week starts at:", weekStart)
}
```

### AES-GCM Encryption

```go
package main

import (
    "fmt"

    "github.com/go-pantheon/fabrica-util/security/aes"
)

func main() {
    // Create AES cipher with 32-byte key
    key := []byte("0123456789abcdef0123456789abcdef")
    cipher, err := aes.NewAESCipher(key)
    if err != nil {
        panic(err)
    }

    data := []byte("sensitive game data")

    // Encrypt data
    encrypted, err := cipher.Encrypt(data)
    if err != nil {
        panic(err)
    }

    // Decrypt data
    decrypted, err := cipher.Decrypt(encrypted)
    if err != nil {
        panic(err)
    }

    fmt.Printf("Original: %s\n", data)
    fmt.Printf("Decrypted: %s\n", decrypted)
}
```

### Zone-based ID Management

```go
package main

import (
    "fmt"

    "github.com/go-pantheon/fabrica-util/xid"
)

func main() {
    // Combine zone ID with zone number
    playerID := int64(12345)
    zoneNum := uint8(3)
    combinedID := xid.CombineZoneID(playerID, zoneNum)

    // Encode ID for frontend display
    encodedID, err := xid.EncodeID(combinedID)
    if err != nil {
        panic(err)
    }
    fmt.Printf("Encoded ID: %s\n", encodedID)

    // Decode ID
    decodedID, err := xid.DecodeID(encodedID)
    if err != nil {
        panic(err)
    }

    // Split ID back to original components
    originalPlayerID, originalZone := xid.SplitID(decodedID)
    fmt.Printf("Player ID: %d, Zone: %d\n", originalPlayerID, originalZone)
}
```

### Synchronization with Delayer

```go
package main

import (
    "fmt"
    "time"

    "github.com/go-pantheon/fabrica-util/xsync"
)

func main() {
    // Create a delayer
    delayer := xsync.NewDelayer()
    defer delayer.Close()

    // Set expiry time (5 seconds from now)
    expiryTime := time.Now().Add(5 * time.Second)
    delayer.SetExpiryTime(expiryTime)

    fmt.Println("Waiting for delayer to expire...")

    // Wait for expiry
    select {
    case <-delayer.Wait():
        fmt.Println("Delayer expired!")
    case <-time.After(10 * time.Second):
        fmt.Println("Timeout waiting for delayer")
    }
}
```

## Project Structure

```
.
├── xtime/              # Time utilities with locale support
├── xsync/              # Synchronization primitives
│   ├── delayer.go      # Time-based task scheduling
│   ├── future.go       # Asynchronous computation
│   ├── closure.go      # Thread-safe function wrappers
│   └── routines.go     # Goroutine management
├── xrand/              # Secure random number generation
├── xid/                # ID generation and obfuscation
├── security/           # Cryptographic operations
│   ├── aes/            # AES-GCM encryption
│   ├── rsa/            # RSA encryption
│   ├── ecdh/           # Elliptic Curve Diffie-Hellman
│   └── certificate/    # X.509 certificate utilities
├── consistenthash/     # Consistent hash implementation
├── multipool/          # Memory pool management
├── errors/             # Enhanced error handling
├── bloom/              # Bloom filter implementation
├── compress/           # Data compression utilities
├── bitmap/             # Bitmap data structure
└── camelcase/          # String case conversion
```

## Integration with go-pantheon Components

Fabrica Util is designed to be imported by other go-pantheon components:

```go
import (
    // Security utilities for token generation in Lares
    "github.com/go-pantheon/fabrica-util/security/aes"

    // Time utilities for game logic in Roma
    "github.com/go-pantheon/fabrica-util/xtime"

    // Synchronization utilities for connection handling in Janus
    "github.com/go-pantheon/fabrica-util/xsync"

    // ID management for distributed player identification
    "github.com/go-pantheon/fabrica-util/xid"
)
```

## Development Guide

### License Compliance

The project enforces license compliance for all dependencies. We only allow the following licenses:
- MIT
- Apache-2.0
- BSD-2-Clause
- BSD-3-Clause
- ISC
- MPL-2.0

License checks are performed:
- Automatically in CI/CD pipelines
- Locally via pre-commit hooks
- Manually using `make license-check`

### Testing

Run the complete test suite:

```bash
# Run all tests with coverage
make test

# Run linting
make lint

# Run go vet
make vet
```

### Adding New Utilities

When adding new utility functions:

1. Create a new package or add to an existing one based on functionality
2. Implement the utility functions with proper error handling
3. Write comprehensive unit tests with edge cases covered
4. Document usage with clear examples
5. Ensure thread safety where applicable
6. Run tests: `make test`
7. Update documentation if needed

### Contribution Guidelines

1. Fork this repository
2. Create a feature branch from `main`
3. Implement changes with comprehensive tests
4. Ensure all tests pass and linting is clean
5. Update documentation for any API changes
6. Submit a Pull Request with clear description

## Performance Considerations

- **Memory Pools**: Use `multipool` for high-frequency object allocation
- **Cryptography**: AES-GCM operations are optimized for throughput
- **ID Generation**: HashID encoding is cached for repeated operations
- **Time Operations**: Timezone loading is cached and reused
- **Synchronization**: All sync primitives are designed for low contention

## License

This project is licensed under the terms specified in the LICENSE file.

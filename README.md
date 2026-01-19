# Mono Contrib

> Repository for third-party middleware and plugin implementations for the [Mono Framework](https://github.com/go-monolith/mono).

<div align="center">

[![Go Reference](https://pkg.go.dev/badge/github.com/go-monolith/contrib.svg)](https://pkg.go.dev/github.com/go-monolith/contrib)
[![Go Report Card](https://goreportcard.com/badge/github.com/go-monolith/contrib)](https://goreportcard.com/report/github.com/go-monolith/contrib)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)

</div>

## Overview

This repository contains community-contributed middleware modules, plugins, and integrations for the Mono Framework. Each package is designed to extend the functionality of Mono applications with minimal configuration.

> **Go version support:** We support the latest two versions of Go. Visit [https://go.dev/doc/devel/release](https://go.dev/doc/devel/release) for more information.

## Middleware Implementations

| Middleware | Description | Documentation |
|------------|-------------|---------------|
| [jwt](./v1/jwt/) | JWT authentication with multi-strategy support (HMAC, RSA, ECDSA, JWKS) | [README](./v1/jwt/README.md) |
| [otel](./v1/otel/) | OpenTelemetry instrumentation (metrics, traces, logs) | [README](./v1/otel/README.md) |

## Installation

Each middleware can be installed independently:

```bash
# JWT Middleware
go get github.com/go-monolith/contrib/v1/jwt

# OTEL Middleware
go get github.com/go-monolith/contrib/v1/otel
```

## Usage Example

```go
package main

import (
    "context"
    "log"

    "github.com/go-monolith/mono"
    "github.com/go-monolith/contrib/v1/otel"
    sdkmetric "go.opentelemetry.io/otel/sdk/metric"
)

func main() {
    // Create OTEL provider
    meterProvider := sdkmetric.NewMeterProvider(/* ... */)

    // Create middleware
    otelMw, err := otel.New(
        otel.WithMeterProvider(meterProvider),
    )
    if err != nil {
        log.Fatal(err)
    }

    // Create Mono application
    app, err := mono.NewMonoApplication()
    if err != nil {
        log.Fatal(err)
    }

    // Register middleware BEFORE other modules
    app.Register(otelMw)
    app.Register(&MyModule{})

    // Start
    app.Start(context.Background())
}
```

## Contributing

We welcome contributions! Please see our [Contributing Guide](CONTRIBUTING.md) for details on how to submit pull requests, report issues, and contribute to the project.

### Adding a New Middleware

1. Create a new directory under `v1/` (e.g., `v1/my-middleware/`)
2. Implement the `mono.MiddlewareModule` interface
3. Add comprehensive tests
4. Create a README.md with documentation
5. Update this README.md to list your middleware
6. Submit a pull request

## Directory Structure

```
contrib/
├── v1/                     # Version 1 middleware implementations
│   ├── jwt/                # JWT authentication middleware
│   │   ├── README.md
│   │   ├── go.mod
│   │   ├── jwt.go
│   │   ├── config.go
│   │   ├── options.go
│   │   ├── validator.go
│   │   ├── jwks.go
│   │   ├── provider.go
│   │   └── *_test.go
│   └── otel/               # OpenTelemetry middleware
│       ├── README.md
│       ├── go.mod
│       ├── otel.go
│       ├── config.go
│       ├── options.go
│       ├── metrics.go
│       ├── traces.go
│       ├── logs.go
│       ├── propagation.go
│       └── *_test.go
├── examples/               # Example applications
├── docs/                   # Documentation
├── scripts/                # Build and utility scripts
├── go.work                 # Go workspace file
└── README.md               # This file
```

## License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.

## Related Projects

- [Mono Framework](https://github.com/go-monolith/mono) - The core modular monolith framework
- [Mono Recipes](https://github.com/go-monolith/mono-recipes) - Example projects and recipes

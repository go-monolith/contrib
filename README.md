# Mono Contrib

> Repository for third-party middleware and plugin implementations for the [Mono Framework](https://github.com/go-monolith/mono).

<div align="center">

[![Go Reference](https://pkg.go.dev/badge/github.com/go-monolith/contrib.svg)](https://pkg.go.dev/github.com/go-monolith/contrib)
[![Go Report Card](https://goreportcard.com/badge/github.com/go-monolith/contrib)](https://goreportcard.com/report/github.com/go-monolith/contrib)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)

</div>

## Overview

This repository contains community-contributed middleware modules, plugins, and integrations for the Mono Framework. Each package is designed to extend the functionality of Mono applications with minimal configuration.

> **Go version support:** each module supports the Go version in the `go` directive of its `go.mod` (currently Go 1.26) and newer. CI tests that minimum and the latest stable release. Visit [https://go.dev/doc/devel/release](https://go.dev/doc/devel/release) for more information.

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
6. Add a `.github/workflows/go-test-<name>.yaml`, copied from `go-test-jwt.yaml`, and add the directory to `.github/dependabot.yml`
7. Submit a pull request

### Releases

Each module under `v1/` is released on its own, with a tag `v1/<name>/vX.Y.Z`. Go downloads a version from that tag; a GitHub release adds the notes.

1. Give the pull request a conventional title: `feat(jwt): ...`, `fix(otel): ...`, `chore(deps): ...`, `docs: ...`, or `feat!: ...` for a breaking change. An issue reference may come first (`#12 fix(jwt): ...`).
2. The PR labeler workflow labels the pull request from its title. Labels set by hand are kept. The label decides two things:
   - **Section of the release notes:**
     - `❗ BreakingChange`: Breaking Changes
     - `✏️ Feature`: New
     - `☢️ Bug`: Fixes
     - `🤖 Dependencies` and `🧹 Updates`: Updates
     - `📒 Documentation`: Documentation
   - **Next version:**
     - `❗ BreakingChange` or `major`: major
     - `✏️ Feature` or `minor`: minor
     - anything else: patch
3. Every merge to `main` updates a draft release for each module the merge changed (`go-release.yaml`).
4. To release, review the draft on the [releases page](https://github.com/go-monolith/contrib/releases): adjust the version and notes if needed, then publish it. Publishing creates the tag on `main`. GitHub marks only one release of the repository as latest, which does not affect `go get`.

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

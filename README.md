# controller-util

A Go library providing common utilities and patterns for building Kubernetes controllers using [controller-runtime](https://github.com/kubernetes-sigs/controller-runtime).

## Overview

`controller-util` simplifies Kubernetes controller development by providing:

- **Component-based reconciliation**: Build controllers from composable, reusable components
- **Finalizer management**: Automatic finalizer handling with component-level cleanup
- **Condition helpers**: Simplified status condition management
- **Action utilities**: Common operations for owned resources (create, update, delete)
- **Metadata providers**: Standard Kubernetes label generation
- **Patch utilities**: Strategic merge patching for resource updates

## Installation

```bash
go get github.com/dominodatalab/controller-util
```

## Usage

### Basic Controller Setup

```go
import (
    "github.com/dominodatalab/controller-util/core"
    ctrl "sigs.k8s.io/controller-runtime"
)

// Create a reconciler with manager
reconciler := core.NewReconciler(mgr).
    For(&v1.YourCRD{}).
    Named("your-controller").
    Component("component-name", yourComponent).
    Build()
```

### Component Interface

Components implement the `core.Component` interface:

```go
type Component interface {
    Reconcile(*Context) (ctrl.Result, error)
}
```

Components can also implement optional interfaces:
- `OwnedComponent`: For resources owned by the controller
- `FinalizerComponent`: For cleanup logic during deletion
- `InitializerComponent`: For setup during controller initialization

### Condition Management

```go
func (c *YourComponent) Reconcile(ctx *core.Context) (ctrl.Result, error) {
    // Set conditions on your CRD status
    ctx.Conditions.SetTrue("Ready", "AllComponentsReady", "All components reconciled successfully")
    ctx.Conditions.SetFalse("Available", "PodNotReady", "Pod is not ready yet")
    
    return ctrl.Result{}, nil
}
```

### Action Utilities

```go
import "github.com/dominodatalab/controller-util/action"

// Create or update an owned resource
err := action.CreateOrUpdateOwnedResource(ctx, owner, deployment)

// Delete resources if they exist
err := action.DeleteIfExists(ctx, service, configMap)

// Delete all PVCs matching label selector
err := action.DeleteStorage(ctx, []client.ListOption{
    client.InNamespace(namespace),
    client.MatchingLabels(labels),
})
```

### Metadata Helpers

```go
import "github.com/dominodatalab/controller-util/metadata"

provider := metadata.NewProvider("myapp",
    metadata.WithCreator("myapp-controller"),
    metadata.WithManager("myapp-operator"),
)

// Generate standard labels
labels := provider.StandardLabels(obj, metadata.AppComponent("api"), extraLabels)

// Generate match labels for selectors
matchLabels := provider.MatchLabels(obj, metadata.AppComponent("api"))
```

## Development

### Prerequisites

- Go 1.25+
- golangci-lint (installed via `make tools`)

### Common Tasks

```bash
make help           # Show all available targets
make test           # Run tests
make lint           # Run linters
make fmt            # Format code
make ci             # Run all CI checks
make tools          # Install development tools
```

## Dependencies

- Kubernetes v0.34.x
- controller-runtime v0.22.x
- Go 1.25+

## License

See [LICENSE](LICENSE) file for details.

## Related Projects

- [hephaestus](https://github.com/dominodatalab/hephaestus) - Uses controller-util for building ImageBuild controllers

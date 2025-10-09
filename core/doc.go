// Package core provides the fundamental building blocks for creating Kubernetes controllers
// using a component-based architecture.
//
// The core package includes:
//   - Reconciler: A flexible, component-based reconciler that manages the reconciliation lifecycle
//   - Component interfaces: Define reusable controller components with optional lifecycle hooks
//   - Context: Rich context object passed to components containing clients, loggers, and helpers
//   - Condition helpers: Utilities for managing Kubernetes status conditions
//   - Patch utilities: Strategic merge patching support for resource updates
//
// # Basic Usage
//
// Create a reconciler with components:
//
//	reconciler := core.NewReconciler(mgr).
//	    For(&v1.MyCRD{}).
//	    Named("my-controller").
//	    Component("deployment", &DeploymentComponent{}).
//	    Component("service", &ServiceComponent{}).
//	    Build()
//
// # Component Architecture
//
// Components implement the Component interface and handle specific resource types:
//
//	type MyComponent struct{}
//
//	func (c *MyComponent) Reconcile(ctx *core.Context) (ctrl.Result, error) {
//	    // Reconciliation logic
//	    return ctrl.Result{}, nil
//	}
//
// Components can implement optional interfaces:
//   - OwnedComponent: Declare resource ownership for automatic watch setup
//   - FinalizerComponent: Define cleanup logic during resource deletion
//   - InitializerComponent: Perform setup during controller initialization
//
// # Context Object
//
// The Context provides components with everything needed for reconciliation:
//   - Client: Kubernetes client for API operations
//   - Log: Structured logger specific to the component
//   - Object: The resource being reconciled
//   - Conditions: Helper for managing status conditions
//   - Patch: Utilities for calculating and applying patches
//   - Recorder: Event recorder for Kubernetes events
//   - Scheme: Runtime scheme for object conversion
//
// # Condition Management
//
// Use the condition helper to set status conditions:
//
//	ctx.Conditions.SetTrue("Ready", "ComponentReady", "Component is ready")
//	ctx.Conditions.SetFalse("Available", "PodNotReady", "Pod not ready")
//
// Conditions are automatically flushed to the resource status at the end of reconciliation.
//
// # Finalizers
//
// Components implementing FinalizerComponent automatically get finalizers:
//
//	func (c *MyComponent) Finalize(ctx *core.Context) (ctrl.Result, bool, error) {
//	    // Cleanup logic
//	    return ctrl.Result{}, true, nil // true indicates finalization is complete
//	}
//
// The reconciler automatically:
//   - Registers finalizers when resources are created
//   - Calls Finalize when resources are being deleted
//   - Removes finalizers when cleanup is complete
//
// # Patching
//
// The Patch utility enables efficient resource updates using strategic merge:
//
//	patchResult, err := ctx.Patch.Maker.Calculate(current, desired, opts...)
//	if !patchResult.IsEmpty() {
//	    // Apply changes
//	}
//
// This ensures only actual changes trigger updates, reducing API server load.
package core

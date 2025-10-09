// Package metadata provides utilities for generating standard Kubernetes labels and metadata
// following the recommended Kubernetes label conventions.
//
// This package implements the standard Kubernetes application labels as defined in:
// https://kubernetes.io/docs/concepts/overview/working-with-objects/common-labels/
//
// # Standard Label Keys
//
// The package defines constants for all standard application labels:
//   - app.kubernetes.io/name: The name of the application
//   - app.kubernetes.io/instance: A unique name identifying the instance
//   - app.kubernetes.io/version: The current version of the application
//   - app.kubernetes.io/component: The component within the architecture
//   - app.kubernetes.io/part-of: The name of a higher level application
//   - app.kubernetes.io/managed-by: The tool being used to manage the application
//   - app.kubernetes.io/created-by: The controller/user who created the resource
//
// # Provider Usage
//
// Create a metadata provider for your application:
//
//	provider := metadata.NewProvider("myapp",
//	    metadata.WithCreator("myapp-controller"),
//	    metadata.WithManager("myapp-operator"),
//	    metadata.WithVersion(func(obj client.Object) string {
//	        return obj.(*v1.MyApp).Spec.Version
//	    }),
//	)
//
// # Generating Labels
//
// Use StandardLabels for comprehensive label sets:
//
//	labels := provider.StandardLabels(myapp, metadata.AppComponent("api"), extraLabels)
//	// Returns: {
//	//   "app.kubernetes.io/name": "myapp",
//	//   "app.kubernetes.io/instance": "myapp-prod",
//	//   "app.kubernetes.io/component": "api",
//	//   "app.kubernetes.io/created-by": "myapp-controller",
//	//   "app.kubernetes.io/managed-by": "myapp-operator",
//	//   ... plus any extra labels
//	// }
//
// Use MatchLabels for label selectors:
//
//	matchLabels := provider.MatchLabels(myapp, metadata.AppComponent("api"))
//	// Returns minimal set for matching:
//	// {
//	//   "app.kubernetes.io/name": "myapp",
//	//   "app.kubernetes.io/instance": "myapp-prod",
//	//   "app.kubernetes.io/component": "api",
//	// }
//
// # Instance Names
//
// Generate consistent names for resources:
//
//	name := provider.InstanceName(myapp, metadata.AppComponent("api"))
//	// Returns: "myapp-prod-myapp-api"
//
//	name := provider.InstanceName(myapp, metadata.AppComponentNone)
//	// Returns: "myapp-prod-myapp"
//
// # Dynamic Labels
//
// Add dynamic labels based on the resource:
//
//	provider := metadata.NewProvider("myapp",
//	    metadata.WithDynamicLabels(func(obj client.Object) map[string]string {
//	        myapp := obj.(*v1.MyApp)
//	        return map[string]string{
//	            "environment": myapp.Spec.Environment,
//	            "tier":        myapp.Spec.Tier,
//	        }
//	    }),
//	)
//
// # Example Component
//
//	type ServiceComponent struct {
//	    provider *metadata.Provider
//	}
//
//	func NewServiceComponent() *ServiceComponent {
//	    return &ServiceComponent{
//	        provider: metadata.NewProvider("myapp",
//	            metadata.WithCreator("myapp-controller"),
//	        ),
//	    }
//	}
//
//	func (c *ServiceComponent) buildService(myapp *v1.MyApp) *corev1.Service {
//	    labels := c.provider.StandardLabels(myapp, metadata.AppComponentNone, nil)
//	    matchLabels := c.provider.MatchLabels(myapp, metadata.AppComponentNone)
//
//	    return &corev1.Service{
//	        ObjectMeta: metav1.ObjectMeta{
//	            Name:      c.provider.InstanceName(myapp, metadata.AppComponentNone),
//	            Namespace: myapp.Namespace,
//	            Labels:    labels,
//	        },
//	        Spec: corev1.ServiceSpec{
//	            Selector: matchLabels,
//	            // ...
//	        },
//	    }
//	}
//
// # Benefits
//
// Using this package ensures:
//   - Consistent labeling across all resources
//   - Compliance with Kubernetes best practices
//   - Easy resource querying and filtering
//   - Clear ownership and management tracking
//   - Support for multi-tenancy and multi-component applications
package metadata

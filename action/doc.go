// Package action provides high-level utilities for common resource management operations
// in Kubernetes controllers.
//
// This package simplifies common controller patterns for creating, updating, and deleting
// Kubernetes resources while properly handling ownership and error conditions.
//
// # Creating Owned Resources
//
// Use CreateOwnedResource to create a resource with proper controller ownership:
//
//	deployment := buildDeployment(myapp)
//	err := action.CreateOwnedResource(ctx, myapp, deployment)
//
// This automatically:
//   - Sets the controller reference
//   - Creates the resource if it doesn't exist
//   - Returns nil if the resource already exists
//
// # Creating or Updating Resources
//
// Use CreateOrUpdateOwnedResource for idempotent resource management:
//
//	service := buildService(myapp)
//	err := action.CreateOrUpdateOwnedResource(ctx, myapp, service)
//
// This automatically:
//   - Creates the resource if it doesn't exist
//   - Calculates a strategic merge patch if it exists
//   - Updates only when changes are detected
//   - Preserves generated fields (e.g., Service.Spec.ClusterIP)
//
// # Deleting Resources
//
// Use DeleteIfExists to safely delete resources:
//
//	err := action.DeleteIfExists(ctx, deployment, service, configMap)
//
// This:
//   - Deletes each resource if it exists
//   - Returns nil if resources don't exist (idempotent)
//   - Returns errors for actual deletion failures
//
// # Deleting Storage
//
// Use DeleteStorage to clean up PersistentVolumeClaims:
//
//	opts := []client.ListOption{
//	    client.InNamespace(namespace),
//	    client.MatchingLabels(labels),
//	}
//	err := action.DeleteStorage(ctx, opts)
//
// This is useful during finalizer cleanup to remove persistent storage.
//
// # Special Handling
//
// CreateOrUpdateOwnedResource handles special cases for certain resource types:
//   - Service: Preserves Spec.ClusterIP (immutable generated field)
//   - Job: Preserves Spec.Selector (immutable generated field)
//
// # Example Component
//
//	type DeploymentComponent struct{}
//
//	func (c *DeploymentComponent) Reconcile(ctx *core.Context) (ctrl.Result, error) {
//	    myapp := ctx.Object.(*v1.MyApp)
//	    deployment := c.buildDeployment(myapp)
//
//	    // Creates or updates the deployment
//	    if err := action.CreateOrUpdateOwnedResource(ctx, myapp, deployment); err != nil {
//	        return ctrl.Result{}, err
//	    }
//
//	    return ctrl.Result{}, nil
//	}
//
//	func (c *DeploymentComponent) Finalize(ctx *core.Context) (ctrl.Result, bool, error) {
//	    myapp := ctx.Object.(*v1.MyApp)
//	    deployment := &appsv1.Deployment{
//	        ObjectMeta: metav1.ObjectMeta{
//	            Name:      "myapp-deployment",
//	            Namespace: myapp.Namespace,
//	        },
//	    }
//
//	    // Clean up deployment during deletion
//	    if err := action.DeleteIfExists(ctx, deployment); err != nil {
//	        return ctrl.Result{}, false, err
//	    }
//
//	    return ctrl.Result{}, true, nil
//	}
package action

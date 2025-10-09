// Package collection provides utility functions for working with Go collections,
// particularly focused on map operations commonly needed in Kubernetes controllers.
//
// This package contains helper functions that simplify common collection manipulation
// tasks when building controller components.
//
// # Map Utilities
//
// MergeStringMaps merges string maps with source values taking precedence:
//
//	base := map[string]string{
//	    "app": "myapp",
//	    "env": "dev",
//	}
//	additional := map[string]string{
//	    "env": "prod",      // This will override base["env"]
//	    "tier": "frontend", // This will be added
//	}
//
//	result := collection.MergeStringMaps(additional, base)
//	// result = {
//	//   "app": "myapp",
//	//   "env": "prod",      // Overridden
//	//   "tier": "frontend", // Added
//	// }
//
// # Common Use Cases
//
// Combining standard labels with custom labels:
//
//	standardLabels := map[string]string{
//	    "app.kubernetes.io/name": "myapp",
//	    "app.kubernetes.io/instance": "prod",
//	}
//	customLabels := map[string]string{
//	    "environment": "production",
//	    "team": "platform",
//	}
//
//	allLabels := collection.MergeStringMaps(customLabels, standardLabels)
//
// Merging user-provided labels with defaults:
//
//	func buildPodLabels(userLabels map[string]string) map[string]string {
//	    defaults := map[string]string{
//	        "app": "myapp",
//	        "version": "v1",
//	    }
//	    return collection.MergeStringMaps(userLabels, defaults)
//	}
//
// # Note on Mutability
//
// MergeStringMaps modifies the destination map in place and returns it for convenience.
// If you need to preserve the original destination map, create a copy first:
//
//	original := map[string]string{"key": "value"}
//	copy := make(map[string]string, len(original))
//	for k, v := range original {
//	    copy[k] = v
//	}
//	result := collection.MergeStringMaps(newValues, copy)
//	// original is unchanged, result contains merged values
package collection

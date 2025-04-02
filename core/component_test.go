package core

import (
	"testing"

	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// These tests dont execute any code, they  verify the types compile
// If they compile, then the Component interface is correctly defined

func TestComponentInterface(t *testing.T) {
	// This test doesn't execute any code, it just verifies the types compile
	// If this compiles, then the Component interface is correctly defined
	var _ Component = (*testComponent)(nil)
}

func TestOwnedComponentInterface(t *testing.T) {
	var _ Component = (*testOwnedComponent)(nil)
	var _ OwnedComponent = (*testOwnedComponent)(nil)
}

func TestInitializerComponentInterface(t *testing.T) {
	var _ Component = (*initializerComponent)(nil)
	var _ InitializerComponent = (*initializerComponent)(nil)
}

func TestFinalizerComponentInterface(t *testing.T) {
	var _ Component = (*finalizerComponent)(nil)
	var _ FinalizerComponent = (*finalizerComponent)(nil)
}

type testComponent struct{}

func (c *testComponent) Reconcile(*Context) (ctrl.Result, error) {
	return ctrl.Result{}, nil
}

type testOwnedComponent struct {
	testComponent
}

func (c *testOwnedComponent) Reconcile(*Context) (ctrl.Result, error) {
	return ctrl.Result{}, nil
}

func (c *testOwnedComponent) Kind() client.Object {
	return nil
}

type initializerComponent struct {
	testComponent
}

func (c *initializerComponent) Initialize(*Context, *ctrl.Builder) error {
	return nil
}

type finalizerComponent struct {
	testComponent
}

func (c *finalizerComponent) Finalize(*Context) (ctrl.Result, bool, error) {
	return ctrl.Result{}, false, nil
}

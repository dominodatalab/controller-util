package core

import (
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type Component interface {
	Reconcile(ctx *Context) (ctrl.Result, error)
}

type OwnedComponent interface {
	Component
	Kind() client.Object
}

type InitializerComponent interface {
	Initialize(ctx *Context, builder *ctrl.Builder) error
}

type FinalizerComponent interface {
	Finalize(ctx *Context) (ctrl.Result, bool, error)
}

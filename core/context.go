package core

import (
	"context"

	"github.com/go-logr/logr"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/record"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type ContextData map[string]interface{}

type Context struct {
	context.Context

	Log        logr.Logger
	Data       ContextData
	Patch      *Patch
	Object     client.Object
	Config     *rest.Config
	Client     client.Client
	Scheme     *runtime.Scheme
	Recorder   record.EventRecorder
	Conditions *conditionHelper
}

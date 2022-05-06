package core

import (
	"context"
	"fmt"
	"path"
	"strings"

	"github.com/go-logr/logr"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	utilerrors "k8s.io/apimachinery/pkg/util/errors"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/apiutil"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

var getGvk = apiutil.GVKForObject

const SkipReconcileAnnotation = "controller-util.dominodatalab.com/skip-reconcile"

type reconcilerComponent struct {
	name string
	comp Component

	finalizer     FinalizerComponent
	finalizerName string
}

type Reconciler struct {
	name              string
	resourceName      string
	mgr               ctrl.Manager
	controllerBuilder *ctrl.Builder
	apiType           client.Object
	config            *rest.Config
	client            client.Client
	log               logr.Logger
	abortNotFound     bool
	webhooksEnabled   bool
	finalizerBaseName string

	patcher     *Patch
	recorder    record.EventRecorder
	controller  controller.Controller
	components  []*reconcilerComponent
	contextData ContextData
}

func NewReconciler(mgr ctrl.Manager) *Reconciler {
	return &Reconciler{
		mgr:               mgr,
		config:            mgr.GetConfig(),
		client:            mgr.GetClient(),
		components:        []*reconcilerComponent{},
		controllerBuilder: builder.ControllerManagedBy(mgr),
		contextData:       ContextData{},
		abortNotFound:     true,
	}
}

func (r *Reconciler) For(apiType client.Object, opts ...builder.ForOption) *Reconciler {
	r.apiType = apiType
	r.controllerBuilder = r.controllerBuilder.For(apiType, opts...)

	return r
}

func (r *Reconciler) Component(name string, comp Component, opts ...builder.OwnsOption) *Reconciler {
	rc := &reconcilerComponent{name: name, comp: comp}

	if ownedComp, ok := comp.(OwnedComponent); ok {
		r.controllerBuilder.Owns(ownedComp.Kind(), opts...)
	}
	if finalizer, ok := comp.(FinalizerComponent); ok {
		rc.finalizer = finalizer
	}
	r.components = append(r.components, rc)

	return r
}

func (r *Reconciler) Named(name string) *Reconciler {
	r.name = name
	r.controllerBuilder.Named(name)
	return r
}

func (r *Reconciler) ReconcileNotFound() *Reconciler {
	r.abortNotFound = false
	return r
}

func (r *Reconciler) WithContextData(key string, obj interface{}) *Reconciler {
	r.contextData[key] = obj
	return r
}

func (r *Reconciler) WithControllerOptions(opts controller.Options) *Reconciler {
	// this library dynamically builds a reconciler, hence, we do not allow an override here
	opts.Reconciler = nil

	r.controllerBuilder.WithOptions(opts)
	return r
}

func (r *Reconciler) WithWebhooks() *Reconciler {
	r.webhooksEnabled = true
	return r
}

func (r *Reconciler) Build() (controller.Controller, error) {
	name, err := r.getControllerName()
	if err != nil {
		return nil, fmt.Errorf("cannot compute controller name: %w", err)
	}
	r.name = name
	r.log = ctrl.Log.WithName("controller").WithName(name)
	r.recorder = r.mgr.GetEventRecorderFor(fmt.Sprintf("%s-%s", r.name, "controller"))

	gvk, err := getGvk(r.apiType, r.mgr.GetScheme())
	if err != nil {
		return nil, fmt.Errorf("cannot get GVK for object %#v: %w", r.apiType, err)
	}

	// resource name should reference api type regardless of controller name
	r.resourceName = strings.ToLower(gvk.Kind)

	// configure finalizer base path and patcher
	if r.finalizerBaseName == "" {
		r.finalizerBaseName = fmt.Sprintf("%s.%s/", name, gvk.Group)
	}
	if r.patcher == nil {
		r.patcher = NewPatch(gvk)
	}

	// minimal context for initializer components (if any)
	initCtx := &Context{
		Context: context.Background(),
		Client:  r.client,
		Scheme:  r.mgr.GetScheme(),
		Data:    r.contextData,
	}
	initLog := r.log.WithName("component")

	components := map[string]Component{}
	for _, rc := range r.components {
		orig, ok := components[rc.name]
		if ok {
			return nil, fmt.Errorf("duplicate component found using name %s: %#v %#v", rc.name, orig, rc.comp)
		}
		rc.finalizerName = path.Join(r.finalizerBaseName, rc.name)

		components[rc.name] = rc.comp

		initComp, ok := rc.comp.(InitializerComponent)
		if !ok {
			continue
		}
		initCtx.Log = initLog.WithName(rc.name)

		if err = initComp.Initialize(initCtx, r.controllerBuilder); err != nil {
			return nil, fmt.Errorf("cannot initialize component %s in controller %s: %w", rc.name, r.name, err)
		}
	}

	r.controller, err = r.controllerBuilder.Build(r)
	if err != nil {
		return nil, fmt.Errorf("unable to build controller: %w", err)
	}

	// setup webhooks
	if r.webhooksEnabled {
		err := ctrl.NewWebhookManagedBy(r.mgr).For(r.apiType).Complete()
		if err != nil {
			return nil, fmt.Errorf("unable to build webhook: %w", err)
		}
	}

	return r.controller, nil
}

func (r *Reconciler) Complete() error {
	_, err := r.Build()
	return err
}

func (r *Reconciler) Reconcile(rootCtx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := r.log.WithValues(r.resourceName, req.NamespacedName)
	log.Info("Starting reconcile")

	// fetch event api object
	obj := r.apiType.DeepCopyObject().(client.Object)
	if err := r.client.Get(rootCtx, req.NamespacedName, obj); err != nil {
		if !apierrors.IsNotFound(err) {
			log.Error(err, "Failed to fetch reconcile object")
			return ctrl.Result{}, err
		}

		if r.abortNotFound {
			log.Info("Aborting reconcile, object not found (assuming it was deleted)")
			return ctrl.Result{}, nil
		}

		obj.SetName(req.Name)
		obj.SetNamespace(req.Namespace)
	}
	cleanObj := obj.DeepCopyObject().(client.Object)

	// skip reconcile when annotated
	skip, ok := obj.GetAnnotations()[SkipReconcileAnnotation]
	if ok && skip == "true" {
		log.Info("Skipping reconcile due to annotation")
		return ctrl.Result{}, nil
	}

	// build context for components
	compLog := log.WithName("component")
	ctx := &Context{
		Context:    rootCtx,
		Object:     obj,
		Config:     r.config,
		Client:     r.client,
		Patch:      r.patcher,
		Scheme:     r.mgr.GetScheme(),
		Recorder:   r.recorder,
		Conditions: NewConditionHelper(obj),
		Data:       r.contextData,
	}

	// reconcile components
	var finalRes ctrl.Result
	var errs []error
	for _, rc := range r.components {
		res := ctrl.Result{}
		var err error

		ctx.Log = compLog.WithName(rc.name)

		if ctx.Object.GetDeletionTimestamp().IsZero() {
			log.Info("Reconciling component", "component", rc.name)
			res, err = rc.comp.Reconcile(ctx)

			if rc.finalizer != nil && !controllerutil.ContainsFinalizer(ctx.Object, rc.finalizerName) {
				log.Info("Registering finalizer", "component", rc.name)
				controllerutil.AddFinalizer(ctx.Object, rc.finalizerName)
			}
		} else if rc.finalizer != nil && controllerutil.ContainsFinalizer(ctx.Object, rc.finalizerName) {
			log.Info("Finalizing component", "component", rc.name)

			var done bool
			res, done, err = rc.finalizer.Finalize(ctx)
			if done {
				log.Info("Removing finalizer", "component", rc.name)
				controllerutil.RemoveFinalizer(ctx.Object, rc.finalizerName)
			}
		}

		ctx.Conditions.Flush()
		if res.Requeue {
			finalRes.Requeue = true
		}
		if res.RequeueAfter != 0 && (finalRes.RequeueAfter == 0 || finalRes.RequeueAfter > res.RequeueAfter) {
			finalRes.RequeueAfter = res.RequeueAfter
		}
		if err != nil {
			log.Error(err, "Component reconciliation failed", "component", rc.name)
			errs = append(errs, err)
		}
	}

	if !r.abortNotFound {
		// patch metadata and status when changes occur
		currentMeta := r.apiType.DeepCopyObject().(client.Object)
		currentMeta.SetName(ctx.Object.GetName())
		currentMeta.SetNamespace(ctx.Object.GetNamespace())
		currentMeta.SetLabels(ctx.Object.GetLabels())
		currentMeta.SetAnnotations(ctx.Object.GetAnnotations())
		currentMeta.SetFinalizers(ctx.Object.GetFinalizers())

		cleanMeta := r.apiType.DeepCopyObject().(client.Object)
		cleanMeta.SetName(cleanObj.GetName())
		cleanMeta.SetNamespace(cleanObj.GetNamespace())
		cleanMeta.SetLabels(cleanObj.GetLabels())
		cleanMeta.SetAnnotations(cleanObj.GetAnnotations())
		cleanMeta.SetFinalizers(cleanObj.GetFinalizers())

		patchOpts := &client.PatchOptions{FieldManager: r.name}

		if err := r.client.Patch(ctx, currentMeta, client.MergeFrom(cleanMeta), patchOpts); err != nil {
			return ctrl.Result{}, fmt.Errorf("error patching metadata: %w", err)
		}
		if err := r.client.Status().Patch(ctx, ctx.Object, client.MergeFrom(cleanObj), patchOpts); err != nil {
			return ctrl.Result{}, fmt.Errorf("error patching status: %w", err)
		}
	}

	// condense all error messages into one
	log.Info("Reconciliation complete")
	return finalRes, utilerrors.NewAggregate(errs)
}

func (r *Reconciler) getControllerName() (string, error) {
	if r.name != "" {
		return r.name, nil
	}

	gvk, err := getGvk(r.apiType, r.mgr.GetScheme())
	if err != nil {
		return "", err
	}

	return strings.ToLower(gvk.Kind), nil
}

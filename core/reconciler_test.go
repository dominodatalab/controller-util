package core

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/go-logr/logr"
	"github.com/go-logr/logr/testr"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

type MockComponent struct {
	mock.Mock
}

func (m *MockComponent) Reconcile(ctx *Context) (ctrl.Result, error) {
	args := m.Called(ctx)
	return args.Get(0).(ctrl.Result), args.Error(1)
}

type MockOwnedComponent struct {
	MockComponent
}

func (m *MockOwnedComponent) Kind() client.Object {
	args := m.Called()
	return args.Get(0).(client.Object)
}

type MockFinalizerComponent struct {
	MockComponent
}

func (m *MockFinalizerComponent) Finalize(ctx *Context) (ctrl.Result, bool, error) {
	args := m.Called(ctx)
	return args.Get(0).(ctrl.Result), args.Bool(1), args.Error(2)
}

type MockInitializerComponent struct {
	MockComponent
}

func (m *MockInitializerComponent) Initialize(ctx *Context, builder *ctrl.Builder) error {
	args := m.Called(ctx, builder)
	return args.Error(0)
}

type MockManager struct {
	manager.Manager

	client   client.Client
	scheme   *runtime.Scheme
	recorder *record.FakeRecorder
	logger   logr.Logger
}

func NewMockManager(client client.Client, scheme *runtime.Scheme, logger logr.Logger) *MockManager {
	return &MockManager{
		client: client,
		scheme: scheme,
		logger: logger,
	}
}

func (m *MockManager) GetClient() client.Client {
	return m.client
}

func (m *MockManager) GetScheme() *runtime.Scheme {
	return m.scheme
}

func (m *MockManager) GetEventRecorderFor(name string) record.EventRecorder {
	if m.recorder == nil {
		m.recorder = record.NewFakeRecorder(100)
	}
	return m.recorder
}

func (m *MockManager) GetLogger() logr.Logger {
	return m.logger
}

func (m *MockManager) GetConfig() *rest.Config {
	return nil
}

func (m *MockManager) GetAPIReader() client.Reader {
	return m.client
}

func (m *MockManager) Elected() <-chan struct{} {
	c := make(chan struct{})
	close(c)
	return c
}

func (m *MockManager) Add(r manager.Runnable) error {
	return nil
}

func setupTest(t *testing.T) (client.Client, *runtime.Scheme) {
	s := runtime.NewScheme()
	require.NoError(t, scheme.AddToScheme(s))
	require.NoError(t, appsv1.AddToScheme(s))

	client := fake.NewClientBuilder().WithScheme(s).Build()
	return client, s
}

func createTestDeployment(name, namespace string) *appsv1.Deployment {
	return &appsv1.Deployment{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "apps/v1",
			Kind:       "Deployment",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:       name,
			Namespace:  namespace,
			Generation: 1,
		},
		Spec: appsv1.DeploymentSpec{
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{"app": name},
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{"app": name},
				},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Name:  "app",
							Image: "test-image:latest",
						},
					},
				},
			},
		},
	}
}

func TestReconciler_Basic(t *testing.T) {
	cl, s := setupTest(t)

	deploy := createTestDeployment("test-deployment", "default")
	require.NoError(t, cl.Create(context.Background(), deploy))

	mgr := NewMockManager(cl, s, testr.New(t))

	mockComp := new(MockComponent)
	expectedResult := ctrl.Result{Requeue: true}
	mockComp.On("Reconcile", mock.AnythingOfType("*core.Context")).Return(expectedResult, nil)

	r := NewReconciler(mgr).
		For(deploy).
		Component("test-component", mockComp).
		Named("test-reconciler")

	r.name = "test-reconciler"
	r.resourceName = "deployment"
	r.patcher = NewPatch(schema.GroupVersionKind{
		Group:   "apps",
		Version: "v1",
		Kind:    "Deployment",
	})
	r.log = mgr.GetLogger().WithName("test-reconciler")

	req := reconcile.Request{
		NamespacedName: types.NamespacedName{
			Name:      "test-deployment",
			Namespace: "default",
		},
	}

	res, err := r.Reconcile(context.Background(), req)

	assert.NoError(t, err)
	assert.Equal(t, expectedResult, res)
	mockComp.AssertExpectations(t)
}

func TestReconciler_MultipleComponents(t *testing.T) {
	cl, s := setupTest(t)

	deploy := createTestDeployment("test-deployment", "default")
	require.NoError(t, cl.Create(context.Background(), deploy))

	mgr := NewMockManager(cl, s, testr.New(t))

	mockComp1 := new(MockComponent)
	mockComp2 := new(MockComponent)

	result1 := ctrl.Result{Requeue: true}
	result2 := ctrl.Result{RequeueAfter: 5 * time.Minute}

	mockComp1.On("Reconcile", mock.AnythingOfType("*core.Context")).Return(result1, nil)
	mockComp2.On("Reconcile", mock.AnythingOfType("*core.Context")).Return(result2, nil)

	r := NewReconciler(mgr).
		For(deploy).
		Component("component1", mockComp1).
		Component("component2", mockComp2).
		Named("test-reconciler")

	r.name = "test-reconciler"
	r.resourceName = "deployment"
	r.patcher = NewPatch(schema.GroupVersionKind{
		Group:   "apps",
		Version: "v1",
		Kind:    "Deployment",
	})
	r.log = mgr.GetLogger().WithName("test-reconciler")

	req := reconcile.Request{
		NamespacedName: types.NamespacedName{
			Name:      "test-deployment",
			Namespace: "default",
		},
	}

	res, err := r.Reconcile(context.Background(), req)

	assert.NoError(t, err)
	assert.True(t, res.Requeue)
	assert.Equal(t, 5*time.Minute, res.RequeueAfter)
	mockComp1.AssertExpectations(t)
	mockComp2.AssertExpectations(t)
}

func TestReconciler_NotFound(t *testing.T) {
	cl, s := setupTest(t)

	mgr := NewMockManager(cl, s, testr.New(t))

	mockComp := new(MockComponent)

	r := NewReconciler(mgr).
		For(&appsv1.Deployment{}).
		Component("test-component", mockComp).
		Named("test-reconciler")

	r.name = "test-reconciler"
	r.resourceName = "deployment"
	r.patcher = NewPatch(schema.GroupVersionKind{
		Group:   "apps",
		Version: "v1",
		Kind:    "Deployment",
	})
	r.log = mgr.GetLogger().WithName("test-reconciler")

	req := reconcile.Request{
		NamespacedName: types.NamespacedName{
			Name:      "non-existent",
			Namespace: "default",
		},
	}

	res, err := r.Reconcile(context.Background(), req)

	assert.NoError(t, err)
	assert.Equal(t, ctrl.Result{}, res)
	mockComp.AssertNotCalled(t, "Reconcile")

	r.ReconcileNotFound()

	mockComp.On("Reconcile", mock.AnythingOfType("*core.Context")).Return(ctrl.Result{}, nil)

	res, err = r.Reconcile(context.Background(), req)

	assert.NoError(t, err)
	assert.Equal(t, ctrl.Result{}, res)
	mockComp.AssertExpectations(t)
}

func TestReconciler_SkipReconcile(t *testing.T) {
	cl, s := setupTest(t)

	deploy := createTestDeployment("test-deployment", "default")
	deploy.Annotations = map[string]string{
		SkipReconcileAnnotation: "true",
	}
	require.NoError(t, cl.Create(context.Background(), deploy))

	mgr := NewMockManager(cl, s, testr.New(t))

	mockComp := new(MockComponent)

	r := NewReconciler(mgr).
		For(deploy).
		Component("test-component", mockComp).
		Named("test-reconciler")

	r.name = "test-reconciler"
	r.resourceName = "deployment"
	r.patcher = NewPatch(schema.GroupVersionKind{
		Group:   "apps",
		Version: "v1",
		Kind:    "Deployment",
	})
	r.log = mgr.GetLogger().WithName("test-reconciler")

	req := reconcile.Request{
		NamespacedName: types.NamespacedName{
			Name:      "test-deployment",
			Namespace: "default",
		},
	}

	res, err := r.Reconcile(context.Background(), req)

	assert.NoError(t, err)
	assert.Equal(t, ctrl.Result{}, res)
	mockComp.AssertNotCalled(t, "Reconcile")
}

func TestReconciler_OwnedComponent(t *testing.T) {
	cl, s := setupTest(t)

	deploy := createTestDeployment("test-deployment", "default")
	require.NoError(t, cl.Create(context.Background(), deploy))

	mgr := NewMockManager(cl, s, testr.New(t))

	mockOwnedComp := new(MockOwnedComponent)

	svc := &corev1.Service{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "v1",
			Kind:       "Service",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-service",
			Namespace: "default",
		},
		Spec: corev1.ServiceSpec{
			Selector: map[string]string{"app": "test-deployment"},
			Ports: []corev1.ServicePort{
				{Port: 80, TargetPort: intstr.FromInt(8080)},
			},
		},
	}

	mockOwnedComp.On("Kind").Return(svc)
	mockOwnedComp.On("Reconcile", mock.AnythingOfType("*core.Context")).Return(ctrl.Result{}, nil)

	r := NewReconciler(mgr).
		For(deploy).
		Component("owned-component", mockOwnedComp).
		Named("test-reconciler")

	r.name = "test-reconciler"
	r.resourceName = "deployment"
	r.patcher = NewPatch(schema.GroupVersionKind{
		Group:   "apps",
		Version: "v1",
		Kind:    "Deployment",
	})
	r.log = mgr.GetLogger().WithName("test-reconciler")

	req := reconcile.Request{
		NamespacedName: types.NamespacedName{
			Name:      "test-deployment",
			Namespace: "default",
		},
	}

	res, err := r.Reconcile(context.Background(), req)

	assert.NoError(t, err)
	assert.Equal(t, ctrl.Result{}, res)
	mockOwnedComp.AssertExpectations(t)
}

func TestReconciler_FinalizerComponent(t *testing.T) {
	cl, s := setupTest(t)

	deploy := createTestDeployment("test-deployment", "default")
	require.NoError(t, cl.Create(context.Background(), deploy))

	mgr := NewMockManager(cl, s, testr.New(t))

	mockFinComp1 := new(MockFinalizerComponent)

	mockFinComp1.On("Reconcile", mock.AnythingOfType("*core.Context")).Return(ctrl.Result{}, nil)

	r := NewReconciler(mgr).
		For(&appsv1.Deployment{}).
		Component("finalizer-component", mockFinComp1).
		Named("test-reconciler")

	r.name = "test-reconciler"
	r.resourceName = "deployment"
	r.patcher = NewPatch(schema.GroupVersionKind{
		Group:   "apps",
		Version: "v1",
		Kind:    "Deployment",
	})
	r.log = mgr.GetLogger().WithName("test-reconciler")

	r.components[0].finalizer = mockFinComp1
	r.components[0].finalizerName = "test-reconciler.apps/finalizer-component"

	req := reconcile.Request{
		NamespacedName: types.NamespacedName{
			Name:      "test-deployment",
			Namespace: "default",
		},
	}

	res, err := r.Reconcile(context.Background(), req)

	assert.NoError(t, err)
	assert.Equal(t, ctrl.Result{}, res)
	mockFinComp1.AssertExpectations(t)

	updatedDeploy := &appsv1.Deployment{}
	err = cl.Get(context.Background(), types.NamespacedName{
		Name: "test-deployment", Namespace: "default",
	}, updatedDeploy)
	assert.NoError(t, err)
	assert.Contains(t, updatedDeploy.Finalizers, "test-reconciler.apps/finalizer-component")

}

func TestReconciler_ComponentError(t *testing.T) {
	cl, s := setupTest(t)

	deploy := createTestDeployment("test-deployment", "default")
	require.NoError(t, cl.Create(context.Background(), deploy))

	mgr := NewMockManager(cl, s, testr.New(t))

	mockComp := new(MockComponent)
	testErr := errors.New("reconcile error")
	mockComp.On("Reconcile", mock.AnythingOfType("*core.Context")).Return(ctrl.Result{}, testErr)

	r := NewReconciler(mgr).
		For(deploy).
		Component("error-component", mockComp).
		Named("test-reconciler")

	r.name = "test-reconciler"
	r.resourceName = "deployment"
	r.patcher = NewPatch(schema.GroupVersionKind{
		Group:   "apps",
		Version: "v1",
		Kind:    "Deployment",
	})
	r.log = mgr.GetLogger().WithName("test-reconciler")

	req := reconcile.Request{
		NamespacedName: types.NamespacedName{
			Name:      "test-deployment",
			Namespace: "default",
		},
	}

	res, err := r.Reconcile(context.Background(), req)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "reconcile error")
	assert.Equal(t, ctrl.Result{}, res)
	mockComp.AssertExpectations(t)
}

func TestReconciler_MultipleComponentErrors(t *testing.T) {
	cl, s := setupTest(t)

	deploy := createTestDeployment("test-deployment", "default")
	require.NoError(t, cl.Create(context.Background(), deploy))

	mgr := NewMockManager(cl, s, testr.New(t))

	mockComp1 := new(MockComponent)
	mockComp2 := new(MockComponent)

	err1 := errors.New("error from component 1")
	err2 := errors.New("error from component 2")

	mockComp1.On("Reconcile", mock.AnythingOfType("*core.Context")).Return(ctrl.Result{}, err1)
	mockComp2.On("Reconcile", mock.AnythingOfType("*core.Context")).Return(ctrl.Result{}, err2)

	r := NewReconciler(mgr).
		For(deploy).
		Component("component1", mockComp1).
		Component("component2", mockComp2).
		Named("test-reconciler")

	r.name = "test-reconciler"
	r.resourceName = "deployment"
	r.patcher = NewPatch(schema.GroupVersionKind{
		Group:   "apps",
		Version: "v1",
		Kind:    "Deployment",
	})
	r.log = mgr.GetLogger().WithName("test-reconciler")

	req := reconcile.Request{
		NamespacedName: types.NamespacedName{
			Name:      "test-deployment",
			Namespace: "default",
		},
	}

	res, err := r.Reconcile(context.Background(), req)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "error from component 1")
	assert.Contains(t, err.Error(), "error from component 2")
	assert.Equal(t, ctrl.Result{}, res)
	mockComp1.AssertExpectations(t)
	mockComp2.AssertExpectations(t)
}

func TestReconciler_InitializerComponent(t *testing.T) {
	t.Skip("Skipping initializer test since we're not using Build() in the tests")
}

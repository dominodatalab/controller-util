package metadata

import (
	"testing"

	"github.com/stretchr/testify/assert"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type mockObject struct {
	metav1.ObjectMeta
	runtime.Object
}

func (m *mockObject) DeepCopyObject() runtime.Object {
	return &mockObject{
		ObjectMeta: *m.ObjectMeta.DeepCopy(),
	}
}

func TestNewProvider(t *testing.T) {
	p := NewProvider("test-app")
	assert.Equal(t, "test-app", p.application)
	assert.Empty(t, p.creator)
	assert.Empty(t, p.manager)
	assert.Nil(t, p.version)
	assert.Nil(t, p.dynamicLabels)

	mockVersionFn := func(obj client.Object) string { return "1.0.0" }
	mockDynamicLabelsFn := func(obj client.Object) map[string]string {
		return map[string]string{"custom": "value"}
	}

	p = NewProvider("test-app",
		WithCreator("test-creator"),
		WithManager("test-manager"),
		WithVersion(mockVersionFn),
		WithDynamicLabels(mockDynamicLabelsFn),
	)

	assert.Equal(t, "test-app", p.application)
	assert.Equal(t, "test-creator", p.creator)
	assert.Equal(t, "test-manager", p.manager)
	assert.NotNil(t, p.version)
	assert.NotNil(t, p.dynamicLabels)
}

func TestProvider_InstanceName(t *testing.T) {
	p := NewProvider("test-app")
	obj := &mockObject{
		ObjectMeta: metav1.ObjectMeta{
			Name: "test-obj",
		},
	}

	name := p.InstanceName(obj, AppComponentNone)
	assert.Equal(t, "test-obj-test-app", name)

	name = p.InstanceName(obj, AppComponent("database"))
	assert.Equal(t, "test-obj-test-app-database", name)
}

func TestProvider_StandardLabels(t *testing.T) {
	mockObj := &mockObject{
		ObjectMeta: metav1.ObjectMeta{
			Name: "test-obj",
		},
	}

	t.Run("basic labels", func(t *testing.T) {
		p := NewProvider("test-app")
		labels := p.StandardLabels(mockObj, AppComponentNone, nil)

		assert.Equal(t, "test-app", labels[ApplicationNameLabelKey])
		assert.Equal(t, "test-obj", labels[ApplicationInstanceLabelKey])
		assert.NotContains(t, labels, ApplicationCreatedByLabelKey)
		assert.NotContains(t, labels, ApplicationManagedByLabelKey)
		assert.NotContains(t, labels, ApplicationVersionLabelKey)
		assert.NotContains(t, labels, ApplicationComponentLabelKey)
	})

	t.Run("with component", func(t *testing.T) {
		p := NewProvider("test-app")
		labels := p.StandardLabels(mockObj, AppComponent("database"), nil)

		assert.Equal(t, "test-app", labels[ApplicationNameLabelKey])
		assert.Equal(t, "test-obj", labels[ApplicationInstanceLabelKey])
		assert.Equal(t, "database", labels[ApplicationComponentLabelKey])
	})

	t.Run("with creator and manager", func(t *testing.T) {
		p := NewProvider("test-app",
			WithCreator("test-creator"),
			WithManager("test-manager"),
		)
		labels := p.StandardLabels(mockObj, AppComponentNone, nil)

		assert.Equal(t, "test-app", labels[ApplicationNameLabelKey])
		assert.Equal(t, "test-obj", labels[ApplicationInstanceLabelKey])
		assert.Equal(t, "test-creator", labels[ApplicationCreatedByLabelKey])
		assert.Equal(t, "test-manager", labels[ApplicationManagedByLabelKey])
	})

	t.Run("with version function", func(t *testing.T) {
		p := NewProvider("test-app",
			WithVersion(func(obj client.Object) string {
				return "1.0.0"
			}),
		)
		labels := p.StandardLabels(mockObj, AppComponentNone, nil)

		assert.Equal(t, "test-app", labels[ApplicationNameLabelKey])
		assert.Equal(t, "test-obj", labels[ApplicationInstanceLabelKey])
		assert.Equal(t, "1.0.0", labels[ApplicationVersionLabelKey])
	})

	t.Run("with dynamic labels", func(t *testing.T) {
		p := NewProvider("test-app",
			WithDynamicLabels(func(obj client.Object) map[string]string {
				return map[string]string{"custom-key": "custom-value"}
			}),
		)
		labels := p.StandardLabels(mockObj, AppComponentNone, nil)

		assert.Equal(t, "test-app", labels[ApplicationNameLabelKey])
		assert.Equal(t, "test-obj", labels[ApplicationInstanceLabelKey])
		assert.Equal(t, "custom-value", labels["custom-key"])
	})

	t.Run("with extra labels", func(t *testing.T) {
		p := NewProvider("test-app")
		extraLabels := map[string]string{
			"extra-key": "extra-value",
		}
		labels := p.StandardLabels(mockObj, AppComponentNone, extraLabels)

		assert.Equal(t, "test-app", labels[ApplicationNameLabelKey])
		assert.Equal(t, "test-obj", labels[ApplicationInstanceLabelKey])
		assert.Equal(t, "extra-value", labels["extra-key"])
	})

	t.Run("with all options", func(t *testing.T) {
		p := NewProvider("test-app",
			WithCreator("test-creator"),
			WithManager("test-manager"),
			WithVersion(func(obj client.Object) string {
				return "1.0.0"
			}),
			WithDynamicLabels(func(obj client.Object) map[string]string {
				return map[string]string{"dynamic-key": "dynamic-value"}
			}),
		)
		extraLabels := map[string]string{
			"extra-key": "extra-value",
		}
		labels := p.StandardLabels(mockObj, AppComponent("database"), extraLabels)

		assert.Equal(t, "test-app", labels[ApplicationNameLabelKey])
		assert.Equal(t, "test-obj", labels[ApplicationInstanceLabelKey])
		assert.Equal(t, "test-creator", labels[ApplicationCreatedByLabelKey])
		assert.Equal(t, "test-manager", labels[ApplicationManagedByLabelKey])
		assert.Equal(t, "1.0.0", labels[ApplicationVersionLabelKey])
		assert.Equal(t, "database", labels[ApplicationComponentLabelKey])
		assert.Equal(t, "dynamic-value", labels["dynamic-key"])
		assert.Equal(t, "extra-value", labels["extra-key"])
	})

	t.Run("extra labels override defaults", func(t *testing.T) {
		p := NewProvider("test-app")
		extraLabels := map[string]string{
			ApplicationNameLabelKey: "override-app",
		}
		labels := p.StandardLabels(mockObj, AppComponentNone, extraLabels)

		assert.Equal(t, "override-app", labels[ApplicationNameLabelKey])
	})

	t.Run("standard labels are overridden by dynamic labels", func(t *testing.T) {
		p := NewProvider("test-app",
			WithDynamicLabels(func(obj client.Object) map[string]string {
				return map[string]string{ApplicationNameLabelKey: "dynamic-app"}
			}),
		)
		labels := p.StandardLabels(mockObj, AppComponentNone, nil)

		// dynamic labels are merged into standard labels
		assert.Equal(t, "dynamic-app", labels[ApplicationNameLabelKey])
	})
}

func TestProvider_MatchLabels(t *testing.T) {
	mockObj := &mockObject{
		ObjectMeta: metav1.ObjectMeta{
			Name: "test-obj",
		},
	}

	t.Run("basic match labels", func(t *testing.T) {
		p := NewProvider("test-app")
		labels := p.MatchLabels(mockObj, AppComponentNone)

		assert.Equal(t, "test-app", labels[ApplicationNameLabelKey])
		assert.Equal(t, "test-obj", labels[ApplicationInstanceLabelKey])
		assert.NotContains(t, labels, ApplicationComponentLabelKey)
		assert.Len(t, labels, 2)
	})

	t.Run("with component", func(t *testing.T) {
		p := NewProvider("test-app")
		labels := p.MatchLabels(mockObj, AppComponent("database"))

		assert.Equal(t, "test-app", labels[ApplicationNameLabelKey])
		assert.Equal(t, "test-obj", labels[ApplicationInstanceLabelKey])
		assert.Equal(t, "database", labels[ApplicationComponentLabelKey])
		assert.Len(t, labels, 3)
	})
}

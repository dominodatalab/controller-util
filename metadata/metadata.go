package metadata

import (
	"fmt"

	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/dominodatalab/controller-util/collection"
)

const (
	// ApplicationNameLabelKey indicates the name of the application.
	ApplicationNameLabelKey = "app.kubernetes.io/name"
	// ApplicationInstanceLabelKey indicates a unique name identifying the instance of an application.
	ApplicationInstanceLabelKey = "app.kubernetes.io/instance"
	// ApplicationVersionLabelKey indicates the current version of the application.
	ApplicationVersionLabelKey = "app.kubernetes.io/version"
	// ApplicationComponentLabelKey indicates the component within the architecture of an application.
	ApplicationComponentLabelKey = "app.kubernetes.io/component"
	// ApplicationPartOfLabelKey indicates the name of a higher level application this one is part of.
	ApplicationPartOfLabelKey = "app.kubernetes.io/part-of"
	// ApplicationManagedByLabelKey indicates the tool being used to manage the operation of an application.
	ApplicationManagedByLabelKey = "app.kubernetes.io/managed-by"
	// ApplicationCreatedByLabelKey indicates the controller/user who created this resource.
	ApplicationCreatedByLabelKey = "app.kubernetes.io/created-by"
)

type AppComponent string

const AppComponentNone AppComponent = "none"

type Provider struct {
	application string
	creator     string
	manager     string

	version       func(client.Object) string
	dynamicLabels func(client.Object) map[string]string
}

type ProviderOpt func(p *Provider)

func WithCreator(creator string) ProviderOpt {
	return func(p *Provider) {
		p.creator = creator
	}
}

func WithManager(manager string) ProviderOpt {
	return func(p *Provider) {
		p.manager = manager
	}
}

func WithVersion(fn func(client.Object) string) ProviderOpt {
	return func(p *Provider) {
		p.version = fn
	}
}

func WithDynamicLabels(fn func(client.Object) map[string]string) ProviderOpt {
	return func(p *Provider) {
		p.dynamicLabels = fn
	}
}

func NewProvider(name string, opts ...ProviderOpt) *Provider {
	p := &Provider{application: name}
	for _, opt := range opts {
		opt(p)
	}

	return p
}

func (p *Provider) InstanceName(obj client.Object, ac AppComponent) string {
	if ac == AppComponentNone {
		return fmt.Sprintf("%s-%s", obj.GetName(), p.application)
	}

	return fmt.Sprintf("%s-%s-%s", obj.GetName(), p.application, ac)
}

func (p *Provider) StandardLabels(obj client.Object, ac AppComponent, extra map[string]string) map[string]string {
	labels := map[string]string{
		ApplicationNameLabelKey:     p.application,
		ApplicationInstanceLabelKey: obj.GetName(),
	}

	if p.creator != "" {
		labels[ApplicationCreatedByLabelKey] = p.creator
	}
	if p.manager != "" {
		labels[ApplicationManagedByLabelKey] = p.manager
	}
	if p.version != nil {
		labels[ApplicationVersionLabelKey] = p.version(obj)
	}
	if p.dynamicLabels != nil {
		labels = collection.MergeStringMaps(p.dynamicLabels(obj), labels)
	}
	if ac != AppComponentNone {
		labels[ApplicationComponentLabelKey] = string(ac)
	}
	if extra != nil {
		labels = collection.MergeStringMaps(extra, labels)
	}

	return labels
}

func (p *Provider) MatchLabels(obj client.Object, ac AppComponent) map[string]string {
	labels := map[string]string{
		ApplicationNameLabelKey:     p.application,
		ApplicationInstanceLabelKey: obj.GetName(),
	}

	if ac != AppComponentNone {
		labels[ApplicationComponentLabelKey] = string(ac)
	}

	return labels
}

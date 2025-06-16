package konflux

import (
	"context"
	"fmt"

	"github.com/jordigilh/korn/internal"
	applicationapiv1alpha1 "github.com/konflux-ci/application-api/api/v1alpha1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func ListComponentWithlabels(labels client.MatchingLabels) ([]applicationapiv1alpha1.Component, error) {

	list := applicationapiv1alpha1.ComponentList{}
	err := internal.KubeClient.List(context.TODO(), &list, &client.ListOptions{Namespace: internal.Namespace}, labels)
	if err != nil {
		return nil, err
	}
	return list.Items, nil

}

func ListComponents() ([]applicationapiv1alpha1.Component, error) {
	return ListComponentsWithMatchingLabels(nil)
}

func ListComponentsWithMatchingLabels(labels client.MatchingLabels) ([]applicationapiv1alpha1.Component, error) {
	list := applicationapiv1alpha1.ComponentList{}
	err := internal.KubeClient.List(context.TODO(), &list, &client.ListOptions{Namespace: internal.Namespace}, labels)
	if err != nil {
		return nil, err
	}
	ret := []applicationapiv1alpha1.Component{}
	if len(ApplicationName) == 0 {
		return list.Items, nil
	}
	for _, c := range list.Items {
		if c.Spec.Application == ApplicationName {
			ret = append(ret, c)
		}
	}
	return ret, nil

}

func GetComponent() (*applicationapiv1alpha1.Component, error) {
	component := applicationapiv1alpha1.Component{}
	err := internal.KubeClient.Get(context.TODO(), types.NamespacedName{Namespace: internal.Namespace, Name: ComponentName}, &component)
	if err != nil {
		if errors.IsNotFound(err) {
			return nil, fmt.Errorf("component %s not found in namespace %s", ComponentName, internal.Namespace)
		}
		return nil, err
	}

	return &component, nil

}

const (
	ComponentTypeLabel      = "korn.redhat.io/component"
	ApplicationTypeLabel    = "korn.redhat.io/application"
	releaseEnvironmentLabel = "korn.redhat.io/environment"
	bundleReferenceLabel    = "korn.redhat.io/bundle-label"

	componentBundleType         = "bundle"
	releaseEnvironmentStageType = "staging"

	operatorApplicationType = "operator"
	fbcApplicationType      = "fbc"
)

func GetBundleComponentForVersion() (*applicationapiv1alpha1.Component, error) {
	l, err := ListComponentsWithMatchingLabels(client.MatchingLabels{ComponentTypeLabel: componentBundleType})
	if err != nil {
		return nil, err
	}
	if len(l) == 0 {
		return nil, fmt.Errorf("no bundle component found for application %s/%s with labels %s=bundle", internal.Namespace, ApplicationName, ComponentTypeLabel)
	}
	var comps []applicationapiv1alpha1.Component
	for _, c := range l {
		// filter out the ones that belong to this app
		if c.Spec.Application == ApplicationName {
			comps = append(comps, c)
		}
	}
	if len(comps) > 1 {
		return nil, fmt.Errorf("more than one bundle component found for application %s/%s: %+v", internal.Namespace, ApplicationName, comps)
	}
	return &comps[0], nil
}

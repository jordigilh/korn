package konflux

import (
	"context"
	"fmt"

	applicationapiv1alpha1 "github.com/konflux-ci/application-api/api/v1alpha1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func (k Korn) ListComponentWithlabels(labels client.MatchingLabels) ([]applicationapiv1alpha1.Component, error) {

	list := applicationapiv1alpha1.ComponentList{}
	err := k.KubeClient.List(context.TODO(), &list, &client.ListOptions{Namespace: k.Namespace}, labels)
	if err != nil {
		return nil, err
	}
	return list.Items, nil

}

func (k Korn) ListComponents() ([]applicationapiv1alpha1.Component, error) {
	return k.ListComponentsWithMatchingLabels(nil)
}

func (k Korn) ListComponentsWithMatchingLabels(labels client.MatchingLabels) ([]applicationapiv1alpha1.Component, error) {
	list := applicationapiv1alpha1.ComponentList{}
	err := k.KubeClient.List(context.TODO(), &list, &client.ListOptions{Namespace: k.Namespace}, labels)
	if err != nil {
		return nil, err
	}
	ret := []applicationapiv1alpha1.Component{}
	if len(k.ApplicationName) == 0 {
		return list.Items, nil
	}
	for _, c := range list.Items {
		if c.Spec.Application == k.ApplicationName {
			ret = append(ret, c)
		}
	}
	return ret, nil

}

func (k Korn) GetComponent() (*applicationapiv1alpha1.Component, error) {
	component := applicationapiv1alpha1.Component{}
	err := k.KubeClient.Get(context.TODO(), types.NamespacedName{Namespace: k.Namespace, Name: k.ComponentName}, &component)
	if err != nil {
		if errors.IsNotFound(err) {
			return nil, fmt.Errorf("component %s not found in namespace %s", k.ComponentName, k.Namespace)
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

func (k Korn) GetBundleComponentForVersion() (*applicationapiv1alpha1.Component, error) {
	l, err := k.ListComponentsWithMatchingLabels(client.MatchingLabels{ComponentTypeLabel: componentBundleType})
	if err != nil {
		return nil, err
	}
	if len(l) == 0 {
		return nil, fmt.Errorf("no bundle component found for application %s/%s with labels %s=bundle", k.Namespace, k.ApplicationName, ComponentTypeLabel)
	}
	var comps []applicationapiv1alpha1.Component
	for _, c := range l {
		// filter out the ones that belong to this app
		if c.Spec.Application == k.ApplicationName {
			comps = append(comps, c)
		}
	}
	if len(comps) > 1 {
		return nil, fmt.Errorf("more than one bundle component found for application %s/%s: %+v", k.Namespace, k.ApplicationName, comps)
	}
	return &comps[0], nil
}

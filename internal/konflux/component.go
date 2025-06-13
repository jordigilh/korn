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

func ListComponentWithlabels(namespace, applicationName string, labels client.MatchingLabels) ([]applicationapiv1alpha1.Component, error) {
	kcli, err := internal.GetClient()
	if err != nil {
		panic(err)
	}

	list := applicationapiv1alpha1.ComponentList{}
	err = kcli.List(context.TODO(), &list, &client.ListOptions{Namespace: namespace}, labels)
	if err != nil {
		return nil, err
	}
	return list.Items, nil

}

func ListComponents(namespace, applicationName string, labels client.MatchingLabels) ([]applicationapiv1alpha1.Component, error) {
	kcli, err := internal.GetClient()
	if err != nil {
		panic(err)
	}

	list := applicationapiv1alpha1.ComponentList{}
	err = kcli.List(context.TODO(), &list, &client.ListOptions{Namespace: namespace}, labels)
	if err != nil {
		return nil, err
	}
	return list.Items, nil

}

func GetComponent(componentName, namespace string) (*applicationapiv1alpha1.Component, error) {
	kcli, err := internal.GetClient()
	if err != nil {
		panic(err)
	}

	component := applicationapiv1alpha1.Component{}
	err = kcli.Get(context.TODO(), types.NamespacedName{Namespace: namespace, Name: componentName}, &component)
	if err != nil {
		if errors.IsNotFound(err) {
			return nil, fmt.Errorf("component %s not found in namespace %s", componentName, namespace)
		}
		return nil, err
	}

	return &component, nil

}

const (
	componentTypeLabel       = "component.type"
	componentVersionLabel    = "component.version"
	componentBundleTypeLabel = "bundle"
)

func GetBundleForVersion(namespace, appName, version string) (*applicationapiv1alpha1.Component, error) {
	l, err := ListComponents(namespace, appName, client.MatchingLabels{componentTypeLabel: componentBundleTypeLabel, componentVersionLabel: fmt.Sprintf("v%s", version)})
	if err != nil {
		return nil, err
	}
	if len(l) == 0 {
		return nil, fmt.Errorf("no bundle component found for application %s/%s with version %s", namespace, appName, version)
	}
	if len(l) > 1 {
		return nil, fmt.Errorf("more than 1 bundle component found for application %s/%s with version %s", namespace, appName, version)
	}
	return &l[0], nil
}

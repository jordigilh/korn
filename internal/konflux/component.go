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

func ListComponents(namespace, applicationName string) ([]applicationapiv1alpha1.Component, error) {
	return ListComponentsWithMatchingLabels(namespace, applicationName, nil)
}

func ListComponentsWithMatchingLabels(namespace, applicationName string, labels client.MatchingLabels) ([]applicationapiv1alpha1.Component, error) {
	kcli, err := internal.GetClient()
	if err != nil {
		panic(err)
	}

	list := applicationapiv1alpha1.ComponentList{}
	err = kcli.List(context.TODO(), &list, &client.ListOptions{Namespace: namespace}, labels)
	if err != nil {
		return nil, err
	}
	ret := []applicationapiv1alpha1.Component{}
	for _, c := range list.Items {
		if c.Spec.Application == applicationName {
			ret = append(ret, c)
		}
	}
	return ret, nil

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
	componentTypeLabel      = "korn.redhat.io/component"
	applicationTypeLabel    = "korn.redhat.io/application"
	versionLabel            = "korn.redhat.io/version"
	releaseEnvironmentLabel = "korn.redhat.io/environment"

	componentBundleType         = "bundle"
	releaseEnvironmentStageType = "staging"

	operatorApplicationType = "operator"
	fbcApplicationType      = "fbc"
)

func GetBundleForVersion(namespace, appName, version string) (*applicationapiv1alpha1.Component, error) {
	l, err := ListComponentsWithMatchingLabels(namespace, appName, client.MatchingLabels{componentTypeLabel: componentBundleType, versionLabel: version})
	if err != nil {
		return nil, err
	}
	if len(l) == 0 {
		return nil, fmt.Errorf("no bundle component found for application %s/%s with labels %s=bundle and %s=%s?", namespace, appName, componentTypeLabel, versionLabel, version)
	}
	if len(l) > 1 {
		return nil, fmt.Errorf("more than 1 bundle component found for application %s/%s with version %s: %+v", namespace, appName, version, l)
	}
	return &l[0], nil
}

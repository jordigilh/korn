package konflux

import (
	"context"
	"fmt"

	applicationapiv1alpha1 "github.com/konflux-ci/application-api/api/v1alpha1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func (k Korn) ListApplications() (*applicationapiv1alpha1.ApplicationList, error) {

	list := applicationapiv1alpha1.ApplicationList{}
	err := k.KubeClient.List(context.TODO(), &list, &client.ListOptions{Namespace: k.Namespace})
	if err != nil {
		return nil, err
	}
	return &list, nil
}

func (k Korn) GetApplication() (*applicationapiv1alpha1.Application, error) {

	app := applicationapiv1alpha1.Application{}
	err := k.KubeClient.Get(context.TODO(), types.NamespacedName{Namespace: k.Namespace, Name: k.ApplicationName}, &app)
	if err != nil {
		if errors.IsNotFound(err) {
			return nil, fmt.Errorf("application %s not found in namespace %s", k.ApplicationName, k.Namespace)
		}
		return nil, err
	}

	return &app, nil

}

func (k Korn) GetApplicationType() (string, error) {
	app, err := k.GetApplication()
	if err != nil {
		return "", err
	}
	appType, ok := app.ObjectMeta.Labels[ApplicationTypeLabel]
	if !ok {
		return "", fmt.Errorf("unable to determine application type: application %s/%s does not contain label %s", k.Namespace, k.ApplicationName, ApplicationTypeLabel)
	}
	return appType, nil
}

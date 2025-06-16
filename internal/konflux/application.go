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

func ListApplications() ([]applicationapiv1alpha1.Application, error) {

	list := applicationapiv1alpha1.ApplicationList{}
	err := internal.KubeClient.List(context.TODO(), &list, &client.ListOptions{Namespace: internal.Namespace})
	if err != nil {
		return nil, err
	}
	return list.Items, nil
}

func GetApplication() (*applicationapiv1alpha1.Application, error) {

	app := applicationapiv1alpha1.Application{}
	err := internal.KubeClient.Get(context.TODO(), types.NamespacedName{Namespace: internal.Namespace, Name: ApplicationName}, &app)
	if err != nil {
		if errors.IsNotFound(err) {
			return nil, fmt.Errorf("application %s not found in namespace %s", internal.Namespace, ApplicationName)
		}
		return nil, err
	}

	return &app, nil

}

func GetApplicationType() (string, error) {
	app, err := GetApplication()
	if err != nil {
		return "", err
	}
	appType, ok := app.ObjectMeta.Labels[ApplicationTypeLabel]
	if !ok {
		return "", fmt.Errorf("unable to determine application type: application %s/%s does not contain label %s", internal.Namespace, ApplicationName, ApplicationTypeLabel)
	}
	return appType, nil
}

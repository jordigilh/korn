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

func ListApplications(namespace string) ([]string, error) {
	kcli, err := internal.GetClient()
	if err != nil {
		panic(err)
	}

	list := applicationapiv1alpha1.ApplicationList{}
	err = kcli.List(context.TODO(), &list, &client.ListOptions{Namespace: namespace})
	if err != nil {
		return nil, err
	}
	var ret []string
	for _, v := range list.Items {
		ret = append(ret, v.Name)
	}
	return ret, nil

}

func GetApplication(namespace, application string) (*applicationapiv1alpha1.Application, error) {
	kcli, err := internal.GetClient()
	if err != nil {
		return nil, err
	}

	app := applicationapiv1alpha1.Application{}
	err = kcli.Get(context.TODO(), types.NamespacedName{Namespace: namespace, Name: application}, &app)
	if err != nil {
		if errors.IsNotFound(err) {
			return nil, fmt.Errorf("application %s not found in namespace %s", namespace, application)
		}
		return nil, err
	}

	return &app, nil

}

func GetApplicationTypeVersion(namespace, application string) (string, string, error) {
	app, err := GetApplication(namespace, application)
	if err != nil {
		return "", "", err
	}
	appType, ok := app.ObjectMeta.Labels[applicationTypeLabel]
	if !ok {
		return "", "", fmt.Errorf("unable to determine application type: application %s/%s does not contain label %s", namespace, application, applicationTypeLabel)
	}
	version, ok := app.ObjectMeta.Labels[versionLabel]
	if !ok {
		return "", "", fmt.Errorf("unable to determine version: application %s/%s does not contain label %s", namespace, application, versionLabel)
	}
	return appType, version, nil
}

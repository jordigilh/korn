package konflux

import (
	"context"
	"fmt"

	releaseapiv1alpha1 "github.com/konflux-ci/release-service/api/v1alpha1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func (k Korn) ListReleasePlans() ([]releaseapiv1alpha1.ReleasePlan, error) {

	list := releaseapiv1alpha1.ReleasePlanList{}
	err := k.KubeClient.List(context.TODO(), &list, &client.ListOptions{Namespace: k.Namespace})
	if err != nil {
		return nil, err
	}
	ret := []releaseapiv1alpha1.ReleasePlan{}
	for _, c := range list.Items {
		if k.ApplicationName == "" || c.Spec.Application == k.ApplicationName {
			ret = append(ret, c)
		}
	}
	return ret, nil
}

func (k Korn) GetReleasePlan() (*releaseapiv1alpha1.ReleasePlan, error) {

	rp := releaseapiv1alpha1.ReleasePlan{}
	err := k.KubeClient.Get(context.TODO(), types.NamespacedName{Namespace: k.Namespace, Name: k.ReleasePlanName}, &rp)
	if err != nil {
		if errors.IsNotFound(err) {
			return nil, fmt.Errorf("ReleasePlan %s not found in namespace %s", k.ReleasePlanName, k.Namespace)
		}
		return nil, err
	}

	return &rp, nil

}

func (k Korn) getReleasePlanForEnvWithVersion(environment string) (*releaseapiv1alpha1.ReleasePlan, error) {
	l := releaseapiv1alpha1.ReleasePlanList{}
	labels := client.MatchingLabels{EnvironmentLabel: environment}
	err := k.KubeClient.List(context.Background(), &l, &client.ListOptions{Namespace: k.Namespace}, &labels)
	if err != nil {
		return nil, err
	}
	for _, v := range l.Items {
		if v.Spec.Application == k.ApplicationName {
			return &v, nil
		}
	}

	return nil, fmt.Errorf("no release plan found for application %s/%s with labels %s=%s", k.Namespace, k.ApplicationName, EnvironmentLabel, environment)
}

// func getReleasePlanAdmission(namespace, application, environment, version string) (*releaseapiv1alpha1.ReleasePlanAdmission, error) {
// 	rp, err := getReleasePlanForEnvWithVersion(namespace, application, environment, version)
// 	if err != nil {
// 		return nil, err
// 	}
// 	if !rp.Status.ReleasePlanAdmission.Active {
// 		return nil, fmt.Errorf("no active Release Plan Admission available for Release Plan %s/%s", rp.Namespace, rp.Name)
// 	}
// 	kcli, err := internal.GetClient()
// 	if err != nil {
// 		return nil, err
// 	}
// 	rpa := releaseapiv1alpha1.ReleasePlanAdmission{}
// 	rpaNamespace, rpaName, err := cache.SplitMetaNamespaceKey(rp.Status.ReleasePlanAdmission.Name)
// 	if err != nil {
// 		return nil, err
// 	}
// 	err = kcli.Get(context.Background(), types.NamespacedName{Namespace: rpaNamespace, Name: rpaName}, &rpa, &client.GetOptions{})
// 	if err != nil {
// 		return nil, err
// 	}
// 	return &rpa, nil
// }

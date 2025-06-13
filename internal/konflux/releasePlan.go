package konflux

import (
	"context"
	"fmt"

	"github.com/jordigilh/korn/internal"
	releaseapiv1alpha1 "github.com/konflux-ci/release-service/api/v1alpha1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func getReleasePlanForEnvWithVersion(namespace, application, environment, version string) (*releaseapiv1alpha1.ReleasePlan, error) {
	l := releaseapiv1alpha1.ReleasePlanList{}
	cli, err := internal.GetClient()
	if err != nil {
		return nil, err
	}
	labels := client.MatchingLabels{releaseEnvironmentLabel: environment, versionLabel: version}
	err = cli.List(context.Background(), &l, &client.ListOptions{Namespace: namespace}, &labels)
	if err != nil {
		return nil, err
	}
	for _, v := range l.Items {
		if v.Spec.Application == application {
			return &v, nil
		}
	}

	return nil, fmt.Errorf("no release plan found for application %s/%s with labels %s=%s and %s=%s", namespace, application, releaseEnvironmentLabel, environment, versionLabel, version)
}

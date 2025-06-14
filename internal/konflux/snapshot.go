package konflux

import (
	"context"
	"fmt"
	"sort"
	"strings"

	"github.com/jordigilh/korn/internal"
	"k8s.io/apimachinery/pkg/api/errors"

	applicationapiv1alpha1 "github.com/konflux-ci/application-api/api/v1alpha1"
	"github.com/sirupsen/logrus"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func ListSnapshots(namespace, applicationName string) ([]applicationapiv1alpha1.Snapshot, error) {
	kcli, err := internal.GetClient()
	if err != nil {
		panic(err)
	}
	appType, version, err := GetApplicationTypeVersion(namespace, applicationName)
	if err != nil {
		return nil, err
	}
	list := applicationapiv1alpha1.SnapshotList{}
	var comp *applicationapiv1alpha1.Component
	if appType == "operator" {
		comp, err = GetBundleComponentForVersion(namespace, applicationName, version)
		if err != nil {
			return nil, err
		}
	} else if appType == "fbc" {
		// Get the first and only component
		comps, err := ListComponents(namespace, applicationName)
		if err != nil {
			return nil, err
		}
		if len(comps) == 0 {
			return nil, fmt.Errorf("application %s/%s does not have any component associated", namespace, applicationName)
		}
		if len(comps) > 1 {
			return nil, fmt.Errorf("application %s/%s of type FBC can only have 1 component per Konflux recommendation ", namespace, applicationName)
		}
		comp = &comps[0]
	}
	err = kcli.List(context.TODO(), &list, &client.ListOptions{Namespace: namespace}, &client.MatchingLabels{"pac.test.appstudio.openshift.io/event-type": "push", "appstudio.openshift.io/component": comp.Name})
	if err != nil {
		return nil, err
	}
	sort.Slice(list.Items,
		func(i, j int) bool {
			return list.Items[j].ObjectMeta.CreationTimestamp.Before(&list.Items[i].ObjectMeta.CreationTimestamp)
		})
	for _, v := range list.Items {
		var relStatus string
		for _, c := range v.Status.Conditions {
			if c.Type == "AppStudioTestSucceeded" {
				relStatus = c.Reason
				break
			}
		}
		logrus.Debugf("Name: %s\tStatus: %s\tComponent: %s\tSHA title: %s\n", v.Name, relStatus, v.Labels["appstudio.openshift.io/component"], v.Annotations["pac.test.appstudio.openshift.io/sha-title"])
	}
	return list.Items, nil
}

func ListSnapshotCandidatesForRelease(namespace, applicationName string) (*applicationapiv1alpha1.Snapshot, error) {
	releasesForVersion, err := ListSuccessfulReleases(namespace, applicationName)
	if err != nil {
		return nil, err
	}
	var lastSnapshot string
	if len(releasesForVersion) > 0 {
		// Copy the last successful snapshot as the cutoff version
		lastSnapshot = releasesForVersion[0].Spec.Snapshot
	}
	list, err := ListSnapshots(namespace, applicationName)
	if err != nil {
		return nil, err
	}
	appType, version, err := GetApplicationTypeVersion(namespace, applicationName)
	if err != nil {
		return nil, err
	}
	var comp *applicationapiv1alpha1.Component
	if appType == "operator" {
		comp, err = GetBundleComponentForVersion(namespace, applicationName, version)
		if err != nil {
			return nil, err
		}
	} else if appType == "fbc" {
		// Get the first and only component
		comps, err := ListComponents(namespace, applicationName)
		if err != nil {
			return nil, err
		}
		if len(comps) == 0 {
			return nil, fmt.Errorf("application %s/%s does not have any component associated", namespace, applicationName)
		}
		if len(comps) > 1 {
			return nil, fmt.Errorf("application %s/%s of type FBC can only have 1 component per Konflux recommendation ", namespace, applicationName)
		}
		comp = &comps[0]
	}
	for _, v := range list {
		if v.Name == lastSnapshot {
			break
		}
		valid, err := validateSnapshotCandicacy(comp.Name, version, v)
		if err != nil {
			return nil, err
		}
		if valid {
			return &v, nil
		}
	}
	return nil, fmt.Errorf("no new valid snapshot candidates found for bundle %s/%s with version %s after the one used for the last release %s", namespace, comp.Name, version, lastSnapshot)
}

func hasSnapshotCompletedSuccessfully(snapshot applicationapiv1alpha1.Snapshot) bool {
	for _, v := range snapshot.Status.Conditions {
		if v.Type == "AppStudioIntegrationStatus" && v.Reason == "Finished" {
			return true
		}
	}
	return false
}

func GetComponentPullspecFromSnapshot(snapshot applicationapiv1alpha1.Snapshot, componentName string) (string, error) {
	specs := map[string]string{}
	for _, c := range snapshot.Spec.Components {
		specs[c.Name] = c.ContainerImage
	}
	imagePullSpec, ok := specs[componentName]
	if !ok {
		return "", fmt.Errorf("component reference %s in snapshot %s not found", componentName, snapshot.Name)
	}
	return imagePullSpec, nil
}

func validateSnapshotCandicacy(bundleName, version string, snapshot applicationapiv1alpha1.Snapshot) (bool, error) {
	specs := map[string]string{}
	if !hasSnapshotCompletedSuccessfully(snapshot) {
		logrus.Debugf("snapshot %s has not finished running yet, discarding", snapshot.Name)
		return false, nil
	}
	bundleSpec, err := GetComponentPullspecFromSnapshot(snapshot, bundleName)
	if err != nil {
		return false, err
	}

	bundleData, err := internal.GetImageData(bundleSpec)
	if err != nil {
		return false, err
	}
	for k, v := range specs {
		if k == bundleName {
			continue
		}
		labelSpec, ok := bundleData.Labels[k[:len(k)-len(version)-1]]
		if !ok {
			logrus.Infof("reference to component %s not found in bundle labels %s", k, bundleSpec)
			return false, nil
		}
		// masage v and labelSpec to only compare the sha256 since the host and path will probably be different
		if labelSpec[strings.LastIndex(labelSpec, "@sha256:"):] != v[strings.LastIndex(v, "@sha256:"):] {
			logrus.Infof("component %s pullspec mismatch in bundle %s, snapshot is not a candidate for release", k, bundleName)
			return false, nil
		}
		componentData, err := internal.GetImageData(bundleSpec)
		if err != nil {
			return false, err
		}
		if componentData.Labels["version"] != bundleData.Labels["version"] {
			logrus.Infof("component %s and bundle %s version mismatch: component has %s and bundle has %s", k, bundleSpec, componentData.Labels["version"], bundleData.Labels["version"])
			return false, nil
		}
	}
	return true, nil
}

func GetSnapshot(snapshotName, namespace string) (string, error) {
	kcli, err := internal.GetClient()
	if err != nil {
		panic(err)
	}

	app := applicationapiv1alpha1.Snapshot{}
	err = kcli.Get(context.TODO(), types.NamespacedName{Namespace: namespace, Name: snapshotName}, &app)
	if err != nil {
		if errors.IsNotFound(err) {
			return "", fmt.Errorf("snapshot %s not found in namespace %s", snapshotName, namespace)
		}
		return "", err
	}

	return app.Name, nil

}

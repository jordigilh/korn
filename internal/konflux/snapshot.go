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

func ListSnapshots() ([]applicationapiv1alpha1.Snapshot, error) {
	list := applicationapiv1alpha1.SnapshotList{}
	labels := client.MatchingLabels{"pac.test.appstudio.openshift.io/event-type": "push"}
	if len(ApplicationName) > 0 {
		appType, err := GetApplicationType()
		if err != nil {
			return nil, err
		}
		var comp *applicationapiv1alpha1.Component
		if appType == "operator" {
			comp, err = GetBundleComponentForVersion()
			if err != nil {
				return nil, err
			}
		} else if appType == "fbc" {
			// Get the first and only component
			comps, err := ListComponents()
			if err != nil {
				return nil, err
			}
			if len(comps) == 0 {
				return nil, fmt.Errorf("application %s/%s does not have any component associated", internal.Namespace, ApplicationName)
			}
			if len(comps) > 1 {
				return nil, fmt.Errorf("application %s/%s of type FBC can only have 1 component per Konflux recommendation ", internal.Namespace, ApplicationName)
			}
			comp = &comps[0]
		}
		labels["appstudio.openshift.io/component"] = comp.Name
	}
	err := internal.KubeClient.List(context.TODO(), &list, &client.ListOptions{Namespace: internal.Namespace}, labels)
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

func GetLatestSnapshotCandidateForRelease() (*applicationapiv1alpha1.Snapshot, error) {
	if len(SnapshotName) > 0 {
		snapshot, err := GetSnapshot()
		if err != nil {
			return nil, err
		}
		return snapshot, nil
	}
	releasesForVersion, err := ListSuccessfulReleases()
	if err != nil {
		return nil, err
	}
	var lastSnapshot string
	if len(releasesForVersion) > 0 {
		// Copy the last successful snapshot as the cutoff version
		lastSnapshot = releasesForVersion[0].Spec.Snapshot
	}
	list, err := ListSnapshots()
	if err != nil {
		return nil, err
	}
	appType, err := GetApplicationType()
	if err != nil {
		return nil, err
	}
	var comp *applicationapiv1alpha1.Component
	if appType == "operator" {
		comp, err = GetBundleComponentForVersion()
		if err != nil {
			return nil, err
		}
	} else if appType == "fbc" {
		// Get the first and only component
		comps, err := ListComponents()
		if err != nil {
			return nil, err
		}
		if len(comps) == 0 {
			return nil, fmt.Errorf("application %s/%s does not have any component associated", internal.Namespace, ApplicationName)
		}
		if len(comps) > 1 {
			return nil, fmt.Errorf("application %s/%s of type FBC can only have 1 component per Konflux recommendation ", internal.Namespace, ApplicationName)
		}
		comp = &comps[0]
	}
	for _, v := range list {
		if v.Name == lastSnapshot && !ForceRelease {
			break
		}
		valid, err := validateSnapshotCandicacy(comp.Name, v)
		if err != nil {
			return nil, err
		}
		if valid {
			return &v, nil
		}
	}
	return nil, fmt.Errorf("no new valid snapshot candidates found for bundle %s/%s after the one used for the last release %s", internal.Namespace, comp.Name, lastSnapshot)
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
	for _, c := range snapshot.Spec.Components {
		if c.Name == componentName {
			return c.ContainerImage, nil
		}
	}
	return "", fmt.Errorf("component reference %s in snapshot %s not found", componentName, snapshot.Name)
}

func validateSnapshotCandicacy(bundleName string, snapshot applicationapiv1alpha1.Snapshot) (bool, error) {
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
	comps, err := ListComponents()
	if err != nil {
		return false, err
	}
	for _, c := range comps {
		if c.Name == bundleName {
			continue
		}
		compLabel, ok := c.Labels[bundleReferenceLabel]
		if !ok {
			return false, fmt.Errorf("label %s not found in component %s/%s", bundleReferenceLabel, snapshot.Namespace, snapshot.Spec.Application)
		}
		labelSpec, ok := bundleData.Labels[compLabel]
		if !ok {
			logrus.Infof("missing label %s for component %s in bundle container image %s", bundleReferenceLabel, c.Name, bundleSpec)
			return false, nil
		}
		snapshotSpec, err := GetComponentPullspecFromSnapshot(snapshot, c.Name)
		if err != nil {
			return false, err
		}
		// masage v and labelSpec to only compare the sha256 since the host and path will probably be different
		if labelSpec[strings.LastIndex(labelSpec, "@sha256:"):] != snapshotSpec[strings.LastIndex(snapshotSpec, "@sha256:"):] {
			logrus.Infof("component %s pullspec mismatch in bundle %s, snapshot is not a candidate for release", c.Name, bundleName)
			return false, nil
		}
		componentData, err := internal.GetImageData(bundleSpec)
		if err != nil {
			return false, err
		}
		if componentData.Labels["version"] != bundleData.Labels["version"] {
			logrus.Infof("component %s and bundle %s version mismatch: component has %s and bundle has %s", c.Name, bundleSpec, componentData.Labels["version"], bundleData.Labels["version"])
			return false, nil
		}
	}
	return true, nil
}

func GetSnapshot() (*applicationapiv1alpha1.Snapshot, error) {
	snapshot := applicationapiv1alpha1.Snapshot{}
	err := internal.KubeClient.Get(context.TODO(), types.NamespacedName{Namespace: internal.Namespace, Name: SnapshotName}, &snapshot)
	if err != nil {
		if errors.IsNotFound(err) {
			return nil, fmt.Errorf("snapshot %s not found in namespace %s", SnapshotName, internal.Namespace)
		}
		return nil, err
	}

	return &snapshot, nil

}

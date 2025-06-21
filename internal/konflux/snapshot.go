package konflux

import (
	"context"
	"fmt"
	"sort"
	"strings"

	"k8s.io/apimachinery/pkg/fields"

	"github.com/blang/semver/v4"
	applicationapiv1alpha1 "github.com/konflux-ci/application-api/api/v1alpha1"
	"github.com/sirupsen/logrus"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func (k Korn) ListSnapshots() ([]applicationapiv1alpha1.Snapshot, error) {
	list := applicationapiv1alpha1.SnapshotList{}
	labels := client.MatchingLabels{"pac.test.appstudio.openshift.io/event-type": "push"}
	if len(k.ApplicationName) > 0 {
		appType, err := k.GetApplicationType()
		if err != nil {
			return nil, err
		}
		var comp *applicationapiv1alpha1.Component
		switch appType {
		case "operator":
			comp, err = k.GetBundleComponentForVersion()
			if err != nil {
				return nil, err
			}
		case "fbc":
			// Get the first and only component
			comps, err := k.ListComponents()
			if err != nil {
				return nil, err
			}
			if len(comps) == 0 {
				return nil, fmt.Errorf("application %s/%s does not have any component associated", k.Namespace, k.ApplicationName)
			}
			if len(comps) > 1 {
				return nil, fmt.Errorf("application %s/%s of type FBC can only have 1 component per Konflux recommendation ", k.Namespace, k.ApplicationName)
			}
			comp = &comps[0]
		}
		labels["appstudio.openshift.io/component"] = comp.Name
	}
	err := k.KubeClient.List(context.TODO(), &list, &client.ListOptions{Namespace: k.Namespace}, labels)
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

func (k Korn) GetLatestSnapshotByVersion() (*applicationapiv1alpha1.Snapshot, error) {
	l, err := k.ListSnapshots()
	if err != nil {
		return nil, err
	}
	semVer, err := semver.ParseTolerant(k.Version)
	if err != nil {
		return nil, err
	}
	defer k.GitClient.Cleanup()
	for _, s := range l {
		v, ok, err := k.getVersionForSnapshot(s)
		if err != nil {
			return nil, err
		}
		if !ok {
			logrus.Debugf("inconsistent version for snapshot %s/%s", s.Namespace, s.Name)
			continue
		}
		if semVer.Equals(*v) {
			return &s, nil
		}
	}
	return nil, fmt.Errorf("no snapshot found for application %s/%s with version %s", k.Namespace, k.ApplicationName, k.Version)
}

func (k Korn) getVersionForSnapshot(snapshot applicationapiv1alpha1.Snapshot) (*semver.Version, bool, error) {
	var version *semver.Version

	for _, c := range snapshot.Spec.Components {
		if c.Source.GitSource == nil {
			logrus.Debugf("git source reference for component %s is missing", c.Name)
			continue
		}
		v, err := k.GitClient.GetVersion(c.Source.GitSource.URL, c.Source.GitSource.Revision)
		if err != nil {
			return nil, false, fmt.Errorf("failed to fetch file content: %v", err)
		}
		if version == nil {
			version = v
		} else if !version.Equals(*v) {
			return nil, false, nil
		}
	}
	return version, true, nil
}

func (k Korn) GetSnapshotCandidateForRelease() (*applicationapiv1alpha1.Snapshot, error) {
	if len(k.SnapshotName) > 0 || len(k.SHA) > 0 {
		return k.GetSnapshot()
	}
	releasesForVersion, err := k.ListSuccessfulReleases()
	if err != nil {
		return nil, err
	}
	var lastSnapshot *applicationapiv1alpha1.Snapshot
	if len(releasesForVersion) > 0 {
		// Copy the last successful snapshot as the cutoff version
		k.SnapshotName = releasesForVersion[0].Spec.Snapshot
		lastSnapshot, err = k.GetSnapshot()
		if err != nil {
			return nil, err
		}
	}
	list, err := k.ListSnapshots()
	if err != nil {
		return nil, err
	}
	appType, err := k.GetApplicationType()
	if err != nil {
		return nil, err
	}
	var comp *applicationapiv1alpha1.Component
	switch appType {
	case "operator":
		comp, err = k.GetBundleComponentForVersion()
		if err != nil {
			return nil, err
		}
	case "fbc":
		// Get the first and only component
		comps, err := k.ListComponents()
		if err != nil {
			return nil, err
		}
		if len(comps) == 0 {
			return nil, fmt.Errorf("application %s/%s does not have any component associated", k.Namespace, k.ApplicationName)
		}
		if len(comps) > 1 {
			return nil, fmt.Errorf("application %s/%s of type FBC can only have 1 component per Konflux recommendation ", k.Namespace, k.ApplicationName)
		}
		comp = &comps[0]
	}
	for _, v := range list {
		if v.Name == lastSnapshot.Name {
			break
		}
		valid, err := k.validateSnapshotCandicacy(comp.Name, v)
		if err != nil {
			return nil, err
		}
		if valid {
			return &v, nil
		}
	}
	if k.ForceRelease {
		// When force is enabled, we will at least return the last snapshot used, unless a newer one is detected. This ensures that the command
		// will always trigger a build
		return lastSnapshot, nil
	}
	return nil, fmt.Errorf("no new valid snapshot candidates found for bundle %s/%s after the one used for the last release %s", k.Namespace, comp.Name, lastSnapshot.Name)
}

func hasSnapshotCompletedSuccessfully(snapshot applicationapiv1alpha1.Snapshot) bool {
	for _, v := range snapshot.Status.Conditions {
		if v.Type == "AppStudioTestSucceeded" && v.Reason == "Finished" {
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

func (k Korn) validateSnapshotCandicacy(bundleName string, snapshot applicationapiv1alpha1.Snapshot) (bool, error) {
	if !hasSnapshotCompletedSuccessfully(snapshot) {
		logrus.Debugf("snapshot %s has not finished running yet, discarding", snapshot.Name)
		return false, nil
	}

	bundleSpec, err := GetComponentPullspecFromSnapshot(snapshot, bundleName)
	if err != nil {
		return false, err
	}

	bundleData, err := k.PodClient.GetImageData(bundleSpec)
	if err != nil {
		return false, err
	}
	comps, err := k.ListComponents()
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
		componentData, err := k.PodClient.GetImageData(bundleSpec)
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

func (k Korn) GetSnapshot() (*applicationapiv1alpha1.Snapshot, error) {
	list := applicationapiv1alpha1.SnapshotList{}
	labels := client.MatchingLabels{}
	if len(k.ApplicationName) > 0 {
		labels["appstudio.openshift.io/application"] = k.ApplicationName
	}
	if len(k.SHA) > 0 {
		labels["pac.test.appstudio.openshift.io/sha"] = k.SHA
	}
	options := client.ListOptions{Namespace: k.Namespace}
	if len(k.SnapshotName) > 0 {
		options.FieldSelector = fields.OneTermEqualSelector("metadata.name", k.SnapshotName)
	}
	err := k.KubeClient.List(context.TODO(), &list, &options, &labels)
	if err != nil {
		return nil, err
	}
	if len(list.Items) == 0 {
		return nil, fmt.Errorf("snapshot with SHA %s not found", k.SHA)
	}
	return &list.Items[0], nil

}

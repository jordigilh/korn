package konflux

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"strings"

	"k8s.io/apimachinery/pkg/fields"

	"github.com/blang/semver/v4"
	applicationapiv1alpha1 "github.com/konflux-ci/application-api/api/v1alpha1"
	"github.com/sirupsen/logrus"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

var (
	matchingLabelsPushEventType = client.MatchingLabels{"pac.test.appstudio.openshift.io/event-type": "push"}
)

func (k Korn) ListSnapshots() ([]applicationapiv1alpha1.Snapshot, error) {
	// If a version is provided, we will filter the snapshots by version
	if len(k.Version) > 0 {
		return k.GetSnapshotsByVersion()
	}
	// Otherwise, we will list all snapshots
	return k.listSnapshots()
}

func (k Korn) listSnapshots() ([]applicationapiv1alpha1.Snapshot, error) {
	list := applicationapiv1alpha1.SnapshotList{}
	labels := matchingLabelsPushEventType
	if len(k.ApplicationName) > 0 {
		comp, err := k.getComponentForRelease()
		if err != nil {
			return nil, err
		}
		labels["appstudio.openshift.io/component"] = comp.Name
	}
	logrus.Debugf("labels: %v", labels)
	logrus.Debugf("namespace: %s", k.Namespace)
	err := k.KubeClient.List(context.TODO(), &list, &client.ListOptions{Namespace: k.Namespace}, labels)
	if err != nil {
		return nil, err
	}
	logrus.Debugf("list of snapshots: %v", list.Items)
	sort.Slice(list.Items,
		func(i, j int) bool {
			return list.Items[j].ObjectMeta.CreationTimestamp.Before(&list.Items[i].ObjectMeta.CreationTimestamp)
		})
	return list.Items, nil
}

func (k Korn) GetSnapshotsByVersion() ([]applicationapiv1alpha1.Snapshot, error) {
	semVer, err := semver.ParseTolerant(k.Version)
	if err != nil {
		return nil, err
	}
	var snapshots []applicationapiv1alpha1.Snapshot
	l, err := k.listSnapshots()
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
			snapshots = append(snapshots, s)
		}
	}
	if len(snapshots) == 0 {
		return nil, fmt.Errorf("no snapshot found for application %s/%s with version %s", k.Namespace, k.ApplicationName, k.Version)
	}
	return snapshots, nil
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

// getComponentForRelease returns the component to use for the release.
// If the application is of "operator" type, the bundle component type is returned
// For FBC based applications, which are expected only to contain one component, the default component is returned
func (k Korn) getComponentForRelease() (*applicationapiv1alpha1.Component, error) {
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
	return comp, nil
}

func (k Korn) getSnapshotFromLastRelease() (*applicationapiv1alpha1.Snapshot, error) {
	var lastSnapshot *applicationapiv1alpha1.Snapshot
	releasesForVersion, err := k.ListSuccessfulReleases()
	if err != nil {
		return nil, err
	}
	if len(releasesForVersion) > 0 {
		// Copy the last successful snapshot as the cutoff version
		k.SnapshotName = releasesForVersion[0].Spec.Snapshot
		lastSnapshot, err = k.GetSnapshot()
		if err != nil {
			return nil, err
		}
	}
	return lastSnapshot, nil
}
func (k Korn) GetSnapshotCandidateForRelease() (*applicationapiv1alpha1.Snapshot, error) {
	if len(k.SnapshotName) > 0 || len(k.SHA) > 0 {
		return k.GetSnapshot()
	}
	lastSnapshot, err := k.getSnapshotFromLastRelease()
	if err != nil {
		return nil, err
	}

	comp, err := k.getComponentForRelease()
	if err != nil {
		return nil, err
	}
	list, err := k.ListSnapshots()
	if err != nil {
		return nil, err
	}
	for _, v := range list {
		if lastSnapshot != nil && v.Name == lastSnapshot.Name {
			break
		}
		valid, err := k.validateSnapshotCandidacy(comp.Name, v)
		if err != nil {
			return nil, err
		}
		if valid {
			return &v, nil
		}
	}
	if k.ForceRelease && lastSnapshot != nil {
		// When force is enabled, we will at least return the last snapshot used, unless a newer one is detected. This ensures that the command
		// will always trigger a build
		return lastSnapshot, nil
	}
	msg := fmt.Sprintf("no new valid snapshot candidates found for bundle %s/%s", comp.Namespace, comp.Name)
	if lastSnapshot != nil {
		msg += fmt.Sprintf(" after the one used for the last release %s", lastSnapshot.Name)
	}
	return nil, errors.New(msg)
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

func (k Korn) validateSnapshotCandidacy(bundleName string, snapshot applicationapiv1alpha1.Snapshot) (bool, error) {
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
		compLabel, ok := c.Labels[BundleReferenceLabel]
		if !ok {
			return false, fmt.Errorf("label %s not found in component %s/%s", BundleReferenceLabel, snapshot.Namespace, snapshot.Spec.Application)
		}
		labelSpec, ok := bundleData.Labels[compLabel]
		if !ok {
			logrus.Infof("missing label %s for component %s in bundle container image %s", BundleReferenceLabel, c.Name, bundleSpec)
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

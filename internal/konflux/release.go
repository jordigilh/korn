package konflux

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"strings"

	"github.com/blang/semver/v4"
	"github.com/jordigilh/korn/internal"

	applicationapiv1alpha1 "github.com/konflux-ci/application-api/api/v1alpha1"
	releaseapiv1alpha1 "github.com/konflux-ci/release-service/api/v1alpha1"
	"k8s.io/apimachinery/pkg/api/errors"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func ListReleases(namespace, applicationName, version string) ([]releaseapiv1alpha1.Release, error) {
	kcli, err := internal.GetClient()
	if err != nil {
		panic(err)
	}

	bundle, err := GetBundleForVersion(namespace, applicationName, version)
	if err != nil {
		return nil, err
	}
	list := releaseapiv1alpha1.ReleaseList{}
	err = kcli.List(context.TODO(), &list, &client.ListOptions{Namespace: namespace}, &client.MatchingLabels{"appstudio.openshift.io/application": applicationName, "appstudio.openshift.io/component": bundle.Name})
	if err != nil {
		return nil, err
	}
	sort.Slice(list.Items,
		func(i, j int) bool {
			return list.Items[j].ObjectMeta.CreationTimestamp.Before(&list.Items[j].ObjectMeta.CreationTimestamp)
		})
	return list.Items, nil
}

func ListSuccessfulReleases(namespace, applicationName, version string) ([]releaseapiv1alpha1.Release, error) {

	l, err := ListReleases(namespace, applicationName, version)
	if err != nil {
		return nil, err
	}
	var releases []releaseapiv1alpha1.Release
	for _, v := range l {
		for _, c := range v.Status.Conditions {
			if c.Type == "Released" && c.Reason == "Succeeded" {
				releases = append(releases, v)
			}
		}
	}
	return releases, nil
}

func GetRelease(releaseName, namespace string) (string, error) {
	kcli, err := internal.GetClient()
	if err != nil {
		panic(err)
	}

	app := releaseapiv1alpha1.Release{}
	err = kcli.Get(context.TODO(), types.NamespacedName{Namespace: namespace, Name: releaseName}, &app)
	if err != nil {
		if errors.IsNotFound(err) {
			return "", fmt.Errorf("release %s not found in namespace %s", releaseName, namespace)
		}
		return "", err
	}

	return app.Name, nil
}

type releaseNote struct {
	Type       releaseType       `json:"type"`
	Issues     map[string]string `json:"issues,omitempty"`
	CVEs       map[string]string `json:"cves,omitempty"`
	References []string          `json:"reference,omitempty"`
}

const (
	bugReleaseType      releaseType = "RHBA"
	securityReleaseType releaseType = "RHSA"
	featureReleaseType  releaseType = "RHEA"
)

type releaseType string

func getBundleVersionFromSnapshot(snapshot applicationapiv1alpha1.Snapshot, version string) (string, error) {

	bundle, err := GetBundleForVersion(snapshot.Namespace, snapshot.Spec.Application, version)
	if err != nil {
		return "", err
	}
	imgPullSpec, err := GetComponentPullspecFromSnapshot(snapshot, bundle.Name)
	bundleData, err := internal.GetImageData(imgPullSpec)
	if ver, ok := bundleData.Labels["version"]; ok {
		return ver, nil
	}
	return "", fmt.Errorf("label 'version' not found in bundle %s/%s", bundle.Namespace, bundle.Name)
}

func CreateRelease(namespace, application, version string, dryrun bool) (*releaseapiv1alpha1.Release, error) {
	candidate, err := ListSnapshotCandidatesForRelease(namespace, application, version)
	if err != nil {
		return nil, err
	}
	bundleVersion, err := getBundleVersionFromSnapshot(*candidate, version)
	if err != nil {
		return nil, err
	}
	semv, err := semver.Parse(bundleVersion)
	if err != nil {
		return nil, err
	}
	var rtype releaseType
	if semv.Patch == 0 {
		rtype = featureReleaseType
	} else {
		rtype = bugReleaseType
	}
	notes := map[string]releaseNote{
		"releaseNotes": {
			Type: rtype,
		},
	}
	bnotes, err := json.Marshal(notes)
	if err != nil {
		return nil, err
	}
	r := releaseapiv1alpha1.Release{
		ObjectMeta: v1.ObjectMeta{
			GenerateName: fmt.Sprintf("%s-%s-", application, strings.ReplaceAll(version, ".", "-")),
		},
		Spec: releaseapiv1alpha1.ReleaseSpec{
			Snapshot:    candidate.Name,
			ReleasePlan: fmt.Sprintf("%s-%s", application, strings.ReplaceAll(version, ".", "-")),
			Data: &runtime.RawExtension{
				Raw: bnotes,
			},
		},
	}
	return &r, nil
}

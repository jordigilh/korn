package konflux

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"time"

	"github.com/blang/semver/v4"
	"github.com/jordigilh/korn/internal"
	"github.com/sirupsen/logrus"

	applicationapiv1alpha1 "github.com/konflux-ci/application-api/api/v1alpha1"
	releaseapiv1alpha1 "github.com/konflux-ci/release-service/api/v1alpha1"
	"k8s.io/apimachinery/pkg/api/errors"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func ListReleases() ([]releaseapiv1alpha1.Release, error) {
	kcli, err := internal.GetClient()
	if err != nil {
		panic(err)
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

	list := releaseapiv1alpha1.ReleaseList{}
	err = kcli.List(context.TODO(), &list, &client.ListOptions{Namespace: internal.Namespace}, &client.MatchingLabels{"appstudio.openshift.io/application": ApplicationName, "appstudio.openshift.io/component": comp.Name})
	if err != nil {
		return nil, err
	}
	sort.Slice(list.Items,
		func(i, j int) bool {
			return list.Items[j].ObjectMeta.CreationTimestamp.Before(&list.Items[j].ObjectMeta.CreationTimestamp)
		})
	return list.Items, nil
}

func ListSuccessfulReleases() ([]releaseapiv1alpha1.Release, error) {

	l, err := ListReleases()
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

func getBundleVersionFromSnapshot(snapshot applicationapiv1alpha1.Snapshot) (string, error) {

	bundle, err := GetBundleComponentForVersion()
	if err != nil {
		return "", err
	}
	imgPullSpec, err := GetComponentPullspecFromSnapshot(snapshot, bundle.Name)
	if err != nil {
		return "", err
	}
	bundleData, err := internal.GetImageData(imgPullSpec)
	if err != nil {
		return "", err
	}
	if ver, ok := bundleData.Labels["version"]; ok {
		return ver, nil
	}
	return "", fmt.Errorf("label 'version' not found in bundle %s/%s", bundle.Namespace, bundle.Name)
}

func GenerateReleaseManifest(environment, rtype string) (*releaseapiv1alpha1.Release, error) {
	appType, err := GetApplicationType()
	if err != nil {
		return nil, err
	}
	if appType == operatorApplicationType {
		return generateReleaseManifestForOperator(environment, rtype)
	}
	if appType == fbcApplicationType {
		return generateReleaseManifestForFBC(environment, rtype)
	}
	return nil, fmt.Errorf("undefined application type %s for application %s/%s", appType, internal.Namespace, ApplicationName)
}

func generateReleaseManifestForFBC(environment, relType string) (*releaseapiv1alpha1.Release, error) {
	candidate, err := GetLatestSnapshotCandidateForRelease()
	if err != nil {
		return nil, err
	}
	rtype := releaseType(relType)
	rp, err := getReleasePlanForEnvWithVersion(environment)
	if err != nil {
		return nil, err
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
			GenerateName: fmt.Sprintf("%s-%s-", ApplicationName, environment),
			Namespace:    internal.Namespace,
		},
		Spec: releaseapiv1alpha1.ReleaseSpec{
			Snapshot:    candidate.Name,
			ReleasePlan: rp.Name,
			Data: &runtime.RawExtension{
				Raw: bnotes,
			},
		},
	}
	return &r, nil
}

func generateReleaseManifestForOperator(environment, relType string) (*releaseapiv1alpha1.Release, error) {
	candidate, err := GetLatestSnapshotCandidateForRelease()
	if err != nil {
		return nil, err
	}
	rtype := releaseType(relType)
	appType, err := GetApplicationType()
	if err != nil {
		return nil, err
	}
	if appType == operatorApplicationType {
		// Only fetch the release version when releasing an operator application type (bundle, etc...)
		bundleVersion, err := getBundleVersionFromSnapshot(*candidate)
		if err != nil {
			return nil, err
		}
		semv, err := semver.ParseTolerant(bundleVersion)
		if err != nil {
			return nil, err
		}
		if semv.Patch != 0 {
			rtype = bugReleaseType
		}
	}
	rp, err := getReleasePlanForEnvWithVersion(environment)
	if err != nil {
		return nil, err
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
			GenerateName: fmt.Sprintf("%s-%s-", ApplicationName, environment),
			Namespace:    internal.Namespace,
		},
		Spec: releaseapiv1alpha1.ReleaseSpec{
			Snapshot:    candidate.Name,
			ReleasePlan: rp.Name,
			Data: &runtime.RawExtension{
				Raw: bnotes,
			},
		},
	}
	return &r, nil
}

func CreateRelease(release releaseapiv1alpha1.Release) (*releaseapiv1alpha1.Release, error) {
	kcli, err := internal.GetClient()
	if err != nil {
		panic(err)
	}
	err = kcli.Create(context.Background(), &release, &client.CreateOptions{})
	if err != nil {
		return nil, err
	}
	return &release, nil
}

func WaitForReleaseToComplete(release releaseapiv1alpha1.Release) error {
	kcli, err := internal.GetClient()
	start := time.Now()
	if err != nil {
		panic(err)
	}
	timer := time.NewTimer(10 * time.Second)
	defer timer.Stop()
	for {
		if start.Add(60 * time.Minute).Before(time.Now()) {
			return fmt.Errorf("timed out while waiting for release %s/%s to finish", release.Namespace, release.Name)
		}
		err = kcli.Get(context.Background(), types.NamespacedName{Namespace: release.Namespace, Name: release.Name}, &release, &client.GetOptions{})
		if err != nil {
			return err
		}
		for _, c := range release.Status.Conditions {
			if c.Type == "Released" {
				switch c.Reason {
				case "Failed":
					return fmt.Errorf("release %s/%s failed", release.Namespace, release.Name)
				case "Succeeded":
					return nil
				case "Progressing":
					logrus.Debugf("Release %s/%s still ongoing after %s", release.Namespace, release.Name, time.Since(start).String())
				}
			}
		}
		<-timer.C
	}
}

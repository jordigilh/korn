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
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/duration"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func ListReleases() ([]releaseapiv1alpha1.Release, error) {
	labels := client.MatchingLabels{}
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
		labels["appstudio.openshift.io/application"] = ApplicationName
		labels["appstudio.openshift.io/component"] = comp.Name
	}
	list := releaseapiv1alpha1.ReleaseList{}
	err := internal.KubeClient.List(context.TODO(), &list, &client.ListOptions{Namespace: internal.Namespace}, labels)
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

func GetRelease() (*releaseapiv1alpha1.Release, error) {
	rel := releaseapiv1alpha1.Release{}
	err := internal.KubeClient.Get(context.TODO(), types.NamespacedName{Namespace: internal.Namespace, Name: ReleaseName}, &rel)
	if err != nil {
		if errors.IsNotFound(err) {
			return nil, fmt.Errorf("release %s not found in namespace %s", ReleaseName, internal.Namespace)
		}
		return nil, err
	}

	return &rel, nil
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

func GenerateReleaseManifest() (*releaseapiv1alpha1.Release, error) {
	appType, err := GetApplicationType()
	if err != nil {
		return nil, err
	}
	if appType == operatorApplicationType {
		return generateReleaseManifestForOperator()
	}
	if appType == fbcApplicationType {
		return generateReleaseManifestForFBC()
	}
	return nil, fmt.Errorf("undefined application type %s for application %s/%s", appType, internal.Namespace, ApplicationName)
}

func generateReleaseManifestForFBC() (*releaseapiv1alpha1.Release, error) {
	candidate, err := GetLatestSnapshotCandidateForRelease()
	if err != nil {
		return nil, err
	}
	rtype := releaseType(ReleaseType)
	rp, err := getReleasePlanForEnvWithVersion(EnvironmentName)
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
			GenerateName: fmt.Sprintf("%s-%s-", ApplicationName, EnvironmentName),
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

func generateReleaseManifestForOperator() (*releaseapiv1alpha1.Release, error) {
	candidate, err := GetLatestSnapshotCandidateForRelease()
	if err != nil {
		return nil, err
	}
	rtype := releaseType(ReleaseType)
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
	rp, err := getReleasePlanForEnvWithVersion(EnvironmentName)
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
			GenerateName: fmt.Sprintf("%s-%s-", ApplicationName, EnvironmentName),
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
	err := internal.KubeClient.Create(context.Background(), &release, &client.CreateOptions{})
	if err != nil {
		return nil, err
	}
	return &release, nil
}

func WaitForReleaseToComplete(release releaseapiv1alpha1.Release) error {

	start := time.Now()
	dynamicClient, err := internal.GetDynamicClient()
	if err != nil {
		return nil
	}
	watch, err := dynamicClient.Resource(schema.GroupVersionResource{
		Group:    "appstudio.redhat.com",
		Version:  "v1alpha1",
		Resource: "releases",
	}).Namespace(internal.Namespace).Watch(context.TODO(), v1.SingleObject(v1.ObjectMeta{Name: release.Name, Namespace: internal.Namespace}))
	if err != nil {
		return err
	}

	timer := time.NewTimer(time.Duration(WaitForTimeout) * time.Minute)

	go func() {
		<-timer.C
		watch.Stop()
	}()
	for event := range watch.ResultChan() {
		logrus.Debugf("Event Object Kind %+v\n", event.Object.GetObjectKind().GroupVersionKind())
		release := releaseapiv1alpha1.Release{}
		if event.Object == nil {
			return fmt.Errorf("object was deleted")
		}
		b, err := json.Marshal(event.Object)
		if err != nil {
			return err
		}
		err = json.Unmarshal(b, &release)
		if err != nil {
			return err
		}
		logrus.Debugf("[%s] Release: %s\n", event.Type, release.GetName())
		creleased := getConditionByType("Released", release.Status.Conditions)
		if creleased == nil {
			// condition not yet defined
			logrus.Debugf("Condition 'Release' not yet created for " + release.Name)
			continue
		}
		switch creleased.Reason {
		case "Failed":
			cpipeline := getConditionByType("ManagedPipelineProcessed", release.Status.Conditions)
			msg := fmt.Sprintf("release %s failed in pipeline %s", release.Name, release.Status.ManagedProcessing.PipelineRun)
			if cpipeline != nil && cpipeline.Reason == "Failed" {
				return fmt.Errorf("%s: %s", msg, cpipeline.Message)
			}
			return fmt.Errorf("%s: %s", msg, creleased.Message)
		case "Succeeded":
			adv := artifact{}
			err := json.Unmarshal(release.Status.Artifacts.Raw, &adv)
			if err != nil {
				return err
			}
			fmt.Printf("Artifacts:\n %+v\n", adv)
			fmt.Printf("RAW Artifacts:\n %+v\n", release.Status.Artifacts.Raw)
			return nil
		case "Progressing":
			logrus.Debugf("Release %s/%s still ongoing after %s", release.Namespace, release.Name, duration.HumanDuration(time.Since(start)))
		}
	}
	fmt.Printf("Timeout of %d minute(s) reached waiting for release %s/%s to complete", WaitForTimeout, release.Namespace, release.Name)
	return nil

}

func getConditionByType(reason string, conditions []v1.Condition) *v1.Condition {
	for _, c := range conditions {
		if c.Type == reason {
			return &c
		}
	}
	return nil
}

type artifact struct {
	Advisory    advisory     `json:"advisory"`
	CatalogURLS []catalogURL `json:"catalog_urls"`
}

type advisory struct {
	// Advisory URL
	InternalURL string `json:"internal_url,omitempty"`
	// Errata URL
	URL string `json:"url,omitempty"`
}

type catalogURL struct {
	Name string `json:"name"`
	URL  string `json:"url"`
}

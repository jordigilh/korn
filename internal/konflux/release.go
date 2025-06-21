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

func (k Korn) ListReleases() ([]releaseapiv1alpha1.Release, error) {
	labels := client.MatchingLabels{}
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
		labels["appstudio.openshift.io/application"] = k.ApplicationName
		labels["appstudio.openshift.io/component"] = comp.Name
	}
	list := releaseapiv1alpha1.ReleaseList{}
	err := k.KubeClient.List(context.TODO(), &list, &client.ListOptions{Namespace: k.Namespace}, labels)
	if err != nil {
		return nil, err
	}
	sort.Slice(list.Items,
		func(i, j int) bool {
			return list.Items[j].ObjectMeta.CreationTimestamp.Before(&list.Items[j].ObjectMeta.CreationTimestamp)
		})
	return list.Items, nil
}

func (k Korn) ListSuccessfulReleases() ([]releaseapiv1alpha1.Release, error) {

	l, err := k.ListReleases()
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

func (k Korn) GetRelease() (*releaseapiv1alpha1.Release, error) {
	rel := releaseapiv1alpha1.Release{}
	err := k.KubeClient.Get(context.TODO(), types.NamespacedName{Namespace: k.Namespace, Name: k.ReleaseName}, &rel)
	if err != nil {
		if errors.IsNotFound(err) {
			return nil, fmt.Errorf("release %s not found in namespace %s", k.ReleaseName, k.Namespace)
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

func (k Korn) getBundleVersionFromSnapshot(snapshot applicationapiv1alpha1.Snapshot) (string, error) {

	bundle, err := k.GetBundleComponentForVersion()
	if err != nil {
		return "", err
	}
	imgPullSpec, err := GetComponentPullspecFromSnapshot(snapshot, bundle.Name)
	if err != nil {
		return "", err
	}
	bundleData, err := k.PodClient.GetImageData(imgPullSpec)
	if err != nil {
		return "", err
	}
	if ver, ok := bundleData.Labels["version"]; ok {
		return ver, nil
	}
	return "", fmt.Errorf("label 'version' not found in bundle %s/%s", bundle.Namespace, bundle.Name)
}

func (k Korn) GenerateReleaseManifest() (*releaseapiv1alpha1.Release, error) {
	appType, err := k.GetApplicationType()
	if err != nil {
		return nil, err
	}
	if appType == operatorApplicationType {
		return k.generateReleaseManifestForOperator()
	}
	if appType == fbcApplicationType {
		return k.generateReleaseManifestForFBC()
	}
	return nil, fmt.Errorf("undefined application type %s for application %s/%s", appType, k.Namespace, k.ApplicationName)
}

func (k Korn) generateReleaseManifestForFBC() (*releaseapiv1alpha1.Release, error) {
	candidate, err := k.GetSnapshotCandidateForRelease()
	if err != nil {
		return nil, err
	}
	rtype := releaseType(k.ReleaseType)
	rp, err := k.getReleasePlanForEnvWithVersion(k.EnvironmentName)
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
			GenerateName: fmt.Sprintf("%s-%s-", k.ApplicationName, k.EnvironmentName),
			Namespace:    k.Namespace,
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

func (k Korn) generateReleaseManifestForOperator() (*releaseapiv1alpha1.Release, error) {
	candidate, err := k.GetSnapshotCandidateForRelease()
	if err != nil {
		return nil, err
	}
	rtype := releaseType(k.ReleaseType)
	appType, err := k.GetApplicationType()
	if err != nil {
		return nil, err
	}
	if appType == operatorApplicationType {
		// Only fetch the release version when releasing an operator application type (bundle, etc...)
		bundleVersion, err := k.getBundleVersionFromSnapshot(*candidate)
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
	rp, err := k.getReleasePlanForEnvWithVersion(k.EnvironmentName)
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
	gkv := releaseapiv1alpha1.SchemeBuilder.GroupVersion.WithKind("Release")

	r := releaseapiv1alpha1.Release{
		TypeMeta: v1.TypeMeta{
			Kind:       gkv.Kind,
			APIVersion: fmt.Sprintf("%s/%s", gkv.Group, gkv.Version),
		},
		ObjectMeta: v1.ObjectMeta{
			GenerateName: fmt.Sprintf("%s-%s-", k.ApplicationName, k.EnvironmentName),
			Namespace:    k.Namespace,
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

func (k Korn) CreateRelease(release releaseapiv1alpha1.Release) (*releaseapiv1alpha1.Release, error) {
	opts := client.CreateOptions{}
	if k.DryRun {
		opts.DryRun = append(opts.DryRun, "all")
	}
	err := k.KubeClient.Create(context.Background(), &release, &opts)
	if err != nil {
		return nil, err
	}
	return &release, nil
}

func (k Korn) WaitForReleaseToComplete(release releaseapiv1alpha1.Release, kubeConfigPath string) error {

	start := time.Now()
	dynamicClient, err := internal.GetDynamicClient(kubeConfigPath)
	if err != nil {
		return nil
	}
	watch, err := dynamicClient.Resource(schema.GroupVersionResource{
		Group:    "appstudio.redhat.com",
		Version:  "v1alpha1",
		Resource: "releases",
	}).Namespace(k.Namespace).Watch(context.TODO(), v1.SingleObject(v1.ObjectMeta{Name: release.Name, Namespace: k.Namespace}))
	if err != nil {
		return err
	}

	timer := time.NewTimer(time.Duration(k.WaitForTimeout) * time.Minute)

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
	fmt.Printf("Timeout of %d minute(s) reached waiting for release %s/%s to complete", k.WaitForTimeout, release.Namespace, release.Name)
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

package snapshot

import (
	"context"
	"fmt"
	"os"
	"sort"
	"strings"
	"time"

	"github.com/containers/podman/v5/pkg/bindings"
	"github.com/containers/podman/v5/pkg/bindings/images"
	"github.com/sirupsen/logrus"

	"github.com/jordigilh/korn/internal"
	"github.com/jordigilh/korn/internal/component"
	"github.com/jordigilh/korn/internal/release"
	applicationapiv1alpha1 "github.com/konflux-ci/application-api/api/v1alpha1"
	"github.com/urfave/cli/v3"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func ListSnapshots(namespace, applicationName, version string) ([]applicationapiv1alpha1.Snapshot, error) {
	kcli, err := internal.GetClient()
	if err != nil {
		panic(err)
	}

	bundle, err := component.GetBundleForVersion(namespace, applicationName, version)
	if err != nil {
		return nil, err
	}
	list := applicationapiv1alpha1.SnapshotList{}
	err = kcli.List(context.TODO(), &list, &client.ListOptions{Namespace: namespace}, &client.MatchingLabels{"pac.test.appstudio.openshift.io/event-type": "push", "appstudio.openshift.io/component": bundle.Name})
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

func ListSnapshotCandidatesForRelease(namespace, applicationName, version string) (*applicationapiv1alpha1.Snapshot, error) {
	releasesForVersion, err := release.ListSuccessfulReleases(namespace, applicationName, version)
	if err != nil {
		return nil, err
	}
	var lastSnapshot string
	if len(releasesForVersion) > 0 {
		// Copy the last successful snapshot as the cutoff version
		lastSnapshot = releasesForVersion[0].Spec.Snapshot
	}
	list, err := ListSnapshots(namespace, applicationName, version)
	if err != nil {
		return nil, err
	}
	bundle, err := component.GetBundleForVersion(namespace, applicationName, version)
	if err != nil {
		return nil, err
	}
	for _, v := range list {
		if v.Name == lastSnapshot {
			break
		}
		valid, err := validateSnapshotCandicacy(bundle.Name, version, v)
		if err != nil {
			return nil, err
		}
		if valid {
			return &v, nil
		}
	}
	return nil, fmt.Errorf("no new valid snapshot candidates found for bundle %s/%s with version %s after the one used for the last release %s", namespace, bundle.Name, version, lastSnapshot)
}

var (
	linux       = "linux"
	amd64       = "amd64"
	forceRemove = true
	quietPull   = true
)

func hasSnapshotCompletedSuccessfully(snapshot applicationapiv1alpha1.Snapshot) bool {
	for _, v := range snapshot.Status.Conditions {
		if v.Type == "AppStudioIntegrationStatus" && v.Reason == "Finished" {
			return true
		}
	}
	return false
}
func validateSnapshotCandicacy(bundleName, version string, snapshot applicationapiv1alpha1.Snapshot) (bool, error) {
	specs := map[string]string{}
	imageCleanup := []string{}
	if !hasSnapshotCompletedSuccessfully(snapshot) {
		logrus.Debugf("snapshot %s has not finished running yet, discarding", snapshot.Name)
		return false, nil
	}

	for _, c := range snapshot.Spec.Components {
		specs[c.Name] = c.ContainerImage
	}
	bundleSpec, ok := specs[bundleName]
	if !ok {
		return false, fmt.Errorf("bundle component reference %s in snapshot %s not found", bundleName, snapshot.Name)
	}
	dockerHostEnv, ok := os.LookupEnv("DOCKER_HOST")
	if !ok {
		return false, fmt.Errorf("DOCKER_HOST not defined in environment")
	}
	conn, err := bindings.NewConnection(context.Background(), dockerHostEnv)
	if err != nil {
		return false, err
	}
	// Pull the image to be inspected
	id, err := images.Pull(conn, bundleSpec, &images.PullOptions{OS: &linux, Arch: &amd64, Quiet: &quietPull})
	if err != nil {
		return false, err
	}
	defer images.Remove(conn, imageCleanup, &images.RemoveOptions{Force: &forceRemove})

	// Inspect the image's labels
	bundleData, err := images.GetImage(conn, id[0], new(images.GetOptions).WithSize(true))
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
		// Pull the image to be inspected
		id, err := images.Pull(conn, v, &images.PullOptions{OS: &linux, Arch: &amd64, Quiet: &quietPull})
		if err != nil {
			return false, err
		}
		imageCleanup = append(imageCleanup, id...)
		// Inspect the component's labels
		componentData, err := images.GetImage(conn, id[0], new(images.GetOptions).WithSize(true))
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

func GetCommand() *cli.Command {

	return &cli.Command{
		Name:                  "snapshot",
		Aliases:               []string{"snapshots"},
		Usage:                 "get snapshots",
		EnableShellCompletion: true,
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:        "namespace",
				Aliases:     []string{"n"},
				Usage:       "-namespace <target_namespace>",
				DefaultText: "Target namespace",
			},
			&cli.StringFlag{
				Name:        "application",
				Aliases:     []string{"app"},
				Usage:       "-application <application_name>",
				DefaultText: "Application where the components are derived from",
			},
			&cli.StringFlag{
				Name:        "version",
				Aliases:     []string{"v"},
				Usage:       "Example: -version 0.1",
				DefaultText: "Version",
			},
			&cli.BoolFlag{
				Name:        "candidate",
				Aliases:     []string{"c"},
				Usage:       "Example: -candidate",
				DefaultText: "Filters the snapshots that are suitable for the next release. The cutoff snapshot is the last used in a successful release",
				Value:       false,
			},
		},
		Description: "Retrieves a snapshot or the list of components. If application or version is not provided, it will list all snapshots in the namespace",
		Action: func(ctx context.Context, cmd *cli.Command) error {
			if len(cmd.Args().First()) == 0 {
				if cmd.Bool("candidate") {
					snapshot, err := ListSnapshotCandidatesForRelease(cmd.String("namespace"), cmd.String("application"), cmd.String("version"))
					if err != nil {
						return err
					}
					fmt.Printf("Candidate snapshot found with name:%s and creation date: %s\n", snapshot.Name, snapshot.CreationTimestamp.Format(time.UnixDate))
					return nil
				}
				l, err := ListSnapshots(cmd.String("namespace"), cmd.String("application"), cmd.String("version"))
				if err != nil {
					return err
				}
				for _, v := range l {
					logrus.Debugf("Name:%s\tCreation Date:%s\n", v.Name, v.CreationTimestamp.Format(time.UnixDate))
				}
				return nil
			}
			a, err := GetSnapshot(cmd.Args().First(), cmd.String("namespace"))
			if err != nil {
				return err
			}
			logrus.Debugf("%+v", a)
			return nil
		},
	}
}

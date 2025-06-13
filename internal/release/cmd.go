package release

import (
	"context"
	"fmt"
	"sort"
	"time"

	"github.com/jordigilh/korn/internal"
	"github.com/jordigilh/korn/internal/component"
	releaseapiv1alpha1 "github.com/konflux-ci/release-service/api/v1alpha1"
	"github.com/sirupsen/logrus"
	"github.com/urfave/cli/v3"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func ListReleases(namespace, applicationName, version string) ([]releaseapiv1alpha1.Release, error) {
	kcli, err := internal.GetClient()
	if err != nil {
		panic(err)
	}

	bundle, err := component.GetBundleForVersion(namespace, applicationName, version)
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

func CreateCommand() *cli.Command {
	return nil
}

func GetCommand() *cli.Command {

	return &cli.Command{
		Name:                  "release",
		Aliases:               []string{"releases"},
		Usage:                 "get releases",
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
				DefaultText: "Application where the releases are derived from",
			},
			&cli.StringFlag{
				Name:        "version",
				Aliases:     []string{"v"},
				Usage:       "Example: -version 0.1",
				DefaultText: "Version",
			},
		},
		Description: "Retrieves a release or the list of components. If application or version is not provided, it will list all snapshots in the namespace",
		Action: func(ctx context.Context, cmd *cli.Command) error {
			if len(cmd.Args().First()) == 0 {
				l, err := ListReleases(cmd.String("namespace"), cmd.String("application"), cmd.String("version"))
				if err != nil {
					return err
				}
				var relStatus string
				for _, v := range l {
					for _, c := range v.Status.Conditions {
						if c.Type == "Released" {
							relStatus = c.Reason
							break
						}
					}
					logrus.Debugf("Name:%s\tSnapshot:%s\tRelease Plan:%s\tRelease Status:%s\tAge:%s\n", v.Name, v.Spec.Snapshot, v.Spec.ReleasePlan, relStatus, time.Since(v.CreationTimestamp.Time))
				}
				return nil
			}
			a, err := GetRelease(cmd.Args().First(), cmd.String("namespace"))
			if err != nil {
				return err
			}
			logrus.Debugf("%+v", a)
			return nil
		},
	}
}

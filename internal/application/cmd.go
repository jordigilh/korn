package application

import (
	"context"
	"fmt"

	"github.com/jordigilh/korn/internal"
	applicationapiv1alpha1 "github.com/konflux-ci/application-api/api/v1alpha1"
	"github.com/sirupsen/logrus"
	"github.com/urfave/cli/v3"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func ListApplications(namespace string) ([]string, error) {
	kcli, err := internal.GetClient()
	if err != nil {
		panic(err)
	}

	list := applicationapiv1alpha1.ApplicationList{}
	err = kcli.List(context.TODO(), &list, &client.ListOptions{Namespace: namespace})
	if err != nil {
		return nil, err
	}
	var ret []string
	for _, v := range list.Items {
		ret = append(ret, v.Name)
	}
	return ret, nil

}

func GetApplication(appName, namespace string) (string, error) {
	kcli, err := internal.GetClient()
	if err != nil {
		panic(err)
	}

	app := applicationapiv1alpha1.Application{}
	err = kcli.Get(context.TODO(), types.NamespacedName{Namespace: "storage-scale-releng-tenant", Name: appName}, &app)
	if err != nil {
		if errors.IsNotFound(err) {
			return "", fmt.Errorf("application %s not found in namespace %s", appName, namespace)
		}
		return "", err
	}

	return app.Name, nil

}

func GetCommand() *cli.Command {

	return &cli.Command{
		Name:                  "application",
		Aliases:               []string{"app", "apps", "applications"},
		Usage:                 "get applications",
		EnableShellCompletion: true,
		Flags: []cli.Flag{&cli.StringFlag{
			Name:        "namespace",
			Aliases:     []string{"n"},
			Usage:       "-namespace <target_namespace>",
			DefaultText: "Target namespace",
		}},
		Description: "Retrieves the list of applications in your ",
		Action: func(ctx context.Context, cmd *cli.Command) error {
			if len(cmd.Args().First()) == 0 {
				l, err := ListApplications(cmd.String("namespace"))
				if err != nil {
					return err
				}
				logrus.Debugf("%+v", l)
				return nil
			}
			a, err := GetApplication(cmd.Args().First(), cmd.String("namespace"))
			if err != nil {
				return err
			}
			logrus.Debugf("%+v", a)
			return nil
		},
	}
}

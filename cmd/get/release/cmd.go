package release

import (
	"context"
	"fmt"
	"time"

	"github.com/jordigilh/korn/internal"
	"github.com/jordigilh/korn/internal/konflux"
	"github.com/urfave/cli/v3"
)

func GetCommand() *cli.Command {

	return &cli.Command{
		Name:    "release",
		Aliases: []string{"releases"},
		Usage:   "get releases",
		Arguments: []cli.Argument{&cli.StringArg{
			Name:        "release",
			Destination: &konflux.ReleaseName,
		}},
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:        "application",
				Aliases:     []string{"app"},
				Usage:       "-application <application_name>",
				DefaultText: "Application where the releases are derived from",
				Destination: &konflux.ApplicationName,
			},
		},
		Description: "Retrieves a release or the list of components. If application is not provided, it will list all snapshots in the namespace",
		Action: func(ctx context.Context, cmd *cli.Command) error {
			if len(konflux.ReleaseName) == 0 {
				l, err := konflux.ListReleases()
				if err != nil {
					return err
				}
				if len(l) == 0 {
					fmt.Printf("No releases found for %s/%s\n", internal.Namespace, konflux.ApplicationName)
				}
				var relStatus string
				for _, v := range l {
					for _, c := range v.Status.Conditions {
						if c.Type == "Released" {
							relStatus = c.Reason
							break
						}
					}
					fmt.Printf("Name:%s\tSnapshot:%s\tRelease Plan:%s\tRelease Status:%s\tAge:%s\n", v.Name, v.Spec.Snapshot, v.Spec.ReleasePlan, relStatus, time.Since(v.CreationTimestamp.Time))
				}
				return nil
			}
			a, err := konflux.GetRelease()
			if err != nil {
				return err
			}
			fmt.Printf("%+v", a)
			return nil
		},
	}
}

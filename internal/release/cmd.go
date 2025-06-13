package release

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/jordigilh/korn/internal/konflux"
	"github.com/urfave/cli/v3"
)

func CreateCommand() *cli.Command {
	return &cli.Command{
		Name:                  "release",
		Aliases:               []string{"releases"},
		Usage:                 "create releases",
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
			&cli.BoolFlag{
				Name:        "dryrun",
				Usage:       "Example: -dryrun ",
				DefaultText: "Outputs the manifest to use when creating a new release",
			},
		},
		Description: "Creates a release or the list of components. If application or version is not provided, it will list all snapshots in the namespace",
		Action: func(ctx context.Context, cmd *cli.Command) error {
			r, err := konflux.CreateRelease(cmd.String("namespace"), cmd.String("application"), cmd.String("version"), cmd.Bool("dryrun"))
			if err != nil {
				return err
			}
			if cmd.Bool("dryrun") {
				b, err := json.Marshal(r)
				if err != nil {
					return err
				}
				fmt.Printf("%s\n", string(b))
				return nil
			}
			fmt.Printf("Release created")
			return nil
		},
	}
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
				l, err := konflux.ListReleases(cmd.String("namespace"), cmd.String("application"), cmd.String("version"))
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
					fmt.Printf("Name:%s\tSnapshot:%s\tRelease Plan:%s\tRelease Status:%s\tAge:%s\n", v.Name, v.Spec.Snapshot, v.Spec.ReleasePlan, relStatus, time.Since(v.CreationTimestamp.Time))
				}
				return nil
			}
			a, err := konflux.GetRelease(cmd.Args().First(), cmd.String("namespace"))
			if err != nil {
				return err
			}
			fmt.Printf("%+v", a)
			return nil
		},
	}
}

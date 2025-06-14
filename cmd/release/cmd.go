package release

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/jordigilh/korn/internal/konflux"
	"github.com/sirupsen/logrus"
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
				Name:        "application",
				Aliases:     []string{"app"},
				Usage:       "-application <application_name>",
				DefaultText: "Application where the releases are derived from",
			},
			&cli.StringFlag{
				Name:    "environment",
				Aliases: []string{"env"},
				Usage:   "Example: -env staging",
				Validator: func(val string) error {
					if val != "staging" && val != "production" {
						return fmt.Errorf("invalid value %s: only 'staging' or 'production' supported", val)
					}
					return nil
				},
				Value:       "staging",
				DefaultText: "Captures the target environment: staging or production",
			},
			&cli.BoolFlag{
				Name:        "dryrun",
				Usage:       "Example: -dryrun ",
				Value:       false,
				DefaultText: "Outputs the manifest to use when creating a new release. This command is incompatible with the 'wait' flag",
			},
			&cli.BoolFlag{
				Name:        "wait",
				Aliases:     []string{"w"},
				Usage:       "Example: -w ",
				Value:       true,
				DefaultText: "When creating a release, this command will instruct the CLI to wait for the completion of the release pipeline and return the results. This command is incompatible with the 'dryrun' flag",
			},
			&cli.StringFlag{
				Name:        "releaseType",
				Aliases:     []string{"rt"},
				Value:       "RHEA",
				Usage:       "-rt <release type>",
				DefaultText: "Release type to use in the releaseNotes: RHEA, RHBA or RHSA. Defaults to RHEA",
				Validator: func(val string) error {
					if val != "RHEA" && val != "RHBA" && val != "RHSA" {
						return fmt.Errorf("invalid release type %s: only 'RHEA', 'RHBA' or 'RHSA' are supported", val)
					}
					return nil
				},
			},
		},
		Description: "Creates a release or the list of components. If application or version is not provided, it will list all snapshots in the namespace",
		Action: func(ctx context.Context, cmd *cli.Command) error {
			m, err := konflux.GenerateReleaseManifest(cmd.String("environment"), cmd.String("releaseType"))
			if err != nil {
				return err
			}
			if cmd.Bool("dryrun") {
				b, err := json.Marshal(m)
				if err != nil {
					return err
				}
				fmt.Printf("%s\n", string(b))
				return nil
			}
			r, err := konflux.CreateRelease(*m)
			if err != nil {
				return err
			}
			logrus.Infof("Release created %s", r.Name)
			if cmd.Bool("wait") {
				err = konflux.WaitForReleaseToComplete(*r)
				if err != nil {
					return err
				}
				fmt.Printf("Release %s/%s has completed successfully", r.Namespace, r.Name)
			}
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
				Name:        "application",
				Aliases:     []string{"app"},
				Usage:       "-application <application_name>",
				DefaultText: "Application where the releases are derived from",
			},
		},
		Description: "Retrieves a release or the list of components. If application is not provided, it will list all snapshots in the namespace",
		Action: func(ctx context.Context, cmd *cli.Command) error {
			if len(cmd.Args().First()) == 0 {
				l, err := konflux.ListReleases()
				if err != nil {
					return err
				}
				if len(l) == 0 {
					fmt.Printf("No releases found for %s/%s\n", cmd.String("namespace"), cmd.String("application"))
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

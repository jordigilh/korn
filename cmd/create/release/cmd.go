package release

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/jordigilh/korn/internal/konflux"
	"github.com/sirupsen/logrus"
	"github.com/urfave/cli/v3"
)

func CreateCommand() *cli.Command {
	return &cli.Command{
		Name:    "release",
		Aliases: []string{"releases"},
		Usage:   "create releases",
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
				Destination: &konflux.EnvironmentName,
			},
			&cli.StringFlag{
				Name:        "snapshot",
				Usage:       "-snapshot <application_name>",
				DefaultText: "Use this snapshot for the release instead of the latest candidate",
				Destination: &konflux.SnapshotName,
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
			&cli.BoolFlag{
				Name:        "force",
				Aliases:     []string{"f"},
				Usage:       "Example: -f ",
				Value:       true,
				DefaultText: "Force the creation of the release with the last candidate, even if the candidate has been used in a previous release. Useful when retrying for a failed release.",
				Destination: &konflux.ForceRelease,
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
				Destination: &konflux.ReleaseType,
			},
			&cli.IntFlag{
				Name:        "timeout",
				Aliases:     []string{"t"},
				Usage:       "-timeout timeout in minutes",
				DefaultText: "Time out in minutes for the wait for operation to complete",
				Destination: &konflux.WaitForTimeout,
				Value:       60,
			},
		},
		Description: "Creates a release or the list of components. If application or version is not provided, it will list all snapshots in the namespace",
		Action: func(ctx context.Context, cmd *cli.Command) error {
			m, err := konflux.GenerateReleaseManifest()
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

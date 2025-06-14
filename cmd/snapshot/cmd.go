package snapshot

import (
	"context"
	"fmt"
	"time"

	"github.com/jordigilh/korn/internal/konflux"
	"github.com/urfave/cli/v3"
)

func GetCommand() *cli.Command {

	return &cli.Command{
		Name:                  "snapshot",
		Aliases:               []string{"snapshots"},
		Usage:                 "get snapshots",
		EnableShellCompletion: true,
		Flags: []cli.Flag{
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
					snapshot, err := konflux.ListSnapshotCandidatesForRelease(cmd.String("namespace"), cmd.String("application"))
					if err != nil {
						return err
					}
					fmt.Printf("Candidate snapshot found with name:%s and creation date: %s\n", snapshot.Name, snapshot.CreationTimestamp.Format(time.UnixDate))
					return nil
				}
				l, err := konflux.ListSnapshots(cmd.String("namespace"), cmd.String("application"))
				if err != nil {
					return err
				}
				for _, v := range l {
					fmt.Printf("Name:%s\tCreation Date:%s\n", v.Name, v.CreationTimestamp.Format(time.UnixDate))
				}
				return nil
			}
			a, err := konflux.GetSnapshot(cmd.Args().First(), cmd.String("namespace"))
			if err != nil {
				return err
			}
			fmt.Printf("%+v", a)
			return nil
		},
	}
}

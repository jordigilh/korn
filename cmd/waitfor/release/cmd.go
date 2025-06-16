package release

import (
	"context"

	"github.com/jordigilh/korn/internal/konflux"
	"github.com/urfave/cli/v3"
)

func WaitForCommand() *cli.Command {
	return &cli.Command{
		Name:    "release",
		Aliases: []string{"releases"},
		Usage:   "waitfor release <release_name>",
		Arguments: []cli.Argument{&cli.StringArg{
			Name:        "release",
			Destination: &konflux.ReleaseName,
		}},
		Description: "Creates a release or the list of components. If application or version is not provided, it will list all snapshots in the namespace",
		Action: func(ctx context.Context, cmd *cli.Command) error {

			r, err := konflux.GetRelease()
			if err != nil {
				return err
			}
			err = konflux.WaitForReleaseToComplete(*r)
			if err != nil {
				return err
			}
			return nil
		},
	}
}

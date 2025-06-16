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
		Flags: []cli.Flag{
			&cli.IntFlag{
				Name:        "timeout",
				Aliases:     []string{"t"},
				Usage:       "-timeout timeout in minutes",
				DefaultText: "Time out in minutes for the wait for operation to complete",
				Destination: &konflux.WaitForTimeout,
				Value:       60,
			},
		},
		Arguments: []cli.Argument{&cli.StringArg{
			Name:        "release",
			Destination: &konflux.ReleaseName,
		}},
		Description: "Waits for an existing release to finish by periodically checking every 10 seconds for the status of the release until it's either Failed or Succeeeded. Timeout occurs after 60 minutes",
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

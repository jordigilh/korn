package create

import (
	"context"
	"fmt"

	"github.com/jordigilh/korn/cmd/create/release"
	"github.com/urfave/cli/v3"
)

func Command() *cli.Command {
	return &cli.Command{
		Name:  "create",
		Usage: "create release",
		Commands: []*cli.Command{
			release.CreateCommand(),
		},
		DefaultCommand: "-h",
		Action: func(ctx context.Context, cmd *cli.Command) error {
			fmt.Println("added task: ", cmd.Args().First())
			return nil
		},
	}
}

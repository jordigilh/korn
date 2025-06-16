package waitfor

import (
	"context"
	"fmt"

	"github.com/jordigilh/korn/cmd/waitfor/release"
	"github.com/urfave/cli/v3"
)

func Command() *cli.Command {
	return &cli.Command{
		Name:  "waitfor",
		Usage: "waitfor",
		Commands: []*cli.Command{
			release.WaitForCommand(),
		},
		Action: func(ctx context.Context, cmd *cli.Command) error {
			fmt.Println("added task: ", cmd.Args().First())
			return nil
		},
	}
}

package get

import (
	"context"
	"fmt"

	"github.com/jordigilh/korn/cmd/get/application"
	"github.com/jordigilh/korn/cmd/get/component"
	"github.com/jordigilh/korn/cmd/get/release"
	"github.com/jordigilh/korn/cmd/get/snapshot"
	"github.com/urfave/cli/v3"
)

func Command() *cli.Command {
	return &cli.Command{
		Name:  "get",
		Usage: "get <resources>",
		Commands: []*cli.Command{
			application.GetCommand(),
			component.GetCommand(),
			snapshot.GetCommand(),
			release.GetCommand(),
		},
		Action: func(ctx context.Context, cmd *cli.Command) error {
			fmt.Println("added task: ", cmd.Args().First())
			return nil
		},
	}
}

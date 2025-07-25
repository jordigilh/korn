package create

import (
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
	}
}

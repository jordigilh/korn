package waitfor

import (
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
	}
}

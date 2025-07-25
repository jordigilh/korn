package get

import (
	"github.com/jordigilh/korn/cmd/get/application"
	"github.com/jordigilh/korn/cmd/get/component"
	"github.com/jordigilh/korn/cmd/get/release"
	"github.com/jordigilh/korn/cmd/get/releaseplan"
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
			releaseplan.GetCommand(),
		},
	}
}

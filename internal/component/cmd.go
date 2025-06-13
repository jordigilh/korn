package component

import (
	"context"
	"fmt"

	"github.com/jordigilh/korn/internal/konflux"
	"github.com/urfave/cli/v3"
)

func GetCommand() *cli.Command {

	return &cli.Command{
		Name:                  "component",
		Aliases:               []string{"comp", "comps", "components"},
		Usage:                 "get components",
		EnableShellCompletion: true,
		Flags: []cli.Flag{&cli.StringFlag{
			Name:        "namespace",
			Aliases:     []string{"n"},
			Usage:       "-namespace <target_namespace>",
			DefaultText: "Target namespace",
		},
			&cli.StringFlag{
				Name:        "application",
				Aliases:     []string{"app"},
				Usage:       "-application <application_name>",
				DefaultText: "Application where the components are derived from",
			}},
		Description: "Retrieves a component or the list of components. If application is not provided, it will list all components in the namespace ",
		Action: func(ctx context.Context, cmd *cli.Command) error {
			if len(cmd.Args().First()) == 0 {
				l, err := konflux.ListComponents(cmd.String("namespace"), cmd.String("application"), nil)
				if err != nil {
					return err
				}
				fmt.Printf("%+v", l)
				return nil
			}
			a, err := konflux.GetComponent(cmd.Args().First(), cmd.String("application"))
			if err != nil {
				return err
			}
			fmt.Printf("%+v", a)
			return nil
		},
	}
}

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
		Arguments: []cli.Argument{&cli.StringArg{
			Name:        "component",
			Destination: &konflux.ComponentName,
		}},
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:        "application",
				Aliases:     []string{"app"},
				Usage:       "-application <application_name>",
				DefaultText: "Application where the components are derived from",
				Destination: &konflux.ApplicationName,
			}},
		Description: "Retrieves a component or the list of components. If application is not provided, it will list all components in the namespace ",
		Action: func(ctx context.Context, cmd *cli.Command) error {
			if len(konflux.ComponentName) == 0 {
				l, err := konflux.ListComponents()
				if err != nil {
					return err
				}
				for _, c := range l {
					fmt.Printf("%s\t%s\n", c.Name, c.Labels[konflux.ComponentTypeLabel])
				}
				return nil
			}
			a, err := konflux.GetComponent()
			if err != nil {
				return err
			}
			fmt.Printf("%+v", a)
			return nil
		},
	}
}

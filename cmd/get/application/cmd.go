package application

import (
	"context"
	"fmt"

	"github.com/jordigilh/korn/internal/konflux"
	"github.com/urfave/cli/v3"
)

func GetCommand() *cli.Command {

	return &cli.Command{
		Name:        "application",
		Aliases:     []string{"app", "apps", "applications"},
		Usage:       "get applications",
		Description: "Retrieves the list of applications in your ",
		Arguments: []cli.Argument{&cli.StringArg{
			Name:        "application",
			Destination: &konflux.ApplicationName,
		}},
		Action: func(ctx context.Context, cmd *cli.Command) error {
			if len(konflux.ApplicationName) == 0 {
				l, err := konflux.ListApplications()
				if err != nil {
					return err
				}
				for _, app := range l {
					fmt.Printf("%s\t%s\n", app.Name, app.Labels[konflux.ApplicationTypeLabel])
				}
				return nil
			}
			a, err := konflux.GetApplication()
			if err != nil {
				return err
			}
			fmt.Printf("%+v", a.Name)
			return nil
		},
	}
}

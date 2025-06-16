package component

import (
	"context"
	"os"
	"time"

	"github.com/jordigilh/korn/internal/konflux"
	applicationapiv1alpha1 "github.com/konflux-ci/application-api/api/v1alpha1"
	"github.com/urfave/cli/v3"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/duration"
	"k8s.io/cli-runtime/pkg/printers"
)

var (
	table = &metav1.Table{
		ColumnDefinitions: []metav1.TableColumnDefinition{
			{Name: "Name", Type: "string"},
			{Name: "Type", Type: "string"},
			{Name: "Age", Type: "string"},
		},
	}
	p = printers.NewTablePrinter(printers.PrintOptions{})
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
				print(l)
				return nil
			}
			a, err := konflux.GetComponent()
			if err != nil {
				return err
			}
			print([]applicationapiv1alpha1.Component{*a})
			return nil
		},
	}
}

func print(comps []applicationapiv1alpha1.Component) {
	rows := []metav1.TableRow{}
	for _, v := range comps {
		rows = append(rows, metav1.TableRow{Cells: []interface{}{
			v.Name,
			v.Labels[konflux.ComponentTypeLabel],
			duration.HumanDuration(time.Since(v.CreationTimestamp.Time)),
		}})
	}
	table.Rows = rows
	p.PrintObj(table, os.Stdout)
}

package component

import (
	"context"
	"os"
	"time"

	"github.com/jordigilh/korn/internal"
	"github.com/jordigilh/korn/internal/konflux"
	applicationapiv1alpha1 "github.com/konflux-ci/application-api/api/v1alpha1"
	"github.com/urfave/cli/v3"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/duration"
	"k8s.io/cli-runtime/pkg/printers"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

var (
	table = &metav1.Table{
		ColumnDefinitions: []metav1.TableColumnDefinition{
			{Name: "Name", Type: "string"},
			{Name: "Type", Type: "string"},
			{Name: "Age", Type: "string"},
		},
	}
	p    = printers.NewTablePrinter(printers.PrintOptions{})
	korn = konflux.Korn{}
)

func GetCommand() *cli.Command {

	return &cli.Command{
		Name:                  "component",
		Aliases:               []string{"comp", "comps", "components"},
		Usage:                 "get components",
		EnableShellCompletion: true,
		Arguments: []cli.Argument{&cli.StringArg{
			Name:        "component",
			Destination: &korn.ComponentName,
		}},
		Before: func(ctx context.Context, cmd *cli.Command) (context.Context, error) {
			korn.Namespace = ctx.Value(internal.NamespaceCtxType).(string)
			korn.KubeClient = ctx.Value(internal.KubeCliCtxType).(client.Client)
			return ctx, nil
		},
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:        "application",
				Aliases:     []string{"app"},
				Usage:       "-application <application_name>",
				DefaultText: "Application where the components are derived from",
				Destination: &korn.ApplicationName,
			}},
		Description: "Retrieves a component or the list of components. If application is not provided, it will list all components in the namespace ",
		Action: func(ctx context.Context, cmd *cli.Command) error {
			if len(korn.ComponentName) == 0 {
				l, err := korn.ListComponents()
				if err != nil {
					return err
				}
				print(l)
				return nil
			}
			a, err := korn.GetComponent()
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

package application

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
		Name:        "application",
		Aliases:     []string{"app", "apps", "applications"},
		Usage:       "get applications",
		Description: "Retrieves the list of applications in your ",
		Arguments: []cli.Argument{&cli.StringArg{
			Destination: &korn.ApplicationName,
		}},
		Before: func(ctx context.Context, cmd *cli.Command) (context.Context, error) {
			korn.Namespace = ctx.Value(internal.NamespaceCtxType).(string)
			korn.KubeClient = ctx.Value(internal.KubeCliCtxType).(client.Client)
			return ctx, nil
		},
		Action: func(ctx context.Context, cmd *cli.Command) error {
			if len(korn.ApplicationName) == 0 {
				l, err := korn.ListApplications()
				if err != nil {
					return err
				}
				print(l.Items)
				return nil
			}
			a, err := korn.GetApplication()
			if err != nil {
				return err
			}
			print([]applicationapiv1alpha1.Application{*a})
			return nil
		},
	}
}

func print(apps []applicationapiv1alpha1.Application) {
	rows := []metav1.TableRow{}
	for _, v := range apps {
		rows = append(rows, metav1.TableRow{Cells: []interface{}{
			v.Name,
			v.Labels[konflux.ApplicationTypeLabel],
			duration.HumanDuration(time.Since(v.CreationTimestamp.Time)),
		}})
	}
	table.Rows = rows
	p.PrintObj(table, os.Stdout)
}

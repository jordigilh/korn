package releaseplan

import (
	"context"
	"os"
	"time"

	"github.com/jordigilh/korn/internal"
	"github.com/jordigilh/korn/internal/konflux"
	releaseapiv1alpha1 "github.com/konflux-ci/release-service/api/v1alpha1"
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
			{Name: "Application", Type: "string"},
			{Name: "Environment", Type: "string"},
			{Name: "Release Plan Admission", Type: "string"},
			{Name: "Active", Type: "string"},
			{Name: "Age", Type: "string"},
		},
	}
	p    = printers.NewTablePrinter(printers.PrintOptions{})
	korn = konflux.Korn{}
)

func GetCommand() *cli.Command {

	return &cli.Command{
		Name:    "releaseplan",
		Aliases: []string{"rp", "releaseplans"},
		Usage:   "get releaseplans",
		Arguments: []cli.Argument{&cli.StringArg{
			Name:        "releasePlan",
			Destination: &korn.ReleasePlanName,
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
				DefaultText: "Application where the release plans belong to",
				Destination: &korn.ApplicationName,
			},
		},
		Description: "Retrieves a release plan. If application is not provided, it will list all plans in the namespace",
		Action: func(ctx context.Context, cmd *cli.Command) error {
			if len(korn.ReleasePlanName) == 0 {
				l, err := korn.ListReleasePlans()
				if err != nil {
					return err
				}
				print(l)
				return nil
			}
			r, err := korn.GetReleasePlan()
			if err != nil {
				return err
			}
			print([]releaseapiv1alpha1.ReleasePlan{*r})
			return nil
		},
	}
}

func print(releasePlans []releaseapiv1alpha1.ReleasePlan) {
	rows := []metav1.TableRow{}
	for _, v := range releasePlans {
		rows = append(rows, metav1.TableRow{Cells: []interface{}{
			v.Name,
			v.Spec.Application,
			v.Labels[konflux.EnvironmentLabel],
			v.Status.ReleasePlanAdmission.Name,
			v.Status.ReleasePlanAdmission.Active,
			duration.HumanDuration(time.Since(v.CreationTimestamp.Time)),
		}})
	}
	table.Rows = rows
	p.PrintObj(table, os.Stdout)
}

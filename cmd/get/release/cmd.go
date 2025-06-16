package release

import (
	"context"
	"os"
	"time"

	"github.com/jordigilh/korn/internal/konflux"
	releaseapiv1alpha1 "github.com/konflux-ci/release-service/api/v1alpha1"
	"github.com/urfave/cli/v3"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/duration"
	"k8s.io/cli-runtime/pkg/printers"
)

var (
	table = &metav1.Table{
		ColumnDefinitions: []metav1.TableColumnDefinition{
			{Name: "Name", Type: "string"},
			{Name: "Snapshot", Type: "string"},
			{Name: "Release Plan", Type: "string"},
			{Name: "Status", Type: "string"},
			{Name: "Age", Type: "string"},
		},
	}
	p = printers.NewTablePrinter(printers.PrintOptions{})
)

func GetCommand() *cli.Command {

	return &cli.Command{
		Name:    "release",
		Aliases: []string{"releases"},
		Usage:   "get releases",
		Arguments: []cli.Argument{&cli.StringArg{
			Name:        "release",
			Destination: &konflux.ReleaseName,
		}},
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:        "application",
				Aliases:     []string{"app"},
				Usage:       "-application <application_name>",
				DefaultText: "Application where the releases are derived from",
				Destination: &konflux.ApplicationName,
			},
		},
		Description: "Retrieves a release or the list of components. If application is not provided, it will list all snapshots in the namespace",
		Action: func(ctx context.Context, cmd *cli.Command) error {
			if len(konflux.ReleaseName) == 0 {
				l, err := konflux.ListReleases()
				if err != nil {
					return err
				}
				print(l)
				return nil
			}
			r, err := konflux.GetRelease()
			if err != nil {
				return err
			}
			print([]releaseapiv1alpha1.Release{*r})
			return nil
		},
	}
}

func print(comps []releaseapiv1alpha1.Release) {
	rows := []metav1.TableRow{}
	var relStatus string
	for _, v := range comps {
		for _, c := range v.Status.Conditions {
			if c.Type == "Released" {
				relStatus = c.Reason
				break
			}
		}
		rows = append(rows, metav1.TableRow{Cells: []interface{}{
			v.Name,
			v.Spec.Snapshot,
			v.Spec.ReleasePlan,
			relStatus,
			duration.HumanDuration(time.Since(v.CreationTimestamp.Time)),
		}})
	}
	table.Rows = rows
	p.PrintObj(table, os.Stdout)
}

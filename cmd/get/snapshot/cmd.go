package snapshot

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
			{Name: "Application", Type: "string"},
			{Name: "Status", Type: "string"},
			{Name: "Age", Type: "string"},
		},
	}
	p = printers.NewTablePrinter(printers.PrintOptions{})
)

func GetCommand() *cli.Command {

	return &cli.Command{
		Name:    "snapshot",
		Aliases: []string{"snapshots"},
		Usage:   "get snapshots",
		Arguments: []cli.Argument{&cli.StringArg{
			Name:        "snapshot",
			Destination: &konflux.SnapshotName,
		}},
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:        "application",
				Aliases:     []string{"app"},
				Usage:       "-application <application_name>",
				DefaultText: "Application where the components are derived from",
				Destination: &konflux.ApplicationName,
			},
			&cli.BoolFlag{
				Name:        "candidate",
				Aliases:     []string{"c"},
				Usage:       "Example: -candidate",
				DefaultText: "Filters the snapshots that are suitable for the next release. The cutoff snapshot is the last used in a successful release",
				Value:       false,
			},
		},
		Description: "Retrieves a snapshot or the list of components. If application or version is not provided, it will list all snapshots in the namespace",
		Action: func(ctx context.Context, cmd *cli.Command) error {
			if len(konflux.SnapshotName) == 0 {
				if cmd.Bool("candidate") {
					snapshot, err := konflux.GetLatestSnapshotCandidateForRelease()
					if err != nil {
						return err
					}
					print([]applicationapiv1alpha1.Snapshot{*snapshot})
					return nil
				}
				l, err := konflux.ListSnapshots()
				if err != nil {
					return err
				}
				print(l)
				return nil
			}
			s, err := konflux.GetSnapshot()
			if err != nil {
				return err
			}
			print([]applicationapiv1alpha1.Snapshot{*s})
			return nil
		},
	}
}

func print(comps []applicationapiv1alpha1.Snapshot) {
	rows := []metav1.TableRow{}
	var status string
	for _, v := range comps {
		for _, c := range v.Status.Conditions {
			if c.Type == "AppStudioTestSucceeded" {
				status = c.Reason
				break
			}
		}
		rows = append(rows, metav1.TableRow{Cells: []interface{}{
			v.Name,
			v.Spec.Application,
			status,
			duration.HumanDuration(time.Since(v.CreationTimestamp.Time)),
		}})
	}
	table.Rows = rows
	p.PrintObj(table, os.Stdout)
}

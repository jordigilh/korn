package snapshot

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
			{Name: "Application", Type: "string"},
			{Name: "SHA", Type: "string"},
			{Name: "Commit", Type: "string"},
			{Name: "Status", Type: "string"},
			{Name: "Age", Type: "string"},
		},
	}
	p    = printers.NewTablePrinter(printers.PrintOptions{})
	korn = konflux.Korn{}
)

func GetCommand() *cli.Command {

	return &cli.Command{
		Name:    "snapshot",
		Aliases: []string{"snapshots"},
		Usage:   "get snapshots",
		Arguments: []cli.Argument{&cli.StringArg{
			Name:        "snapshot",
			Destination: &korn.SnapshotName,
		}},
		Before: func(ctx context.Context, cmd *cli.Command) (context.Context, error) {
			korn.Namespace = ctx.Value(internal.NamespaceCtxType).(string)
			korn.KubeClient = ctx.Value(internal.KubeCliCtxType).(client.Client)
			korn.PodClient = ctx.Value(internal.PodmanCliCtxType).(internal.ImageClient)
			korn.GitClient = ctx.Value(internal.GitCliCtxType).(internal.GitCommitVersioner)
			return ctx, nil
		},
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:        "application",
				Aliases:     []string{"app"},
				Usage:       "-application <application_name>",
				DefaultText: "Application where the components are derived from",
				Destination: &korn.ApplicationName,
			},
			&cli.StringFlag{
				Name:        "sha",
				Usage:       "Example: -sha 245fca6109a1f32e5ded0f7e330a85401aa2704a",
				DefaultText: "Snapshot associated with the commit SHA",
				Destination: &korn.SHA,
			},
			&cli.StringFlag{
				Name:        "version",
				Usage:       "Example: -version v0.0.11",
				DefaultText: "Retrieves the latest snapshot that matches the given version in the bundle's label",
				Destination: &korn.Version,
			},
			&cli.BoolFlag{
				Name:        "candidate",
				Aliases:     []string{"c"},
				Usage:       "Example: -candidate",
				DefaultText: "Filters the snapshots that are suitable for the next release. The cutoff snapshot is the last used in a successful release",
			},
		},
		Description: "Retrieves a snapshot or the list of components. If application or version is not provided, it will list all snapshots in the namespace",
		Action: func(ctx context.Context, cmd *cli.Command) error {
			if len(korn.SnapshotName) != 0 || len(korn.SHA) > 0 {
				s, err := korn.GetSnapshot()
				if err != nil {
					return err
				}
				print([]applicationapiv1alpha1.Snapshot{*s})
				return nil
			}
			if len(korn.Version) > 0 {
				s, err := korn.GetLatestSnapshotByVersion()
				if err != nil {
					return err
				}
				print([]applicationapiv1alpha1.Snapshot{*s})
				return nil
			}
			if cmd.Bool("candidate") {
				snapshot, err := korn.GetSnapshotCandidateForRelease()
				if err != nil {
					return err
				}
				print([]applicationapiv1alpha1.Snapshot{*snapshot})
				return nil
			}
			l, err := korn.ListSnapshots()
			if err != nil {
				return err
			}
			print(l)
			return nil
		},
	}
}

func print(snapshots []applicationapiv1alpha1.Snapshot) {
	rows := []metav1.TableRow{}
	var status string
	for _, v := range snapshots {
		for _, c := range v.Status.Conditions {
			if c.Type == "AppStudioTestSucceeded" {
				status = c.Reason
				break
			}
		}
		rows = append(rows, metav1.TableRow{Cells: []interface{}{
			v.Name,
			v.Spec.Application,
			v.Labels["pac.test.appstudio.openshift.io/sha"],
			v.Annotations["pac.test.appstudio.openshift.io/sha-title"],
			status,
			duration.HumanDuration(time.Since(v.CreationTimestamp.Time)),
		}})
	}
	table.Rows = rows
	p.PrintObj(table, os.Stdout)
}

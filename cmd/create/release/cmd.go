package release

import (
	"context"
	"fmt"
	"os"
	"reflect"

	"github.com/jordigilh/korn/internal"
	"github.com/jordigilh/korn/internal/konflux"
	"github.com/sirupsen/logrus"

	"github.com/urfave/cli/v3"
	mjson "k8s.io/apimachinery/pkg/runtime/serializer/json"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/yaml"
)

var (
	korn = konflux.Korn{}
)

func CreateCommand() *cli.Command {
	return &cli.Command{
		Name:    "release",
		Aliases: []string{"releases"},
		Usage:   "create releases",
		Arguments: []cli.Argument{&cli.StringArg{
			Name: "release",
		}},
		Before: func(ctx context.Context, cmd *cli.Command) (context.Context, error) {
			korn.Namespace = ctx.Value(internal.NamespaceCtxType).(string)
			korn.KubeClient = ctx.Value(internal.KubeCliCtxType).(client.Client)
			korn.PodClient = ctx.Value(internal.PodmanCliCtxType).(internal.ImageClient)
			return ctx, nil
		},
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:        "application",
				Aliases:     []string{"app"},
				Usage:       "-application <application_name>",
				DefaultText: "Application where the releases are derived from",
				Destination: &korn.ApplicationName,
			},
			&cli.StringFlag{
				Name:    "environment",
				Aliases: []string{"env"},
				Usage:   "Example: -env staging",
				Validator: func(val string) error {
					if val != "staging" && val != "production" {
						return fmt.Errorf("invalid value %s: only 'staging' or 'production' supported", val)
					}
					return nil
				},
				Value:       "staging",
				DefaultText: "Captures the target environment: staging or production",
				Destination: &korn.EnvironmentName,
			},
			&cli.StringFlag{
				Name:        "snapshot",
				Usage:       "-snapshot <application_name>",
				DefaultText: "Use this snapshot for the release instead of the latest candidate",
				Destination: &korn.SnapshotName,
			},
			&cli.BoolFlag{
				Name:        "dryrun",
				Usage:       "Example: -dryrun ",
				Value:       false,
				Destination: &korn.DryRun,
				DefaultText: "Outputs the manifest to use when creating a new release. This command is incompatible with the 'wait' flag",
			},
			&cli.BoolFlag{
				Name:        "wait",
				Aliases:     []string{"w"},
				Usage:       "Example: -w ",
				Value:       true,
				DefaultText: "When creating a release, this command will instruct the CLI to wait for the completion of the release pipeline and return the results. This command is incompatible with the 'dryrun' flag",
			},
			&cli.BoolFlag{
				Name:        "force",
				Aliases:     []string{"f"},
				Usage:       "Example: -f ",
				Value:       true,
				DefaultText: "Force the creation of the release, even if the snapshot has been used in a previous release. Useful when retrying for a failed release. If no filter is provided (snapshot name or hash), it will fetch the last valid candidate.",
				Destination: &korn.ForceRelease,
			},
			&cli.StringFlag{
				Name:        "sha",
				Usage:       "Example: -sha 245fca6109a1f32e5ded0f7e330a85401aa2704a",
				DefaultText: "Use the snapshot associated to this commit SHA in the release instead of latest candidate",
				Destination: &korn.SHA,
			},
			&cli.StringFlag{
				Name:        "output",
				Aliases:     []string{"o"},
				Usage:       "-o <type>",
				DefaultText: "Print the release manifest: valid entries are 'json' or 'yaml'",
				Validator: func(val string) error {
					if val != "json" && val != "yaml" {
						return fmt.Errorf("invalid output type %s: only 'json' or 'yaml' are supported", val)
					}
					return nil
				},
				Destination: &korn.OutputType,
			},
			&cli.IntFlag{
				Name:        "timeout",
				Aliases:     []string{"t"},
				Usage:       "-timeout timeout in minutes",
				DefaultText: "Time out in minutes for the wait for operation to complete",
				Destination: &korn.WaitForTimeout,
				Value:       60,
			},
			&cli.StringFlag{
				Name:        "releaseNotes",
				Aliases:     []string{"rn"},
				Usage:       "-releaseNotes 'FILE'",
				DefaultText: "Path to the file containing release notes to be attached to the Release manifest in YAML format",
				Validator: func(val string) error {
					b, err := os.ReadFile(val)
					if err != nil {
						return err
					}
					rn := konflux.ReleaseNote{}
					err = yaml.Unmarshal(b, &rn)
					if err != nil {
						return err
					}
					if reflect.DeepEqual(rn, konflux.ReleaseNote{}) {
						return fmt.Errorf("no content found for release notes in %s", val)
					}
					return nil
				},
				Action: func(ctx context.Context, c *cli.Command, s string) error {
					b, err := os.ReadFile(c.String("releaseNotes"))
					if err != nil {
						return err
					}
					rn := konflux.ReleaseNote{}
					err = yaml.Unmarshal(b, &rn)
					if err != nil {
						return err
					}
					korn.ReleaseNotes = &rn
					return nil
				},
			},
		},
		Description: "Creates a release or the list of components. If application or version is not provided, it will list all snapshots in the namespace",
		Action: func(ctx context.Context, cmd *cli.Command) error {
			m, err := korn.GenerateReleaseManifest()
			if err != nil {
				return err
			}
			if len(korn.OutputType) > 0 {
				s := mjson.NewSerializerWithOptions(
					mjson.DefaultMetaFactory, nil, nil,
					mjson.SerializerOptions{Yaml: korn.OutputType == "yaml", Pretty: true, Strict: true},
				)
				return s.Encode(m, os.Stdout)
			}
			r, err := korn.CreateRelease(*m)
			if err != nil {
				return err
			}
			logrus.Infof("Release created %s", r.Name)
			if cmd.Bool("wait") {
				kubeCfgPath := ctx.Value(internal.KubeConfigCtxType).(string)
				err = korn.WaitForReleaseToComplete(*r, kubeCfgPath)
				if err != nil {
					return err
				}
				fmt.Printf("Release %s/%s has completed successfully", r.Namespace, r.Name)
			}
			return nil
		},
	}
}

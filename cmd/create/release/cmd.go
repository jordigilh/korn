package release

import (
	"context"
	"fmt"
	"os"
	"reflect"
	"strconv"

	"github.com/jordigilh/korn/internal"
	"github.com/jordigilh/korn/internal/konflux"
	"github.com/sirupsen/logrus"

	"github.com/urfave/cli/v3"
	mjson "k8s.io/apimachinery/pkg/runtime/serializer/json"
	"k8s.io/client-go/dynamic"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/yaml"
)

var (
	korn = konflux.Korn{WaitForTimeout: 60, EnvironmentName: "staging"}
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
			korn.DynamicClient = ctx.Value(internal.DynamicCliCtxType).(dynamic.Interface)
			return ctx, nil
		},
		MutuallyExclusiveFlags: []cli.MutuallyExclusiveFlags{
			{
				Flags: [][]cli.Flag{
					{
						&cli.BoolFlag{
							Name:        "dryrun",
							Usage:       "Outputs the manifest to use when creating a new release. This command is incompatible with the 'wait' flag.",
							Value:       false,
							Destination: &korn.DryRun,
							DefaultText: strconv.FormatBool(korn.DryRun),
						},
						&cli.BoolFlag{
							Name:        "wait",
							Aliases:     []string{"w"},
							Usage:       "When creating a release, this command will instruct the CLI to wait for the completion of the release pipeline and return the results. This command is incompatible with the 'dryrun' flag",
							Value:       true,
							DefaultText: strconv.FormatBool(true),
						},
					},
				},
			},
		},
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:        "application",
				Aliases:     []string{"app"},
				Usage:       "Example: -application my-application",
				DefaultText: korn.ApplicationName,
				Destination: &korn.ApplicationName,
			},
			&cli.StringFlag{
				Name:    "environment",
				Aliases: []string{"env"},
				Usage:   "Example: -environment staging",
				Validator: func(val string) error {
					if val != "staging" && val != "production" {
						return fmt.Errorf("invalid value %s: only 'staging' or 'production' supported", val)
					}
					return nil
				},
				DefaultText: korn.EnvironmentName,
				Destination: &korn.EnvironmentName,
			},
			&cli.StringFlag{
				Name:        "snapshot",
				Usage:       "Example: -snapshot my-app-snapshot-abc123",
				DefaultText: "",
				Destination: &korn.SnapshotName,
			},
			&cli.BoolFlag{
				Name:        "force",
				Aliases:     []string{"f"},
				Usage:       "Force the creation of the release, even if the snapshot has been used in a previous release. Useful when retrying for a failed release. If no filter is provided (snapshot name or hash), it will fetch the last valid candidate.",
				Value:       false,
				DefaultText: strconv.FormatBool(korn.ForceRelease),
				Destination: &korn.ForceRelease,
			},
			&cli.StringFlag{
				Name:        "sha",
				Usage:       "Example: -sha 245fca6109a1f32e5ded0f7e330a85401aa2704a",
				DefaultText: korn.SHA,
				Destination: &korn.SHA,
			},
			&cli.StringFlag{
				Name:        "output",
				Aliases:     []string{"o"},
				Usage:       "Ouptuts the manifest in yaml or json format. Example: -output yaml",
				DefaultText: korn.OutputType,
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
				Usage:       "Time out in minutes for the wait for operation to complete. Example: -timeout 10",
				DefaultText: fmt.Sprintf("%d", korn.WaitForTimeout),
				Destination: &korn.WaitForTimeout,
				Value:       korn.WaitForTimeout,
			},
			&cli.StringFlag{
				Name:    "releaseNotes",
				Aliases: []string{"rn"},
				Usage:   "Example: -releaseNotes /path/to/release-notes.yaml",
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
		Description: "Creates a release for a given application and environment",
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
				err = korn.WaitForReleaseToComplete(*r)
				if err != nil {
					return err
				}
				fmt.Printf("Release %s/%s has completed successfully", r.Namespace, r.Name)
			}
			return nil
		},
	}
}

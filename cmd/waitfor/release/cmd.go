package release

import (
	"context"

	"github.com/jordigilh/korn/internal"
	"github.com/jordigilh/korn/internal/konflux"
	"github.com/urfave/cli/v3"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

var (
	korn = konflux.Korn{}
)

func WaitForCommand() *cli.Command {
	return &cli.Command{
		Name:    "release",
		Aliases: []string{"releases"},
		Usage:   "waitfor release <release_name>",
		Flags: []cli.Flag{
			&cli.IntFlag{
				Name:        "timeout",
				Aliases:     []string{"t"},
				Usage:       "-timeout timeout in minutes",
				DefaultText: "Time out in minutes for the wait for operation to complete",
				Value:       60,
				Destination: &korn.WaitForTimeout,
			},
		},
		Before: func(ctx context.Context, cmd *cli.Command) (context.Context, error) {
			korn.Namespace = ctx.Value(internal.NamespaceCtxType).(string)
			korn.KubeClient = ctx.Value(internal.KubeCliCtxType).(client.Client)
			return ctx, nil
		},
		Arguments: []cli.Argument{&cli.StringArg{
			Name:        "release",
			Destination: &korn.ReleaseName,
		}},
		Description: "Waits for an existing release to finish by periodically checking every 10 seconds for the status of the release until it's either Failed or Succeeeded. Timeout occurs after 60 minutes",
		Action: func(ctx context.Context, cmd *cli.Command) error {
			r, err := korn.GetRelease()
			if err != nil {
				return err
			}
			kubeConfigPath := ctx.Value(internal.KubeConfigCtxType).(string)
			err = korn.WaitForReleaseToComplete(*r, kubeConfigPath)
			if err != nil {
				return err
			}
			return nil
		},
	}
}

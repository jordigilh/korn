package main

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/jordigilh/korn/cmd/create"
	"github.com/jordigilh/korn/cmd/get"
	"github.com/jordigilh/korn/cmd/waitfor"
	"github.com/jordigilh/korn/internal"
	"github.com/sirupsen/logrus"
	"github.com/urfave/cli/v3"
)

var (
	debug   bool
	version string = "dev" // Set via ldflags during build
)

func main() {
	// Set up logrus defaults
	logrus.SetFormatter(&logrus.TextFormatter{
		FullTimestamp: true,
	})
	logrus.SetOutput(os.Stdout)
	cmd := &cli.Command{
		Name:                  "korn",
		Usage:                 "",
		DefaultCommand:        "korn -h",
		EnableShellCompletion: true,
		Description:           "korn is an opinionated application that is designed to simplify the release of an operator in Konflux by extracting the arduous tasks that are necessary to ensure the success of a release. The tool requires the konflux manifests that represent the construct of the operator to be labeled accordingly so that the CLI can navigate through its structures",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:  "kubeconfig",
				Usage: "Example: -kubeconfig ~/.kube/config",
				Value: internal.GetDefaultKubeconfigPath(),
			},
			&cli.StringFlag{
				Name:    "namespace",
				Aliases: []string{"n"},
				Usage:   "Example: -namespace my-namespace",
				Value:   internal.GetCurrentNamespace(),
			},
			&cli.BoolFlag{
				Name:        "debug",
				Aliases:     []string{"d"},
				Usage:       "Enable debug mode",
				Destination: &debug,
			},
			&cli.BoolFlag{
				Name:    "version",
				Aliases: []string{"v"},
				Usage:   "Print version information",
				Action: func(ctx context.Context, cmd *cli.Command, value bool) error {
					if value {
						fmt.Printf("korn version %s\n", version)
						os.Exit(0)
					}
					return nil
				},
			},
		},
		Before: func(ctx context.Context, cmd *cli.Command) (context.Context, error) {
			if cmd.Bool("version") {
				return ctx, nil
			}
			if debug {
				logrus.SetLevel(logrus.DebugLevel)
				logrus.Debug("Debug mode enabled")
			} else {
				logrus.SetLevel(logrus.InfoLevel)
			}
			podClient, err := internal.NewPodmanClient()
			if err != nil {
				return nil, err
			}
			kubeClient, err := internal.GetClient(cmd.String("kubeconfig"))
			if err != nil {
				return nil, err
			}
			dynamicClient, err := internal.GetDynamicClient(cmd.String("kubeconfig"))
			if err != nil {
				return nil, err
			}
			ctx = context.WithValue(ctx, internal.NamespaceCtxType, cmd.String("namespace"))
			ctx = context.WithValue(ctx, internal.PodmanCliCtxType, podClient)
			ctx = context.WithValue(ctx, internal.GitCliCtxType, internal.NewGitClient())
			ctx = context.WithValue(ctx, internal.DynamicCliCtxType, dynamicClient)
			ctx = context.WithValue(ctx, internal.KubeCliCtxType, kubeClient)
			return ctx, nil
		},
		Commands: []*cli.Command{
			get.Command(),
			create.Command(),
			waitfor.Command()},
	}

	if err := cmd.Run(context.Background(), os.Args); err != nil {
		log.Fatal(err)
	}
}

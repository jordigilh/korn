package main

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/jordigilh/korn/cmd/application"
	"github.com/jordigilh/korn/cmd/component"
	"github.com/jordigilh/korn/cmd/release"
	"github.com/jordigilh/korn/cmd/snapshot"
	"github.com/jordigilh/korn/internal"
	"github.com/sirupsen/logrus"
	"github.com/urfave/cli/v3"
)

func main() {
	// Set up logrus defaults
	logrus.SetFormatter(&logrus.TextFormatter{
		FullTimestamp: true,
	})
	logrus.SetOutput(os.Stdout)
	cmd := &cli.Command{
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:  "kubeconfig",
				Value: internal.GetDefaultKubeconfigPath(),
			},
			&cli.StringFlag{
				Name:        "namespace",
				Aliases:     []string{"n"},
				Usage:       "-namespace <namespace>",
				DefaultText: "Current namespace",
				Value:       internal.GetCurrentNamespace(),
			},
			&cli.BoolFlag{
				Name:    "debug",
				Aliases: []string{"d"},
				Usage:   "Enable debug mode",
			},
		},
		Before: func(ctx context.Context, cmd *cli.Command) (context.Context, error) {
			internal.Kubeconfig = cmd.String("kubeconfig")
			if cmd.Bool("debug") {
				logrus.SetLevel(logrus.DebugLevel)
				logrus.Debug("Debug mode enabled")
			} else {
				logrus.SetLevel(logrus.InfoLevel)
			}
			return ctx, nil
		},

		Commands: []*cli.Command{
			{
				Name:  "get",
				Usage: "get <resources>",
				Commands: []*cli.Command{
					application.GetCommand(),
					component.GetCommand(),
					snapshot.GetCommand(),
					release.GetCommand(),
				},
				Action: func(ctx context.Context, cmd *cli.Command) error {
					fmt.Println("added task: ", cmd.Args().First())
					return nil
				},
			},
			{
				Name:  "create",
				Usage: "create release",
				Commands: []*cli.Command{
					release.CreateCommand(),
				},
				Action: func(ctx context.Context, cmd *cli.Command) error {
					fmt.Println("added task: ", cmd.Args().First())
					return nil
				},
			},
		},
	}

	if err := cmd.Run(context.Background(), os.Args); err != nil {
		log.Fatal(err)
	}
}

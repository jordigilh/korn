package konflux

import (
	"github.com/jordigilh/korn/internal"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type Korn struct {
	Namespace       string
	ApplicationName string
	ComponentName   string
	ReleaseName     string
	EnvironmentName string
	ReleaseType     string
	SnapshotName    string
	Version         string
	ForceRelease    bool
	WaitForTimeout  int
	DryRun          bool
	OutputType      string
	SHA             string
	KubeClient      client.Client
	PodClient       internal.ImageClient
	GitClient       internal.GetRevisionVersioner
}

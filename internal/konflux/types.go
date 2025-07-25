package konflux

import (
	"github.com/jordigilh/korn/internal"
	"k8s.io/client-go/dynamic"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type Korn struct {
	Namespace       string
	ApplicationName string
	ComponentName   string
	ReleaseName     string
	ReleasePlanName string
	EnvironmentName string
	SnapshotName    string
	Version         string
	ForceRelease    bool
	WaitForTimeout  int
	ReleaseNotes    *ReleaseNote
	DryRun          bool
	OutputType      string
	SHA             string
	KubeClient      client.Client
	PodClient       internal.ImageClient
	GitClient       internal.GitCommitVersioner
	DynamicClient   dynamic.Interface
}

type ReleaseNote struct {
	Type       releaseType         `json:"type" yaml:"type"`
	Issues     map[string]any      `json:"issues,omitempty" yaml:"issues,omitempty"`
	CVEs       []map[string]string `json:"cves,omitempty" yaml:"cves,omitempty"`
	References []string            `json:"reference,omitempty" yaml:"reference,omitempty"`
}

const (
	bugReleaseType      releaseType = "RHBA"
	securityReleaseType releaseType = "RHSA"
	featureReleaseType  releaseType = "RHEA"
)

type releaseType string

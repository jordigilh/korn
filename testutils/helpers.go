package testutils

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/containers/podman/v5/pkg/domain/entities/types"
	"github.com/containers/podman/v5/pkg/inspect"
	"github.com/jordigilh/korn/internal"
	"github.com/jordigilh/korn/internal/konflux"
	applicationapiv1alpha1 "github.com/konflux-ci/application-api/api/v1alpha1"
	releaseapiv1alpha1 "github.com/konflux-ci/release-service/api/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/dynamic"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

// Test constants
const (
	TestNamespace           = "test-namespace"
	TestAppName             = "test-app"
	OtherAppName            = "other-app"
	TestComponentName       = "test-component"
	TestReleaseName         = "test-release"
	TestSnapshotName        = "test-snapshot"
	TestReleasePlan         = "test-releaseplan"
	BundleComponentName     = "bundle-component"
	ControllerComponentName = "controller-component"
)

// Konflux label constants for tests
const (
	EventTypeLabel   = "pac.test.appstudio.openshift.io/event-type"
	ApplicationLabel = "appstudio.openshift.io/application"
	ComponentLabel   = "appstudio.openshift.io/component"
	SHALabel         = "pac.test.appstudio.openshift.io/sha"
	SHATitleLabel    = "pac.test.appstudio.openshift.io/sha-title"

	// Event type values
	PushEventType = "push"
)

// Common test setup helpers
type TestSetup struct {
	FakeClientBuilder *fake.ClientBuilder
	Scheme            *runtime.Scheme
	Namespace         *corev1.Namespace
	Context           context.Context
}

func NewTestSetup(scheme *runtime.Scheme) *TestSetup {
	ns := NewNamespace(TestNamespace)
	return &TestSetup{
		FakeClientBuilder: fake.NewClientBuilder().WithScheme(scheme).WithRuntimeObjects(ns),
		Scheme:            scheme,
		Namespace:         ns,
		Context:           context.WithValue(context.Background(), internal.NamespaceCtxType, TestNamespace),
	}
}

func (ts *TestSetup) WithObjects(objects ...runtime.Object) *TestSetup {
	if len(objects) > 0 {
		ts.FakeClientBuilder = ts.FakeClientBuilder.WithRuntimeObjects(objects...)
	}
	return ts
}

func (ts *TestSetup) WithKubeClient() context.Context {
	return context.WithValue(ts.Context, internal.KubeCliCtxType, ts.FakeClientBuilder.Build())
}

// Basic object creation helpers
func NewNamespace(name string) *corev1.Namespace {
	return &corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: name}}
}

// Application helpers
func NewApplication(name, namespace string, labels map[string]string) *applicationapiv1alpha1.Application {
	return &applicationapiv1alpha1.Application{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
			Labels:    labels,
		},
	}
}

func NewOperatorApplication(name, namespace string) *applicationapiv1alpha1.Application {
	return NewApplication(name, namespace, map[string]string{
		konflux.ApplicationTypeLabel: "operator",
	})
}

func NewFBCApplication(name, namespace string) *applicationapiv1alpha1.Application {
	return NewApplication(name, namespace, map[string]string{
		konflux.ApplicationTypeLabel: "fbc",
	})
}

// Component helpers
func NewComponent(name, namespace, application string, labels map[string]string) *applicationapiv1alpha1.Component {
	return &applicationapiv1alpha1.Component{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
			Labels:    labels,
		},
		Spec: applicationapiv1alpha1.ComponentSpec{Application: application},
	}
}

func NewBundleComponent(name, namespace, application string) *applicationapiv1alpha1.Component {
	return NewComponent(name, namespace, application, map[string]string{
		konflux.ComponentTypeLabel: "bundle",
	})
}

func NewControllerComponent(name, namespace, application string) *applicationapiv1alpha1.Component {
	return NewComponent(name, namespace, application, map[string]string{
		konflux.BundleReferenceLabel: "controller",
	})
}

// Release helpers
func NewRelease(name, namespace, snapshot, releasePlan string, labels map[string]string) *releaseapiv1alpha1.Release {
	return &releaseapiv1alpha1.Release{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
			Labels:    labels,
		},
		Spec: releaseapiv1alpha1.ReleaseSpec{
			Snapshot:    snapshot,
			ReleasePlan: releasePlan,
		},
	}
}

func NewSuccessfulRelease(name, namespace, snapshot, releasePlan, app, component string) *releaseapiv1alpha1.Release {
	release := NewRelease(name, namespace, snapshot, releasePlan, map[string]string{
		"appstudio.openshift.io/application": app,
		"appstudio.openshift.io/component":   component,
	})
	release.Status = releaseapiv1alpha1.ReleaseStatus{
		Conditions: []metav1.Condition{
			{
				Type:   "Released",
				Reason: "Succeeded",
				Status: metav1.ConditionTrue,
			},
		},
	}
	return release
}

func NewFailedRelease(name, namespace, snapshot, releasePlan, app, component string) *releaseapiv1alpha1.Release {
	release := NewRelease(name, namespace, snapshot, releasePlan, map[string]string{
		"appstudio.openshift.io/application": app,
		"appstudio.openshift.io/component":   component,
	})
	release.Status = releaseapiv1alpha1.ReleaseStatus{
		Conditions: []metav1.Condition{
			{
				Type:   "Released",
				Reason: "Failed",
				Status: metav1.ConditionFalse,
			},
		},
	}
	return release
}

// ReleasePlan helpers
func NewReleasePlan(name, namespace, application string, labels map[string]string) *releaseapiv1alpha1.ReleasePlan {
	return &releaseapiv1alpha1.ReleasePlan{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
			Labels:    labels,
		},
		Spec: releaseapiv1alpha1.ReleasePlanSpec{
			Application: application,
		},
	}
}

func NewStagingReleasePlan(name, namespace, application string) *releaseapiv1alpha1.ReleasePlan {
	return NewReleasePlan(name, namespace, application, map[string]string{
		konflux.EnvironmentLabel: "staging",
	})
}

// Snapshot helpers
func NewSnapshot(name, namespace, application, component, sha string) *applicationapiv1alpha1.Snapshot {
	return &applicationapiv1alpha1.Snapshot{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
			Labels: map[string]string{
				EventTypeLabel:   PushEventType,
				ApplicationLabel: application,
				ComponentLabel:   component,
				SHALabel:         sha,
			},
		},
		Spec: applicationapiv1alpha1.SnapshotSpec{
			Application: application,
			Components: []applicationapiv1alpha1.SnapshotComponent{
				{
					Name:           "controller-component",
					ContainerImage: "registry.test.com/controller@sha256:abc123",
				},
				{
					Name:           component,
					ContainerImage: "registry.test.com/bundle@sha256:def456",
				},
			},
		},
		Status: applicationapiv1alpha1.SnapshotStatus{
			Conditions: []metav1.Condition{
				{
					Type:   "AppStudioTestSucceeded",
					Reason: "Finished",
					Status: metav1.ConditionTrue,
				},
			},
		},
	}
}

func NewTestSnapshot() *applicationapiv1alpha1.Snapshot {
	return NewSnapshot(TestSnapshotName, TestNamespace, TestAppName, BundleComponentName, "abc123def456")
}

// Common test data sets
func GetBasicApplications() []runtime.Object {
	return []runtime.Object{
		NewOperatorApplication("operator-app", TestNamespace),
		NewFBCApplication("fbc-app", TestNamespace),
	}
}

func GetBasicComponents() []runtime.Object {
	return []runtime.Object{
		NewBundleComponent(BundleComponentName, TestNamespace, TestAppName),
		NewControllerComponent(ControllerComponentName, TestNamespace, TestAppName),
	}
}

func GetBasicReleases() []runtime.Object {
	return []runtime.Object{
		NewSuccessfulRelease(TestReleaseName, TestNamespace, TestSnapshotName, TestReleasePlan, TestAppName, TestComponentName),
		NewFailedRelease("failed-release", TestNamespace, "failed-snapshot", TestReleasePlan, TestAppName, TestComponentName),
	}
}

// Complete test sets for commands that need multiple object types
func GetCompleteReleaseTestSet() []runtime.Object {
	return []runtime.Object{
		// Applications
		NewOperatorApplication(TestAppName, TestNamespace),
		NewOperatorApplication(OtherAppName, TestNamespace),
		// Components
		NewBundleComponent(BundleComponentName, TestNamespace, TestAppName),
		NewBundleComponent("other-bundle", TestNamespace, OtherAppName),
		// Releases
		NewSuccessfulRelease(TestReleaseName, TestNamespace, TestSnapshotName, TestReleasePlan, TestAppName, TestComponentName),
		NewFailedRelease("failed-release", TestNamespace, "failed-snapshot", TestReleasePlan, TestAppName, TestComponentName),
	}
}

func GetReleaseTestSetForApp(appName string) []runtime.Object {
	return []runtime.Object{
		// Application
		NewOperatorApplication(appName, TestNamespace),
		// Component
		NewBundleComponent(BundleComponentName, TestNamespace, appName),
		// Releases
		NewSuccessfulRelease("release1", TestNamespace, "snapshot1", TestReleasePlan, appName, TestComponentName),
		NewSuccessfulRelease("release2", TestNamespace, "snapshot2", TestReleasePlan, appName, TestComponentName),
	}
}

func GetAppWithoutReleases(appName string) []runtime.Object {
	return []runtime.Object{
		// Application
		NewOperatorApplication(appName, TestNamespace),
		// Component (needed for release command filtering)
		NewBundleComponent(BundleComponentName, TestNamespace, appName),
		// No releases for this app
	}
}

// Complete test sets for create commands
func GetCompleteCreateReleaseTestSet() []runtime.Object {
	return []runtime.Object{
		// Application
		NewOperatorApplication(TestAppName, TestNamespace),
		// Components
		NewControllerComponent(ControllerComponentName, TestNamespace, TestAppName),
		NewBundleComponent(BundleComponentName, TestNamespace, TestAppName),
		// Snapshot
		NewTestSnapshot(),
		// Release Plan
		NewStagingReleasePlan(TestReleasePlan, TestNamespace, TestAppName),
	}
}

// Mock clients for create command tests
type MockImageClient struct{}

func (m *MockImageClient) GetImageData(image string) (*types.ImageInspectReport, error) {
	return &types.ImageInspectReport{
		ImageData: &inspect.ImageData{
			Labels: map[string]string{
				"controller": "registry.test.com/controller@sha256:abc123",
				"version":    "1.0.0",
			},
		},
	}, nil
}

// Test file helpers
func CreateTempReleaseNotesFile() (string, string, error) {
	tempDir, err := os.MkdirTemp("", "korn-test")
	if err != nil {
		return "", "", err
	}

	releaseNotesFile := filepath.Join(tempDir, "release-notes.yaml")
	content := `---
type: RHBA
issues:
  fixed:
    - id: TEST-12345
      source: issues.redhat.com
    - id: TEST-67890
      source: bugzilla.redhat.com
`
	err = os.WriteFile(releaseNotesFile, []byte(content), 0644)
	if err != nil {
		os.RemoveAll(tempDir)
		return "", "", err
	}

	return tempDir, releaseNotesFile, nil
}

// Index helpers for fake clients
func FilterBySnapshotName(obj client.Object) []string {
	return []string{string(obj.(*applicationapiv1alpha1.Snapshot).Name)}
}

// Enhanced test setup for create commands
type CreateTestSetup struct {
	*TestSetup
	DynamicClient dynamic.Interface
	MockPodClient *MockImageClient
	TempDir       string
	ReleaseNotes  string
}

func NewCreateTestSetup(scheme *runtime.Scheme) (*CreateTestSetup, error) {
	baseSetup := NewTestSetup(scheme)

	tempDir, releaseNotesFile, err := CreateTempReleaseNotesFile()
	if err != nil {
		return nil, fmt.Errorf("failed to create temp release notes: %w", err)
	}

	return &CreateTestSetup{
		TestSetup:     baseSetup,
		MockPodClient: &MockImageClient{},
		TempDir:       tempDir,
		ReleaseNotes:  releaseNotesFile,
	}, nil
}

func (cts *CreateTestSetup) WithKubeClientAndMocks() context.Context {
	ctx := cts.WithKubeClient()
	ctx = context.WithValue(ctx, internal.PodmanCliCtxType, cts.MockPodClient)
	if cts.DynamicClient != nil {
		ctx = context.WithValue(ctx, internal.DynamicCliCtxType, cts.DynamicClient)
	}
	return ctx
}

func (cts *CreateTestSetup) Cleanup() {
	if cts.TempDir != "" {
		os.RemoveAll(cts.TempDir)
	}
}

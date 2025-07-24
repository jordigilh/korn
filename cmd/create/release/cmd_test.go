package release_test

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/containers/podman/v5/pkg/domain/entities/types"
	"github.com/containers/podman/v5/pkg/inspect"
	"github.com/jordigilh/korn/cmd/create/release"
	"github.com/jordigilh/korn/internal"
	"github.com/jordigilh/korn/internal/konflux"
	applicationapiv1alpha1 "github.com/konflux-ci/application-api/api/v1alpha1"
	releaseapiv1alpha1 "github.com/konflux-ci/release-service/api/v1alpha1"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/urfave/cli/v3"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/watch"
	dfake "k8s.io/client-go/dynamic/fake"
	k8stesting "k8s.io/client-go/testing"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

// Test case structure
type releaseTestCase struct {
	name             string
	args             []string
	releaseNotesFile bool
	expectedError    bool
	description      string
}

var _ = Describe("Create Release Command", func() {
	var (
		fakeClientBuilder *fake.ClientBuilder
		scheme            *runtime.Scheme
		ns                *corev1.Namespace
		app               *applicationapiv1alpha1.Application
		component1        *applicationapiv1alpha1.Component
		bundleComponent   *applicationapiv1alpha1.Component
		snapshot          *applicationapiv1alpha1.Snapshot
		releasePlan       *releaseapiv1alpha1.ReleasePlan
		ctx               context.Context
		cmd               *cli.Command
		tempDir           string
		fw                *watch.FakeWatcher
		releaseNotesFile  string
	)

	BeforeEach(func() {
		scheme = createFakeScheme()
		ns = newNamespace("test-namespace")

		app = &applicationapiv1alpha1.Application{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test-app",
				Namespace: "test-namespace",
				Labels: map[string]string{
					konflux.ApplicationTypeLabel: "operator",
				},
			},
		}

		component1 = &applicationapiv1alpha1.Component{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "controller-component",
				Namespace: "test-namespace",
				Labels: map[string]string{
					konflux.BundleReferenceLabel: "controller",
				},
			},
			Spec: applicationapiv1alpha1.ComponentSpec{
				Application: "test-app",
			},
		}

		bundleComponent = &applicationapiv1alpha1.Component{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "bundle-component",
				Namespace: "test-namespace",
				Labels: map[string]string{
					konflux.ComponentTypeLabel: "bundle",
				},
			},
			Spec: applicationapiv1alpha1.ComponentSpec{
				Application: "test-app",
			},
		}

		snapshot = &applicationapiv1alpha1.Snapshot{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test-snapshot",
				Namespace: "test-namespace",
				Labels: map[string]string{
					"pac.test.appstudio.openshift.io/event-type": "push",
					"appstudio.openshift.io/application":         "test-app",
					"appstudio.openshift.io/component":           "bundle-component",
					"pac.test.appstudio.openshift.io/sha":        "abc123def456",
				},
			},
			Spec: applicationapiv1alpha1.SnapshotSpec{
				Application: "test-app",
				Components: []applicationapiv1alpha1.SnapshotComponent{
					{
						Name:           "controller-component",
						ContainerImage: "registry.test.com/controller@sha256:abc123",
					},
					{
						Name:           "bundle-component",
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

		releasePlan = &releaseapiv1alpha1.ReleasePlan{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test-release-plan",
				Namespace: "test-namespace",
				Labels: map[string]string{
					konflux.EnvironmentLabel: "staging",
				},
			},
			Spec: releaseapiv1alpha1.ReleasePlanSpec{
				Application: "test-app",
			},
		}

		// Create temp directory and release notes file
		var err error
		tempDir, err = os.MkdirTemp("", "korn-test")
		Expect(err).ToNot(HaveOccurred())

		releaseNotesFile = filepath.Join(tempDir, "release-notes.yaml")
		fmt.Printf("releaseNotesFile: %s\n", releaseNotesFile)
		err = os.WriteFile(releaseNotesFile, []byte(`---
type: RHBA
issues:
  fixed:
    - id: TEST-12345
      source: issues.redhat.com
    - id: TEST-67890
      source: bugzilla.redhat.com
`), 0644)
		Expect(err).ToNot(HaveOccurred())

		fakeClientBuilder = fake.NewClientBuilder().WithScheme(scheme).WithRuntimeObjects(
			ns, app, component1, bundleComponent, snapshot, releasePlan,
		).WithIndex(&applicationapiv1alpha1.Snapshot{}, "metadata.name", filterBySnapshotName)

		ctx = context.WithValue(context.Background(), internal.NamespaceCtxType, "test-namespace")
		ctx = context.WithValue(ctx, internal.KubeCliCtxType, fakeClientBuilder.Build())

		mockPodClient := &mockImageClient{}
		ctx = context.WithValue(ctx, internal.PodmanCliCtxType, mockPodClient)

		// Create a fake dynamic client with proper list kinds
		scheme := runtime.NewScheme()
		dynamicClient := dfake.NewSimpleDynamicClient(scheme)

		// Setup a fake watch
		fw = watch.NewFake()

		// Inject the fake watch into the client
		dynamicClient.PrependWatchReactor("releases", func(action k8stesting.Action) (bool, watch.Interface, error) {
			return true, fw, nil
		})
		ctx = context.WithValue(ctx, internal.DynamicCliCtxType, dynamicClient)

		cmd = release.CreateCommand()
	})

	AfterEach(func() {
		os.RemoveAll(tempDir)
	})

	Context("Basic positive scenarios", func() {
		DescribeTable("should create release successfully",
			func(testCase releaseTestCase) {
				args := testCase.args
				if testCase.releaseNotesFile {
					args = append(args, "--releaseNotes", releaseNotesFile)
				}

				fullArgs := append([]string{"create", "release"}, args...)
				err := cmd.Run(ctx, fullArgs)

				if testCase.expectedError {
					Expect(err).To(HaveOccurred(), testCase.description)
				} else {
					Expect(err).ToNot(HaveOccurred(), testCase.description)
				}
			},

			Entry("basic staging release", releaseTestCase{
				name:          "basic staging release",
				args:          []string{"--app", "test-app", "--environment", "staging", "--wait=false", "--dryrun"},
				expectedError: false,
				description:   "Should create basic staging release",
			}),

			Entry("staging release with release notes", releaseTestCase{
				name:             "staging release with notes",
				args:             []string{"--app", "test-app", "--environment", "staging", "--wait=false", "--dryrun"},
				releaseNotesFile: true,
				expectedError:    false,
				description:      "Should create staging release with notes",
			}),

			Entry("release with specific snapshot", releaseTestCase{
				name:          "release with specific snapshot",
				args:          []string{"--app", "test-app", "--environment", "staging", "--snapshot", "test-snapshot", "--wait=false", "--dryrun"},
				expectedError: false,
				description:   "Should create release with specific snapshot",
			}),

			Entry("release with commit SHA", releaseTestCase{
				name:          "release with commit SHA",
				args:          []string{"--app", "test-app", "--environment", "staging", "--sha", "abc123def456", "--wait=false", "--dryrun"},
				expectedError: false,
				description:   "Should create release with commit SHA",
			}),

			Entry("forced release", releaseTestCase{
				name:          "forced release",
				args:          []string{"--app", "test-app", "--environment", "staging", "--force", "--wait=false", "--dryrun"},
				expectedError: false,
				description:   "Should create forced release",
			}),

			Entry("dry run with YAML output", releaseTestCase{
				name:          "dry run with YAML output",
				args:          []string{"--app", "test-app", "--environment", "staging", "--dryrun", "--output", "yaml"},
				expectedError: false,
				description:   "Should perform dry run with YAML output",
			}),

			Entry("dry run with JSON output", releaseTestCase{
				name:          "dry run with JSON output",
				args:          []string{"--app", "test-app", "--environment", "staging", "--dryrun", "--output", "json"},
				expectedError: false,
				description:   "Should perform dry run with JSON output",
			}),

			Entry("release with custom timeout", releaseTestCase{
				name:          "release with custom timeout",
				args:          []string{"--app", "test-app", "--environment", "staging", "--timeout", "120", "--wait=false", "--dryrun"},
				expectedError: false,
				description:   "Should create release with custom timeout",
			}),
		)
	})

	Context("Error scenarios", func() {
		DescribeTable("should handle errors appropriately",
			func(testCase releaseTestCase) {
				fullArgs := append([]string{"create", "release"}, testCase.args...)
				err := cmd.Run(ctx, fullArgs)
				if testCase.expectedError {
					Expect(err).To(HaveOccurred(), testCase.description)
				} else {
					Expect(err).ToNot(HaveOccurred(), testCase.description)
				}
			},

			Entry("invalid environment", releaseTestCase{
				name:          "invalid environment",
				args:          []string{"--app", "test-app", "--environment", "invalid"},
				expectedError: true,
				description:   "Should fail with invalid environment",
			}),

			Entry("invalid output format", releaseTestCase{
				name:          "invalid output format",
				args:          []string{"--app", "test-app", "--environment", "staging", "--dryrun", "--output", "xml"},
				expectedError: true,
				description:   "Should fail with invalid output format",
			}),

			Entry("missing application name", releaseTestCase{
				name:          "missing application name",
				args:          []string{"--environment", "staging"},
				expectedError: true,
				description:   "Should fail without application name",
			}),

			Entry("invalid release notes file", releaseTestCase{
				name:          "invalid release notes file",
				args:          []string{"--app", "test-app", "--environment", "staging", "--releaseNotes", "/nonexistent/file.yaml"},
				expectedError: true,
				description:   "Should fail with invalid release notes file",
			}),
		)
	})

	Context("Flag aliases", func() {
		It("should work with short flag names", func() {
			testCase := releaseTestCase{
				name:          "short flag names",
				args:          []string{"--app", "test-app", "--env", "staging", "-f", "-w=false", "-o", "yaml", "--dryrun"},
				expectedError: false,
				description:   "Should work with short flag names",
			}
			fullArgs := append([]string{"create", "release"}, testCase.args...)
			err := cmd.Run(ctx, fullArgs)
			Expect(err).ToNot(HaveOccurred())
		})

		It("should work with release notes short flag", func() {
			testCase := releaseTestCase{
				name:          "release notes short flag",
				args:          []string{"--app", "test-app", "--environment", "staging", "--rn", releaseNotesFile, "--wait=false", "--dryrun"},
				expectedError: false,
				description:   "Should work with release notes short flag",
			}
			fullArgs := append([]string{"create", "release"}, testCase.args...)
			err := cmd.Run(ctx, fullArgs)
			Expect(err).ToNot(HaveOccurred())
		})
	})
})

func newNamespace(name string) *corev1.Namespace {
	return &corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: name}}
}

// Mock image client that implements the interface properly
type mockImageClient struct{}

func (m *mockImageClient) GetImageData(image string) (*types.ImageInspectReport, error) {
	// Create a mock ImageInspectReport with the Labels field populated
	// Based on the usage in the code, Labels appears to be a direct field
	return &types.ImageInspectReport{
		ImageData: &inspect.ImageData{
			Labels: map[string]string{
				"controller": "registry.test.com/controller@sha256:abc123",
				"version":    "1.0.0",
			},
		},
	}, nil
}

func filterBySnapshotName(obj client.Object) []string {
	return []string{string(obj.(*applicationapiv1alpha1.Snapshot).Name)}
}

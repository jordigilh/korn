// NOTE: This file contains AI-generated test cases and patterns (Cursor)
// All test logic has been reviewed and validated for correctness

package release_test

import (
	"github.com/jordigilh/korn/cmd/create/release"
	"github.com/jordigilh/korn/testutils"
	applicationapiv1alpha1 "github.com/konflux-ci/application-api/api/v1alpha1"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/urfave/cli/v3"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/watch"
	dfake "k8s.io/client-go/dynamic/fake"
	k8stesting "k8s.io/client-go/testing"
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
		createTestSetup *testutils.CreateTestSetup
		cmd             *cli.Command
		fw              *watch.FakeWatcher
	)

	BeforeEach(func() {
		var err error
		createTestSetup, err = testutils.NewCreateTestSetup(createFakeScheme())
		Expect(err).ToNot(HaveOccurred())

		// Set up fake dynamic client with watch
		scheme := runtime.NewScheme()
		dynamicClient := dfake.NewSimpleDynamicClient(scheme)
		fw = watch.NewFake()
		dynamicClient.PrependWatchReactor("releases", func(action k8stesting.Action) (bool, watch.Interface, error) {
			return true, fw, nil
		})
		createTestSetup.DynamicClient = dynamicClient

		// Add test objects
		objects := testutils.GetCompleteCreateReleaseTestSet()
		createTestSetup.WithObjects(objects...)

		// Add snapshot index for filtering
		createTestSetup.FakeClientBuilder = createTestSetup.FakeClientBuilder.WithIndex(&applicationapiv1alpha1.Snapshot{}, "metadata.name", testutils.FilterBySnapshotName)

		cmd = release.CreateCommand()
	})

	AfterEach(func() {
		createTestSetup.Cleanup()
	})

	Context("Basic positive scenarios", func() {
		DescribeTable("should create release successfully",
			func(testCase releaseTestCase) {
				ctx := createTestSetup.WithKubeClientAndMocks()
				args := testCase.args
				if testCase.releaseNotesFile {
					args = append(args, "--releaseNotes", createTestSetup.ReleaseNotes)
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
				args:          []string{"--app", testutils.TestAppName, "--environment", "staging", "--wait=false", "--dryrun"},
				expectedError: false,
				description:   "Should create basic staging release",
			}),

			Entry("staging release with release notes", releaseTestCase{
				name:             "staging release with notes",
				args:             []string{"--app", testutils.TestAppName, "--environment", "staging", "--wait=false", "--dryrun"},
				releaseNotesFile: true,
				expectedError:    false,
				description:      "Should create staging release with notes",
			}),

			Entry("release with specific snapshot", releaseTestCase{
				name:          "release with specific snapshot",
				args:          []string{"--app", testutils.TestAppName, "--environment", "staging", "--snapshot", testutils.TestSnapshotName, "--wait=false", "--dryrun"},
				expectedError: false,
				description:   "Should create release with specific snapshot",
			}),

			Entry("release with commit SHA", releaseTestCase{
				name:          "release with commit SHA",
				args:          []string{"--app", testutils.TestAppName, "--environment", "staging", "--sha", "abc123def456", "--wait=false", "--dryrun"},
				expectedError: false,
				description:   "Should create release with commit SHA",
			}),

			Entry("forced release", releaseTestCase{
				name:          "forced release",
				args:          []string{"--app", testutils.TestAppName, "--environment", "staging", "--force", "--wait=false", "--dryrun"},
				expectedError: false,
				description:   "Should create forced release",
			}),

			Entry("dry run with YAML output", releaseTestCase{
				name:          "dry run with YAML output",
				args:          []string{"--app", testutils.TestAppName, "--environment", "staging", "--dryrun", "--output", "yaml"},
				expectedError: false,
				description:   "Should perform dry run with YAML output",
			}),

			Entry("dry run with JSON output", releaseTestCase{
				name:          "dry run with JSON output",
				args:          []string{"--app", testutils.TestAppName, "--environment", "staging", "--dryrun", "--output", "json"},
				expectedError: false,
				description:   "Should perform dry run with JSON output",
			}),

			Entry("release with custom timeout", releaseTestCase{
				name:          "release with custom timeout",
				args:          []string{"--app", testutils.TestAppName, "--environment", "staging", "--timeout", "120", "--wait=false", "--dryrun"},
				expectedError: false,
				description:   "Should create release with custom timeout",
			}),
		)
	})

	Context("Error scenarios", func() {
		DescribeTable("should handle errors appropriately",
			func(testCase releaseTestCase) {
				ctx := createTestSetup.WithKubeClientAndMocks()
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
				args:          []string{"--app", testutils.TestAppName, "--environment", "invalid"},
				expectedError: true,
				description:   "Should fail with invalid environment",
			}),

			Entry("invalid output format", releaseTestCase{
				name:          "invalid output format",
				args:          []string{"--app", testutils.TestAppName, "--environment", "staging", "--dryrun", "--output", "xml"},
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
				args:          []string{"--app", testutils.TestAppName, "--environment", "staging", "--releaseNotes", "/nonexistent/file.yaml"},
				expectedError: true,
				description:   "Should fail with invalid release notes file",
			}),
		)
	})

	Context("Flag aliases", func() {
		It("should work with short flag names", func() {
			ctx := createTestSetup.WithKubeClientAndMocks()
			testCase := releaseTestCase{
				name:          "short flag names",
				args:          []string{"--app", testutils.TestAppName, "--env", "staging", "-f", "-w=false", "-o", "yaml", "--dryrun"},
				expectedError: false,
				description:   "Should work with short flag names",
			}
			fullArgs := append([]string{"create", "release"}, testCase.args...)
			err := cmd.Run(ctx, fullArgs)
			Expect(err).ToNot(HaveOccurred())
		})

		It("should work with release notes short flag", func() {
			ctx := createTestSetup.WithKubeClientAndMocks()
			testCase := releaseTestCase{
				name:          "release notes short flag",
				args:          []string{"--app", testutils.TestAppName, "--environment", "staging", "--rn", createTestSetup.ReleaseNotes, "--wait=false", "--dryrun"},
				expectedError: false,
				description:   "Should work with release notes short flag",
			}
			fullArgs := append([]string{"create", "release"}, testCase.args...)
			err := cmd.Run(ctx, fullArgs)
			Expect(err).ToNot(HaveOccurred())
		})
	})
})

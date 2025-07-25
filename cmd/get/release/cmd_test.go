package release_test

import (
	"github.com/jordigilh/korn/cmd/get/release"
	"github.com/jordigilh/korn/testutils"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/urfave/cli/v3"
	"k8s.io/apimachinery/pkg/runtime"
)

var _ = Describe("Get Release Command", func() {
	var (
		testSetup *testutils.TestSetup
		cmd       *cli.Command
	)

	BeforeEach(func() {
		testSetup = testutils.NewTestSetup(createFakeScheme())
		cmd = release.GetCommand()
	})

	Context("List all releases", func() {
		DescribeTable("should list releases successfully",
			func(releases []runtime.Object, description string) {
				ctx := testSetup.WithObjects(releases...).WithKubeClient()
				args := []string{""}

				err := cmd.Run(ctx, args)
				Expect(err).ToNot(HaveOccurred(), description)
			},

			Entry("with no releases",
				[]runtime.Object{},
				"Should handle empty release list"),

			Entry("with single successful release",
				[]runtime.Object{
					testutils.NewSuccessfulRelease(testutils.TestReleaseName, testutils.TestNamespace, testutils.TestSnapshotName, testutils.TestReleasePlan, testutils.TestAppName, testutils.TestComponentName),
				},
				"Should list single successful release"),

			Entry("with multiple releases",
				testutils.GetBasicReleases(),
				"Should list multiple releases"),
		)
	})

	Context("List releases with --application flag", func() {
		DescribeTable("should filter releases by application",
			func(objects []runtime.Object, appName string, description string) {
				ctx := testSetup.WithObjects(objects...).WithKubeClient()
				args := []string{"", "--app", appName}

				err := cmd.Run(ctx, args)
				Expect(err).ToNot(HaveOccurred(), description)
			},

			Entry("with matching application",
				testutils.GetReleaseTestSetForApp(testutils.TestAppName),
				testutils.TestAppName,
				"Should filter by application name"),

			Entry("with no matching application",
				testutils.GetAppWithoutReleases(testutils.TestAppName),
				testutils.TestAppName,
				"Should return empty list for non-matching application"),
		)
	})

	Context("Get specific release", func() {
		DescribeTable("should get release by name",
			func(objects []runtime.Object, releaseName string, expectError bool, description string) {
				ctx := testSetup.WithObjects(objects...).WithKubeClient()
				args := []string{"", releaseName, "--app", testutils.TestAppName}

				err := cmd.Run(ctx, args)

				if expectError {
					Expect(err).To(HaveOccurred(), description)
				} else {
					Expect(err).ToNot(HaveOccurred(), description)
				}
			},

			Entry("existing release",
				append(testutils.GetReleaseTestSetForApp(testutils.TestAppName),
					testutils.NewSuccessfulRelease(testutils.TestReleaseName, testutils.TestNamespace, testutils.TestSnapshotName, testutils.TestReleasePlan, testutils.TestAppName, testutils.TestComponentName),
				),
				testutils.TestReleaseName, false,
				"Should get existing release"),

			Entry("non-existing release",
				testutils.GetReleaseTestSetForApp(testutils.TestAppName),
				"non-existing-release", true,
				"Should fail for non-existing release"),
		)
	})

	Context("Release status", func() {
		It("should handle successful releases", func() {
			objects := testutils.GetReleaseTestSetForApp(testutils.TestAppName)
			ctx := testSetup.WithObjects(objects...).WithKubeClient()

			args := []string{"", "--app", testutils.TestAppName}
			err := cmd.Run(ctx, args)
			Expect(err).ToNot(HaveOccurred())
		})

		It("should handle failed releases", func() {
			objects := append(testutils.GetReleaseTestSetForApp(testutils.TestAppName),
				testutils.NewFailedRelease("failed-release", testutils.TestNamespace, testutils.TestSnapshotName, testutils.TestReleasePlan, testutils.TestAppName, testutils.TestComponentName),
			)
			ctx := testSetup.WithObjects(objects...).WithKubeClient()

			args := []string{"", "--app", testutils.TestAppName}
			err := cmd.Run(ctx, args)
			Expect(err).ToNot(HaveOccurred())
		})
	})
})

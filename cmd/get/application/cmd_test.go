package application_test

import (
	"github.com/jordigilh/korn/cmd/get/application"
	"github.com/jordigilh/korn/testutils"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/urfave/cli/v3"

	"k8s.io/apimachinery/pkg/runtime"
)

var _ = Describe("Get Application Command", func() {
	var (
		testSetup *testutils.TestSetup
		cmd       *cli.Command
	)

	BeforeEach(func() {
		testSetup = testutils.NewTestSetup(createFakeScheme())
		cmd = application.GetCommand()
	})

	Context("List all applications", func() {
		DescribeTable("should list applications successfully",
			func(applications []runtime.Object, expectedCount int, description string) {
				ctx := testSetup.WithObjects(applications...).WithKubeClient()
				args := []string{""}

				err := cmd.Run(ctx, args)
				Expect(err).ToNot(HaveOccurred(), description)
			},

			Entry("with no applications",
				[]runtime.Object{}, 0,
				"Should handle empty application list"),

			Entry("with single operator application",
				[]runtime.Object{
					testutils.NewOperatorApplication("operator-app", testutils.TestNamespace),
				}, 1,
				"Should list single operator application"),

			Entry("with single FBC application",
				[]runtime.Object{
					testutils.NewFBCApplication("fbc-app", testutils.TestNamespace),
				}, 1,
				"Should list single FBC application"),

			Entry("with multiple applications",
				testutils.GetBasicApplications(), 2,
				"Should list multiple applications"),
		)
	})

	Context("Get specific application", func() {
		DescribeTable("should get application by name",
			func(applications []runtime.Object, appName string, withError bool, description string) {
				ctx := testSetup.WithObjects(applications...).WithKubeClient()
				args := []string{"", appName}

				err := cmd.Run(ctx, args)

				if withError {
					Expect(err).To(HaveOccurred(), description)
				} else {
					Expect(err).ToNot(HaveOccurred(), description)
				}
			},

			Entry("existing application",
				[]runtime.Object{
					testutils.NewOperatorApplication(testutils.TestAppName, testutils.TestNamespace),
				},
				testutils.TestAppName, false,
				"Should get existing application"),

			Entry("non-existing application",
				[]runtime.Object{},
				"non-existing-app", true,
				"Should fail for non-existing application"),

			Entry("application in different namespace",
				[]runtime.Object{
					testutils.NewOperatorApplication(testutils.TestAppName, "other-namespace"),
				},
				testutils.TestAppName, true,
				"Should fail for application in different namespace"),
		)
	})

	Context("Command aliases", func() {
		It("should contain 'app' and 'apps aliases", func() {
			Expect(cmd.Aliases).To(BeEquivalentTo([]string{"app", "apps", "applications"}))
		})
	})
})

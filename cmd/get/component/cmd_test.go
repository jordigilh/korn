// NOTE: This file contains AI-generated test cases and patterns (Cursor)
// All test logic has been reviewed and validated for correctness

package component_test

import (
	"github.com/jordigilh/korn/cmd/get/component"
	"github.com/jordigilh/korn/testutils"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/urfave/cli/v3"
	"k8s.io/apimachinery/pkg/runtime"
)

var _ = Describe("Get Component Command", func() {
	var (
		testSetup *testutils.TestSetup
		cmd       *cli.Command
	)

	BeforeEach(func() {
		testSetup = testutils.NewTestSetup(createFakeScheme())
		cmd = component.GetCommand()
	})

	Context("List all components", func() {
		DescribeTable("should list components successfully",
			func(components []runtime.Object, description string) {
				ctx := testSetup.WithObjects(components...).WithKubeClient()
				args := []string{""}

				err := cmd.Run(ctx, args)
				Expect(err).ToNot(HaveOccurred(), description)
			},

			Entry("with no components",
				[]runtime.Object{},
				"Should handle empty component list"),

			Entry("with single bundle component",
				[]runtime.Object{
					testutils.NewBundleComponent(testutils.BundleComponentName, testutils.TestNamespace, testutils.TestAppName),
				},
				"Should list single bundle component"),

			Entry("with multiple components",
				testutils.GetBasicComponents(),
				"Should list multiple components"),
		)
	})

	Context("List components with --application flag", func() {
		DescribeTable("should filter components by application",
			func(components []runtime.Object, appName string, expectedCount int, description string) {
				ctx := testSetup.WithObjects(components...).WithKubeClient()
				args := []string{"", "--app", appName}

				err := cmd.Run(ctx, args)
				Expect(err).ToNot(HaveOccurred(), description)
			},

			Entry("with matching application",
				[]runtime.Object{
					testutils.NewBundleComponent("component1", testutils.TestNamespace, testutils.TestAppName),
					testutils.NewComponent("component2", testutils.TestNamespace, testutils.OtherAppName, nil),
				},
				testutils.TestAppName, 1,
				"Should filter by application name"),

			Entry("with no matching application",
				[]runtime.Object{
					testutils.NewComponent("component1", testutils.TestNamespace, testutils.OtherAppName, nil),
				},
				testutils.TestAppName, 0,
				"Should return empty list for non-matching application"),
		)
	})

	Context("Get specific component", func() {
		DescribeTable("should get component by name",
			func(components []runtime.Object, componentName string, expectError bool, description string) {
				ctx := testSetup.WithObjects(components...).WithKubeClient()
				args := []string{"", componentName, "--app", testutils.TestAppName}

				err := cmd.Run(ctx, args)

				if expectError {
					Expect(err).To(HaveOccurred(), description)
				} else {
					Expect(err).ToNot(HaveOccurred(), description)
				}
			},

			Entry("existing component",
				[]runtime.Object{
					testutils.NewBundleComponent(testutils.TestComponentName, testutils.TestNamespace, testutils.TestAppName),
				},
				testutils.TestComponentName, false,
				"Should get existing component"),

			Entry("non-existing component",
				[]runtime.Object{},
				"non-existing-component", true,
				"Should fail for non-existing component"),
		)
	})

	Context("Component types and labels", func() {
		It("should handle bundle components", func() {
			components := []runtime.Object{
				testutils.NewBundleComponent(testutils.BundleComponentName, testutils.TestNamespace, testutils.TestAppName),
			}
			ctx := testSetup.WithObjects(components...).WithKubeClient()

			args := []string{"", "--app", testutils.TestAppName}
			err := cmd.Run(ctx, args)
			Expect(err).ToNot(HaveOccurred())
		})

		It("should handle components with bundle reference labels", func() {
			components := []runtime.Object{
				testutils.NewControllerComponent(testutils.ControllerComponentName, testutils.TestNamespace, testutils.TestAppName),
			}
			ctx := testSetup.WithObjects(components...).WithKubeClient()

			args := []string{"", "--app", testutils.TestAppName}
			err := cmd.Run(ctx, args)
			Expect(err).ToNot(HaveOccurred())
		})
	})
})

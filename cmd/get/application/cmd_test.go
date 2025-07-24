package application_test

import (
	"context"

	"github.com/jordigilh/korn/cmd/get/application"
	"github.com/jordigilh/korn/internal"
	"github.com/jordigilh/korn/internal/konflux"
	applicationapiv1alpha1 "github.com/konflux-ci/application-api/api/v1alpha1"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/urfave/cli/v3"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

var _ = Describe("Get Application Command", func() {
	var (
		fakeClientBuilder *fake.ClientBuilder
		scheme            *runtime.Scheme
		ns                *corev1.Namespace
		ctx               context.Context
		cmd               *cli.Command
	)

	BeforeEach(func() {
		scheme = createFakeScheme()
		ns = newNamespace("test-namespace")
		fakeClientBuilder = fake.NewClientBuilder().WithScheme(scheme).WithRuntimeObjects(ns)

		ctx = context.WithValue(context.Background(), internal.NamespaceCtxType, "test-namespace")

		cmd = application.GetCommand()
	})

	Context("List all applications", func() {
		DescribeTable("should list applications successfully",
			func(applications []runtime.Object, expectedCount int, description string) {
				args := []string{""}
				if len(applications) > 0 {
					fakeClientBuilder = fakeClientBuilder.WithRuntimeObjects(applications...)
					if len(applications) == 1 {
						args = []string{applications[0].(metav1.Object).GetName()}
					}
				}
				ctx = context.WithValue(ctx, internal.KubeCliCtxType, fakeClientBuilder.Build())
				err := cmd.Run(ctx, args)

				Expect(err).ToNot(HaveOccurred(), description)
			},

			Entry("with no applications",
				[]runtime.Object{},
				0,
				"Should handle empty application list"),

			Entry("with single operator application",
				[]runtime.Object{
					&applicationapiv1alpha1.Application{
						ObjectMeta: metav1.ObjectMeta{
							Name:      "operator-app",
							Namespace: "test-namespace",
							Labels: map[string]string{
								konflux.ApplicationTypeLabel: "operator",
							},
						},
					},
				},
				1,
				"Should list single operator application"),

			Entry("with single FBC application",
				[]runtime.Object{
					&applicationapiv1alpha1.Application{
						ObjectMeta: metav1.ObjectMeta{
							Name:      "fbc-app",
							Namespace: "test-namespace",
							Labels: map[string]string{
								konflux.ApplicationTypeLabel: "fbc",
							},
						},
					},
				},
				1,
				"Should list single FBC application"),

			Entry("with multiple applications",
				[]runtime.Object{
					&applicationapiv1alpha1.Application{
						ObjectMeta: metav1.ObjectMeta{
							Name:      "operator-app",
							Namespace: "test-namespace",
							Labels: map[string]string{
								konflux.ApplicationTypeLabel: "operator",
							},
						},
					},
					&applicationapiv1alpha1.Application{
						ObjectMeta: metav1.ObjectMeta{
							Name:      "fbc-app",
							Namespace: "test-namespace",
							Labels: map[string]string{
								konflux.ApplicationTypeLabel: "fbc",
							},
						},
					},
				},
				2,
				"Should list multiple applications"),
		)
	})

	Context("Get specific application", func() {
		DescribeTable("should get application by name",
			func(applications []runtime.Object, appName string, withError bool, description string) {
				if len(applications) > 0 {
					fakeClientBuilder = fakeClientBuilder.WithRuntimeObjects(applications...)
				}
				ctx = context.WithValue(ctx, internal.KubeCliCtxType, fakeClientBuilder.Build())

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
					&applicationapiv1alpha1.Application{
						ObjectMeta: metav1.ObjectMeta{
							Name:      "test-app",
							Namespace: "test-namespace",
							Labels: map[string]string{
								konflux.ApplicationTypeLabel: "operator",
							},
						},
					},
				},
				"test-app",
				false,
				"Should get existing application"),

			Entry("non-existing application",
				[]runtime.Object{},
				"non-existing-app",
				true,
				"Should fail for non-existing application"),

			Entry("application in different namespace",
				[]runtime.Object{
					&applicationapiv1alpha1.Application{
						ObjectMeta: metav1.ObjectMeta{
							Name:      "test-app",
							Namespace: "other-namespace",
							Labels: map[string]string{
								konflux.ApplicationTypeLabel: "operator",
							},
						},
					},
				},
				"test-app",
				true,
				"Should fail for application in different namespace"),
		)
	})

	Context("Command aliases", func() {
		It("should contain 'app' and 'apps aliases", func() {
			Expect(cmd.Aliases).To(BeEquivalentTo([]string{"app", "apps", "applications"}))
		})

	})
})

// Helper functions

func newNamespace(name string) *corev1.Namespace {
	return &corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: name}}
}

package component_test

import (
	"context"

	"github.com/jordigilh/korn/cmd/get/component"
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

var _ = Describe("Get Component Command", func() {
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

		ctx = context.WithValue(context.Background(), internal.NamespaceCtxType, "test-namespace")
		fakeClientBuilder = fake.NewClientBuilder().WithScheme(scheme).WithRuntimeObjects(ns)

		cmd = component.GetCommand()
	})

	Context("List all components", func() {
		DescribeTable("should list components successfully",
			func(components []runtime.Object, description string) {
				args := []string{""}
				if len(components) > 0 {
					fakeClientBuilder = fakeClientBuilder.WithRuntimeObjects(components...)
					if len(components) == 1 {
						args = []string{components[0].(metav1.Object).GetName()}
					}
				}
				ctx = context.WithValue(ctx, internal.KubeCliCtxType, fakeClientBuilder.Build())
				err := cmd.Run(ctx, args)

				Expect(err).ToNot(HaveOccurred(), description)
			},

			Entry("with no components",
				[]runtime.Object{},
				"Should handle empty component list"),

			Entry("with single bundle component",
				[]runtime.Object{
					&applicationapiv1alpha1.Component{
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
					},
				},
				"Should list single bundle component"),

			Entry("with multiple components",
				[]runtime.Object{
					&applicationapiv1alpha1.Component{
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
					},
					&applicationapiv1alpha1.Component{
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
					},
				},
				"Should list multiple components"),
		)
	})

	Context("List components with --application flag", func() {
		DescribeTable("should filter components by application",
			func(components []runtime.Object, appName string, expectedCount int, description string) {
				args := []string{""}
				if len(components) > 0 {
					fakeClientBuilder = fakeClientBuilder.WithRuntimeObjects(components...)
					if len(components) == 1 {
						args = []string{components[0].(metav1.Object).GetName()}
					}
				}
				ctx = context.WithValue(ctx, internal.KubeCliCtxType, fakeClientBuilder.Build())
				args = append(args, []string{"--app", appName}...)
				err := cmd.Run(ctx, args)

				Expect(err).ToNot(HaveOccurred(), description)
			},

			Entry("with matching application",
				[]runtime.Object{
					&applicationapiv1alpha1.Component{
						ObjectMeta: metav1.ObjectMeta{
							Name:      "component1",
							Namespace: "test-namespace",
							Labels: map[string]string{
								konflux.ComponentTypeLabel: "bundle",
							},
						},
						Spec: applicationapiv1alpha1.ComponentSpec{
							Application: "test-app",
						},
					},
					&applicationapiv1alpha1.Component{
						ObjectMeta: metav1.ObjectMeta{
							Name:      "component2",
							Namespace: "test-namespace",
						},
						Spec: applicationapiv1alpha1.ComponentSpec{
							Application: "other-app",
						},
					},
				},
				"test-app",
				1,
				"Should filter by application name"),

			Entry("with no matching application",
				[]runtime.Object{
					&applicationapiv1alpha1.Component{
						ObjectMeta: metav1.ObjectMeta{
							Name:      "component1",
							Namespace: "test-namespace",
						},
						Spec: applicationapiv1alpha1.ComponentSpec{
							Application: "other-app",
						},
					},
				},
				"test-app",
				0,
				"Should return empty list for non-matching application"),
		)
	})

	Context("Get specific component", func() {
		DescribeTable("should get component by name",
			func(components []runtime.Object, componentName string, expectError bool, description string) {
				args := []string{"", componentName, "--app", "test-app"}
				if len(components) > 0 {
					fakeClientBuilder = fakeClientBuilder.WithRuntimeObjects(components...)
				}
				ctx = context.WithValue(ctx, internal.KubeCliCtxType, fakeClientBuilder.Build())

				err := cmd.Run(ctx, args)

				if expectError {
					Expect(err).To(HaveOccurred(), description)
				} else {
					Expect(err).ToNot(HaveOccurred(), description)
				}
			},

			Entry("existing component",
				[]runtime.Object{
					&applicationapiv1alpha1.Component{
						ObjectMeta: metav1.ObjectMeta{
							Name:      "test-component",
							Namespace: "test-namespace",
							Labels: map[string]string{
								konflux.ComponentTypeLabel: "bundle",
							},
						},
						Spec: applicationapiv1alpha1.ComponentSpec{
							Application: "test-app",
						},
					},
				},
				"test-component",
				false,
				"Should get existing component"),

			Entry("non-existing component",
				[]runtime.Object{},
				"non-existing-component",
				true,
				"Should fail for non-existing component"),
		)
	})

	Context("Component types and labels", func() {
		It("should handle bundle components", func() {
			components := []runtime.Object{
				&applicationapiv1alpha1.Component{
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
				},
			}
			fakeClientBuilder = fakeClientBuilder.WithRuntimeObjects(components...)
			ctx = context.WithValue(ctx, internal.KubeCliCtxType, fakeClientBuilder.Build())

			args := []string{"", "--app", "test-app"}
			err := cmd.Run(ctx, args)
			Expect(err).ToNot(HaveOccurred())
		})

		It("should handle components with bundle reference labels", func() {
			components := []runtime.Object{
				&applicationapiv1alpha1.Component{
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
				},
			}
			fakeClientBuilder = fakeClientBuilder.WithRuntimeObjects(components...)
			ctx = context.WithValue(ctx, internal.KubeCliCtxType, fakeClientBuilder.Build())

			args := []string{"", "--app", "test-app"}
			err := cmd.Run(ctx, args)
			Expect(err).ToNot(HaveOccurred())
		})
	})
})

func newNamespace(name string) *corev1.Namespace {
	return &corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: name}}
}

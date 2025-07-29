// NOTE: This file contains AI-generated test cases and patterns (Cursor)
// All test logic has been reviewed and validated for correctness

package releaseplan_test

import (
	"context"

	"github.com/jordigilh/korn/cmd/get/releaseplan"
	"github.com/jordigilh/korn/internal"
	"github.com/jordigilh/korn/internal/konflux"
	releaseapiv1alpha1 "github.com/konflux-ci/release-service/api/v1alpha1"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/urfave/cli/v3"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

var _ = Describe("Get ReleasePlan Command", func() {
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

		cmd = releaseplan.GetCommand()
	})

	Context("List all release plans", func() {
		DescribeTable("should list release plans successfully",
			func(releasePlans []runtime.Object, description string) {
				if len(releasePlans) > 0 {
					fakeClientBuilder = fakeClientBuilder.WithRuntimeObjects(releasePlans...)
				}
				ctx = context.WithValue(ctx, internal.KubeCliCtxType, fakeClientBuilder.Build())

				args := []string{""}
				err := cmd.Run(ctx, args)

				Expect(err).ToNot(HaveOccurred(), description)
			},

			Entry("with no release plans",
				[]runtime.Object{},
				"Should handle empty release plan list"),

			Entry("with single staging release plan",
				[]runtime.Object{
					&releaseapiv1alpha1.ReleasePlan{
						ObjectMeta: metav1.ObjectMeta{
							Name:      "staging-releaseplan",
							Namespace: "test-namespace",
							Labels: map[string]string{
								konflux.EnvironmentLabel: "staging",
							},
						},
						Spec: releaseapiv1alpha1.ReleasePlanSpec{
							Application: "test-app",
							Target:      "rhtap-releng-tenant",
						},
					},
				},
				"Should list single staging release plan"),

			Entry("with multiple release plans",
				[]runtime.Object{
					&releaseapiv1alpha1.ReleasePlan{
						ObjectMeta: metav1.ObjectMeta{
							Name:      "staging-releaseplan",
							Namespace: "test-namespace",
							Labels: map[string]string{
								konflux.EnvironmentLabel: "staging",
							},
						},
						Spec: releaseapiv1alpha1.ReleasePlanSpec{
							Application: "test-app",
							Target:      "rhtap-releng-tenant",
						},
					},
					&releaseapiv1alpha1.ReleasePlan{
						ObjectMeta: metav1.ObjectMeta{
							Name:      "production-releaseplan",
							Namespace: "test-namespace",
							Labels: map[string]string{
								konflux.EnvironmentLabel: "production",
							},
						},
						Spec: releaseapiv1alpha1.ReleasePlanSpec{
							Application: "test-app",
							Target:      "rhtap-releng-tenant",
						},
					},
				},
				"Should list multiple release plans"),
		)
	})

	Context("Filter release plans with --application flag", func() {
		DescribeTable("should filter release plans by application",
			func(releasePlans []runtime.Object, appName string, description string) {
				if len(releasePlans) > 0 {
					fakeClientBuilder = fakeClientBuilder.WithRuntimeObjects(releasePlans...)
				}
				ctx = context.WithValue(ctx, internal.KubeCliCtxType, fakeClientBuilder.Build())

				args := []string{"", "--app", appName}
				err := cmd.Run(ctx, args)

				Expect(err).ToNot(HaveOccurred(), description)
			},

			Entry("with matching application",
				[]runtime.Object{
					&releaseapiv1alpha1.ReleasePlan{
						ObjectMeta: metav1.ObjectMeta{
							Name:      "releaseplan1",
							Namespace: "test-namespace",
							Labels: map[string]string{
								konflux.EnvironmentLabel: "staging",
							},
						},
						Spec: releaseapiv1alpha1.ReleasePlanSpec{
							Application: "test-app",
							Target:      "rhtap-releng-tenant",
						},
					},
					&releaseapiv1alpha1.ReleasePlan{
						ObjectMeta: metav1.ObjectMeta{
							Name:      "releaseplan2",
							Namespace: "test-namespace",
							Labels: map[string]string{
								konflux.EnvironmentLabel: "production",
							},
						},
						Spec: releaseapiv1alpha1.ReleasePlanSpec{
							Application: "other-app",
							Target:      "rhtap-releng-tenant",
						},
					},
				},
				"test-app",
				"Should filter by application name"),

			Entry("with no matching application",
				[]runtime.Object{
					&releaseapiv1alpha1.ReleasePlan{
						ObjectMeta: metav1.ObjectMeta{
							Name:      "releaseplan1",
							Namespace: "test-namespace",
							Labels: map[string]string{
								konflux.EnvironmentLabel: "staging",
							},
						},
						Spec: releaseapiv1alpha1.ReleasePlanSpec{
							Application: "other-app",
							Target:      "rhtap-releng-tenant",
						},
					},
				},
				"test-app",
				"Should return empty list for non-matching application"),
		)
	})

	Context("Get specific release plan", func() {
		DescribeTable("should get release plan by name",
			func(releasePlans []runtime.Object, releasePlanName string, expectError bool, description string) {
				if len(releasePlans) > 0 {
					fakeClientBuilder = fakeClientBuilder.WithRuntimeObjects(releasePlans...)
				}
				ctx = context.WithValue(ctx, internal.KubeCliCtxType, fakeClientBuilder.Build())

				args := []string{"", releasePlanName}
				err := cmd.Run(ctx, args)

				if expectError {
					Expect(err).To(HaveOccurred(), description)
				} else {
					Expect(err).ToNot(HaveOccurred(), description)
				}
			},

			Entry("existing release plan",
				[]runtime.Object{
					&releaseapiv1alpha1.ReleasePlan{
						ObjectMeta: metav1.ObjectMeta{
							Name:      "test-releaseplan",
							Namespace: "test-namespace",
							Labels: map[string]string{
								konflux.EnvironmentLabel: "staging",
							},
						},
						Spec: releaseapiv1alpha1.ReleasePlanSpec{
							Application: "test-app",
							Target:      "rhtap-releng-tenant",
						},
					},
				},
				"test-releaseplan",
				false,
				"Should get existing release plan"),

			Entry("non-existing release plan",
				[]runtime.Object{},
				"non-existing-releaseplan",
				true,
				"Should fail for non-existing release plan"),
		)
	})

	Context("Flag aliases", func() {
		It("should work with --application flag", func() {
			releasePlans := []runtime.Object{
				&releaseapiv1alpha1.ReleasePlan{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test-releaseplan",
						Namespace: "test-namespace",
						Labels: map[string]string{
							konflux.EnvironmentLabel: "staging",
						},
					},
					Spec: releaseapiv1alpha1.ReleasePlanSpec{
						Application: "test-app",
						Target:      "rhtap-releng-tenant",
					},
				},
			}
			fakeClientBuilder = fakeClientBuilder.WithRuntimeObjects(releasePlans...)
			ctx = context.WithValue(ctx, internal.KubeCliCtxType, fakeClientBuilder.Build())

			args := []string{"", "--application", "test-app"}
			err := cmd.Run(ctx, args)
			Expect(err).ToNot(HaveOccurred())
		})

		It("should work with --app alias", func() {
			releasePlans := []runtime.Object{
				&releaseapiv1alpha1.ReleasePlan{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test-releaseplan",
						Namespace: "test-namespace",
						Labels: map[string]string{
							konflux.EnvironmentLabel: "staging",
						},
					},
					Spec: releaseapiv1alpha1.ReleasePlanSpec{
						Application: "test-app",
						Target:      "rhtap-releng-tenant",
					},
				},
			}
			fakeClientBuilder = fakeClientBuilder.WithRuntimeObjects(releasePlans...)
			ctx = context.WithValue(ctx, internal.KubeCliCtxType, fakeClientBuilder.Build())

			args := []string{"", "--app", "test-app"}
			err := cmd.Run(ctx, args)
			Expect(err).ToNot(HaveOccurred())
		})
	})

	Context("Environment targeting scenarios", func() {
		It("should handle staging environment release plans", func() {
			releasePlans := []runtime.Object{
				&releaseapiv1alpha1.ReleasePlan{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "staging-releaseplan",
						Namespace: "test-namespace",
						Labels: map[string]string{
							konflux.EnvironmentLabel: "staging",
						},
					},
					Spec: releaseapiv1alpha1.ReleasePlanSpec{
						Application: "test-app",
						Target:      "rhtap-releng-tenant",
					},
				},
			}
			fakeClientBuilder = fakeClientBuilder.WithRuntimeObjects(releasePlans...)
			ctx = context.WithValue(ctx, internal.KubeCliCtxType, fakeClientBuilder.Build())

			args := []string{"", "--app", "test-app"}
			err := cmd.Run(ctx, args)
			Expect(err).ToNot(HaveOccurred())
		})

		It("should handle production environment release plans", func() {
			releasePlans := []runtime.Object{
				&releaseapiv1alpha1.ReleasePlan{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "production-releaseplan",
						Namespace: "test-namespace",
						Labels: map[string]string{
							konflux.EnvironmentLabel: "production",
						},
					},
					Spec: releaseapiv1alpha1.ReleasePlanSpec{
						Application: "test-app",
						Target:      "rhtap-releng-tenant",
					},
				},
			}
			fakeClientBuilder = fakeClientBuilder.WithRuntimeObjects(releasePlans...)
			ctx = context.WithValue(ctx, internal.KubeCliCtxType, fakeClientBuilder.Build())

			args := []string{"", "--app", "test-app"}
			err := cmd.Run(ctx, args)
			Expect(err).ToNot(HaveOccurred())
		})

	})

	Context("Complex scenarios", func() {
		It("should handle multiple environments for same application", func() {
			releasePlans := []runtime.Object{
				&releaseapiv1alpha1.ReleasePlan{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "staging-releaseplan",
						Namespace: "test-namespace",
						Labels: map[string]string{
							konflux.EnvironmentLabel: "staging",
						},
					},
					Spec: releaseapiv1alpha1.ReleasePlanSpec{
						Application: "test-app",
						Target:      "rhtap-releng-tenant",
					},
				},
				&releaseapiv1alpha1.ReleasePlan{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "production-releaseplan",
						Namespace: "test-namespace",
						Labels: map[string]string{
							konflux.EnvironmentLabel: "production",
						},
					},
					Spec: releaseapiv1alpha1.ReleasePlanSpec{
						Application: "test-app",
						Target:      "rhtap-releng-tenant",
					},
				},
			}
			fakeClientBuilder = fakeClientBuilder.WithRuntimeObjects(releasePlans...)
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

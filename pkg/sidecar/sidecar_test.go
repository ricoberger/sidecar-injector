package sidecar

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
)

var _ = Describe("Sidecar", func() {
	Context("Creating and updating Pods", func() {
		It("Should inject sidecar into Pods which are matching selector from injector configuration", func() {
			By("Create Pod")
			err := k8sClient.Create(ctx, &corev1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-pod-1",
					Namespace: "default",
					Labels: map[string]string{
						"sidecar-injector": "injector-test",
					},
				},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Name:            "my-container",
							Image:           "my-image",
							ImagePullPolicy: corev1.PullIfNotPresent,
							Resources: corev1.ResourceRequirements{
								Requests: corev1.ResourceList{
									"cpu":    resource.MustParse("150m"),
									"memory": resource.MustParse("150Mi"),
								},
								Limits: corev1.ResourceList{
									"cpu":    resource.MustParse("250m"),
									"memory": resource.MustParse("250Mi"),
								},
							},
						},
					},
				},
			})
			Expect(err).NotTo(HaveOccurred())

			pod := &corev1.Pod{}
			err = k8sClient.Get(ctx, types.NamespacedName{Name: "test-pod-1", Namespace: "default"}, pod)
			Expect(err).NotTo(HaveOccurred())

			Expect(pod.Annotations[annotationStatusKey]).To(Equal("injected"))
			Expect(len(pod.Spec.InitContainers)).To(Equal(1))
			Expect(len(pod.Spec.Containers)).To(Equal(2))
			Expect(len(pod.Spec.Volumes)).To(Equal(1))
			Expect(pod.Spec.Containers[1].Name).To(Equal("test-container"))
			Expect(pod.Spec.Containers[1].Resources.Requests["cpu"]).To(Equal(resource.MustParse("100m")))
			Expect(pod.Spec.Containers[1].Resources.Requests["memory"]).To(Equal(resource.MustParse("100Mi")))
			Expect(pod.Spec.Containers[1].Resources.Limits["cpu"]).To(Equal(resource.MustParse("200m")))
			Expect(pod.Spec.Containers[1].Resources.Limits["memory"]).To(Equal(resource.MustParse("200Mi")))
			Expect(len(pod.Spec.Containers[1].Env)).To(Equal(0))

			By("Update Pod")
			podWithSidecar := &corev1.Pod{}
			err = k8sClient.Get(ctx, types.NamespacedName{Name: "test-pod-1", Namespace: "default"}, podWithSidecar)
			Expect(err).NotTo(HaveOccurred())

			podWithSidecar.Labels["newlabel"] = "newlabel"
			err = k8sClient.Update(ctx, podWithSidecar)
			Expect(err).NotTo(HaveOccurred())

			podUpdated := &corev1.Pod{}
			err = k8sClient.Get(ctx, types.NamespacedName{Name: "test-pod-1", Namespace: "default"}, podUpdated)
			Expect(err).NotTo(HaveOccurred())

			Expect(podUpdated.Annotations[annotationStatusKey]).To(Equal("injected"))
			Expect(len(podUpdated.Spec.InitContainers)).To(Equal(1))
			Expect(len(podUpdated.Spec.Containers)).To(Equal(2))
			Expect(len(podUpdated.Spec.Volumes)).To(Equal(1))
			Expect(podUpdated.Spec.Containers[1].Name).To(Equal("test-container"))
			Expect(podUpdated.Spec.Containers[1].Resources.Requests["cpu"]).To(Equal(resource.MustParse("100m")))
			Expect(podUpdated.Spec.Containers[1].Resources.Requests["memory"]).To(Equal(resource.MustParse("100Mi")))
			Expect(podUpdated.Spec.Containers[1].Resources.Limits["cpu"]).To(Equal(resource.MustParse("200m")))
			Expect(podUpdated.Spec.Containers[1].Resources.Limits["memory"]).To(Equal(resource.MustParse("200Mi")))
			Expect(len(podUpdated.Spec.Containers[1].Env)).To(Equal(0))
		})

		It("Should inject sidecar into Pods which have sidecar-injector annotation", func() {
			By("Create Pod")
			err := k8sClient.Create(ctx, &corev1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-pod-2",
					Namespace: "default",
					Annotations: map[string]string{
						annotationInjectKey:         "enabled",
						annotationContainersKey:     "test-container",
						annotationInitContainersKey: "test-initcontainer",
						annotationVolumesKey:        "test-volume",
					},
				},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Name:            "my-container",
							Image:           "my-image",
							ImagePullPolicy: corev1.PullIfNotPresent,
							Resources: corev1.ResourceRequirements{
								Requests: corev1.ResourceList{
									"cpu":    resource.MustParse("150m"),
									"memory": resource.MustParse("150Mi"),
								},
								Limits: corev1.ResourceList{
									"cpu":    resource.MustParse("250m"),
									"memory": resource.MustParse("250Mi"),
								},
							},
						},
					},
				},
			})
			Expect(err).NotTo(HaveOccurred())

			pod := &corev1.Pod{}
			err = k8sClient.Get(ctx, types.NamespacedName{Name: "test-pod-2", Namespace: "default"}, pod)
			Expect(err).NotTo(HaveOccurred())

			Expect(pod.Annotations[annotationStatusKey]).To(Equal("injected"))
			Expect(len(pod.Spec.InitContainers)).To(Equal(1))
			Expect(len(pod.Spec.Containers)).To(Equal(2))
			Expect(len(pod.Spec.Volumes)).To(Equal(1))
			Expect(pod.Spec.Containers[1].Name).To(Equal("test-container"))
			Expect(pod.Spec.Containers[1].Resources.Requests["cpu"]).To(Equal(resource.MustParse("100m")))
			Expect(pod.Spec.Containers[1].Resources.Requests["memory"]).To(Equal(resource.MustParse("100Mi")))
			Expect(pod.Spec.Containers[1].Resources.Limits["cpu"]).To(Equal(resource.MustParse("200m")))
			Expect(pod.Spec.Containers[1].Resources.Limits["memory"]).To(Equal(resource.MustParse("200Mi")))
			Expect(len(pod.Spec.Containers[1].Env)).To(Equal(0))
		})

		It("Should inject sidecar into Pods with updated resources", func() {
			By("Create Pod")
			err := k8sClient.Create(ctx, &corev1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-pod-3",
					Namespace: "default",
					Annotations: map[string]string{
						annotationInjectKey:     "enabled",
						annotationContainersKey: "test-container",
						"sidecar-injector.ricoberger.de/containers-test-container-cpurequests":    "200m",
						"sidecar-injector.ricoberger.de/containers-test-container-cpulimits":      "300m",
						"sidecar-injector.ricoberger.de/containers-test-container-memoryrequests": "200Mi",
						"sidecar-injector.ricoberger.de/containers-test-container-memorylimits":   "300Mi",
					},
				},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Name:            "my-container",
							Image:           "my-image",
							ImagePullPolicy: corev1.PullIfNotPresent,
							Resources: corev1.ResourceRequirements{
								Requests: corev1.ResourceList{
									"cpu":    resource.MustParse("150m"),
									"memory": resource.MustParse("150Mi"),
								},
								Limits: corev1.ResourceList{
									"cpu":    resource.MustParse("250m"),
									"memory": resource.MustParse("250Mi"),
								},
							},
						},
					},
				},
			})
			Expect(err).NotTo(HaveOccurred())

			pod := &corev1.Pod{}
			err = k8sClient.Get(ctx, types.NamespacedName{Name: "test-pod-3", Namespace: "default"}, pod)
			Expect(err).NotTo(HaveOccurred())

			Expect(pod.Annotations[annotationStatusKey]).To(Equal("injected"))
			Expect(len(pod.Spec.InitContainers)).To(Equal(0))
			Expect(len(pod.Spec.Containers)).To(Equal(2))
			Expect(len(pod.Spec.Volumes)).To(Equal(0))
			Expect(pod.Spec.Containers[1].Name).To(Equal("test-container"))
			Expect(pod.Spec.Containers[1].Resources.Requests["cpu"]).To(Equal(resource.MustParse("200m")))
			Expect(pod.Spec.Containers[1].Resources.Requests["memory"]).To(Equal(resource.MustParse("200Mi")))
			Expect(pod.Spec.Containers[1].Resources.Limits["cpu"]).To(Equal(resource.MustParse("300m")))
			Expect(pod.Spec.Containers[1].Resources.Limits["memory"]).To(Equal(resource.MustParse("300Mi")))
			Expect(len(pod.Spec.Containers[1].Env)).To(Equal(0))
		})

		It("Should inject sidecar into Pods with additional environment variables", func() {
			By("Create Pod")
			err := k8sClient.Create(ctx, &corev1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-pod-4",
					Namespace: "default",
					Annotations: map[string]string{
						annotationInjectKey:                           "enabled",
						annotationContainersKey:                       "test-container",
						"sidecar-injector.ricoberger.de/test-env-var": "test-env-var-value",
					},
				},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Name:            "my-container",
							Image:           "my-image",
							ImagePullPolicy: corev1.PullIfNotPresent,
							Resources: corev1.ResourceRequirements{
								Requests: corev1.ResourceList{
									"cpu":    resource.MustParse("150m"),
									"memory": resource.MustParse("150Mi"),
								},
								Limits: corev1.ResourceList{
									"cpu":    resource.MustParse("250m"),
									"memory": resource.MustParse("250Mi"),
								},
							},
						},
					},
				},
			})
			Expect(err).NotTo(HaveOccurred())

			pod := &corev1.Pod{}
			err = k8sClient.Get(ctx, types.NamespacedName{Name: "test-pod-4", Namespace: "default"}, pod)
			Expect(err).NotTo(HaveOccurred())

			Expect(pod.Annotations[annotationStatusKey]).To(Equal("injected"))
			Expect(len(pod.Spec.InitContainers)).To(Equal(0))
			Expect(len(pod.Spec.Containers)).To(Equal(2))
			Expect(len(pod.Spec.Volumes)).To(Equal(0))
			Expect(pod.Spec.Containers[1].Name).To(Equal("test-container"))
			Expect(pod.Spec.Containers[1].Resources.Requests["cpu"]).To(Equal(resource.MustParse("100m")))
			Expect(pod.Spec.Containers[1].Resources.Requests["memory"]).To(Equal(resource.MustParse("100Mi")))
			Expect(pod.Spec.Containers[1].Resources.Limits["cpu"]).To(Equal(resource.MustParse("200m")))
			Expect(pod.Spec.Containers[1].Resources.Limits["memory"]).To(Equal(resource.MustParse("200Mi")))
			Expect(len(pod.Spec.Containers[1].Env)).To(Equal(1))
		})

		It("Should fail when container name is invalid", func() {
			By("Create Pod")
			err := k8sClient.Create(ctx, &corev1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-pod-5",
					Namespace: "default",
					Annotations: map[string]string{
						annotationInjectKey:     "enabled",
						annotationContainersKey: "test-container-invalid",
					},
				},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Name:            "my-container",
							Image:           "my-image",
							ImagePullPolicy: corev1.PullIfNotPresent,
							Resources: corev1.ResourceRequirements{
								Requests: corev1.ResourceList{
									"cpu":    resource.MustParse("150m"),
									"memory": resource.MustParse("150Mi"),
								},
								Limits: corev1.ResourceList{
									"cpu":    resource.MustParse("250m"),
									"memory": resource.MustParse("250Mi"),
								},
							},
						},
					},
				},
			})
			Expect(err).To(HaveOccurred())
		})

		It("Should fail when init container name is invalid", func() {
			By("Create Pod")
			err := k8sClient.Create(ctx, &corev1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-pod-5",
					Namespace: "default",
					Annotations: map[string]string{
						annotationInjectKey:         "enabled",
						annotationInitContainersKey: "test-container-invalid",
					},
				},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Name:            "my-container",
							Image:           "my-image",
							ImagePullPolicy: corev1.PullIfNotPresent,
							Resources: corev1.ResourceRequirements{
								Requests: corev1.ResourceList{
									"cpu":    resource.MustParse("150m"),
									"memory": resource.MustParse("150Mi"),
								},
								Limits: corev1.ResourceList{
									"cpu":    resource.MustParse("250m"),
									"memory": resource.MustParse("250Mi"),
								},
							},
						},
					},
				},
			})
			Expect(err).To(HaveOccurred())
		})

		It("Should fail when volume name is invalid", func() {
			By("Create Pod")
			err := k8sClient.Create(ctx, &corev1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-pod-5",
					Namespace: "default",
					Annotations: map[string]string{
						annotationInjectKey:  "enabled",
						annotationVolumesKey: "test-volume-invalid",
					},
				},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Name:            "my-container",
							Image:           "my-image",
							ImagePullPolicy: corev1.PullIfNotPresent,
							Resources: corev1.ResourceRequirements{
								Requests: corev1.ResourceList{
									"cpu":    resource.MustParse("150m"),
									"memory": resource.MustParse("150Mi"),
								},
								Limits: corev1.ResourceList{
									"cpu":    resource.MustParse("250m"),
									"memory": resource.MustParse("250Mi"),
								},
							},
						},
					},
				},
			})
			Expect(err).To(HaveOccurred())
		})

		It("Should inject sidecar into Pods and ignore invalid resource annotations", func() {
			By("Create Pod")
			err := k8sClient.Create(ctx, &corev1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-pod-6",
					Namespace: "default",
					Annotations: map[string]string{
						annotationInjectKey:     "enabled",
						annotationContainersKey: "test-container",
						"sidecar-injector.ricoberger.de/containers-test-container-cpurequests":    "invalid",
						"sidecar-injector.ricoberger.de/containers-test-container-cpulimits":      "invalid",
						"sidecar-injector.ricoberger.de/containers-test-container-memoryrequests": "invalid",
						"sidecar-injector.ricoberger.de/containers-test-container-memorylimits":   "invalid",
					},
				},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Name:            "my-container",
							Image:           "my-image",
							ImagePullPolicy: corev1.PullIfNotPresent,
							Resources: corev1.ResourceRequirements{
								Requests: corev1.ResourceList{
									"cpu":    resource.MustParse("150m"),
									"memory": resource.MustParse("150Mi"),
								},
								Limits: corev1.ResourceList{
									"cpu":    resource.MustParse("250m"),
									"memory": resource.MustParse("250Mi"),
								},
							},
						},
					},
				},
			})
			Expect(err).NotTo(HaveOccurred())

			pod := &corev1.Pod{}
			err = k8sClient.Get(ctx, types.NamespacedName{Name: "test-pod-6", Namespace: "default"}, pod)
			Expect(err).NotTo(HaveOccurred())

			Expect(pod.Annotations[annotationStatusKey]).To(Equal("injected"))
			Expect(len(pod.Spec.InitContainers)).To(Equal(0))
			Expect(len(pod.Spec.Containers)).To(Equal(2))
			Expect(len(pod.Spec.Volumes)).To(Equal(0))
			Expect(pod.Spec.Containers[1].Name).To(Equal("test-container"))
			Expect(pod.Spec.Containers[1].Resources.Requests["cpu"]).To(Equal(resource.MustParse("100m")))
			Expect(pod.Spec.Containers[1].Resources.Requests["memory"]).To(Equal(resource.MustParse("100Mi")))
			Expect(pod.Spec.Containers[1].Resources.Limits["cpu"]).To(Equal(resource.MustParse("200m")))
			Expect(pod.Spec.Containers[1].Resources.Limits["memory"]).To(Equal(resource.MustParse("200Mi")))
			Expect(len(pod.Spec.Containers[1].Env)).To(Equal(0))
		})

		It("Should do nothing if pod does not require injection", func() {
			By("Create Pod")
			err := k8sClient.Create(ctx, &corev1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-pod-7",
					Namespace: "default",
				},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Name:            "my-container",
							Image:           "my-image",
							ImagePullPolicy: corev1.PullIfNotPresent,
							Resources: corev1.ResourceRequirements{
								Requests: corev1.ResourceList{
									"cpu":    resource.MustParse("150m"),
									"memory": resource.MustParse("150Mi"),
								},
								Limits: corev1.ResourceList{
									"cpu":    resource.MustParse("250m"),
									"memory": resource.MustParse("250Mi"),
								},
							},
						},
					},
				},
			})
			Expect(err).NotTo(HaveOccurred())

			pod := &corev1.Pod{}
			err = k8sClient.Get(ctx, types.NamespacedName{Name: "test-pod-7", Namespace: "default"}, pod)
			Expect(err).NotTo(HaveOccurred())

			Expect(pod.Annotations).To(BeNil())
			Expect(len(pod.Spec.InitContainers)).To(Equal(0))
			Expect(len(pod.Spec.Containers)).To(Equal(1))
			Expect(len(pod.Spec.Volumes)).To(Equal(0))
		})
	})
})

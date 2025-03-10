package sidecar

import (
	"context"
	"crypto/tls"
	"fmt"
	"net"
	"testing"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	admissionv1 "k8s.io/api/admissionregistration/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	apimachineryruntime "k8s.io/apimachinery/pkg/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/envtest"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
	metricsserver "sigs.k8s.io/controller-runtime/pkg/metrics/server"
	"sigs.k8s.io/controller-runtime/pkg/webhook"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
)

var cfg *rest.Config
var k8sClient client.Client
var testEnv *envtest.Environment
var ctx context.Context
var cancel context.CancelFunc

func TestWebhook(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Webhook Suite")
}

var _ = BeforeSuite(func() {
	logf.SetLogger(zap.New(zap.WriteTo(GinkgoWriter), zap.UseDevMode(true)))

	ctx, cancel = context.WithCancel(context.TODO())

	By("Bootstrapping test environment")

	failurePolicy := admissionv1.Fail
	path := "/mutate"
	sideEffects := admissionv1.SideEffectClassNone

	testEnv = &envtest.Environment{
		WebhookInstallOptions: envtest.WebhookInstallOptions{
			MutatingWebhooks: []*admissionv1.MutatingWebhookConfiguration{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name: "sidecar-injector",
					},
					TypeMeta: metav1.TypeMeta{
						Kind:       "MutatingWebhookConfiguration",
						APIVersion: "admissionregistration.k8s.io/v1",
					},
					Webhooks: []admissionv1.MutatingWebhook{
						{
							Name:                    "sidecar-injector.ricoberger.de",
							AdmissionReviewVersions: []string{"v1"},
							FailurePolicy:           &failurePolicy,
							ClientConfig: admissionv1.WebhookClientConfig{
								Service: &admissionv1.ServiceReference{
									Path: &path,
								},
							},
							Rules: []admissionv1.RuleWithOperations{
								{
									Operations: []admissionv1.OperationType{
										admissionv1.Create,
										admissionv1.Update,
									},
									Rule: admissionv1.Rule{
										APIGroups:   []string{""},
										APIVersions: []string{"v1"},
										Resources:   []string{"pods"},
									},
								},
							},
							SideEffects: &sideEffects,
						},
					},
				},
			},
		},
	}

	var err error

	cfg, err = testEnv.Start()
	Expect(err).NotTo(HaveOccurred())
	Expect(cfg).NotTo(BeNil())

	scheme := apimachineryruntime.NewScheme()
	err = clientgoscheme.AddToScheme(scheme)
	Expect(err).NotTo(HaveOccurred())

	err = admissionv1.AddToScheme(scheme)
	Expect(err).NotTo(HaveOccurred())

	k8sClient, err = client.New(cfg, client.Options{Scheme: scheme})
	Expect(err).NotTo(HaveOccurred())
	Expect(k8sClient).NotTo(BeNil())

	webhookInstallOptions := &testEnv.WebhookInstallOptions
	mgr, err := ctrl.NewManager(cfg, ctrl.Options{
		Scheme: scheme,
		WebhookServer: webhook.NewServer(webhook.Options{
			Host:    webhookInstallOptions.LocalServingHost,
			Port:    webhookInstallOptions.LocalServingPort,
			CertDir: webhookInstallOptions.LocalServingCertDir,
		}),
		LeaderElection: false,
		Metrics:        metricsserver.Options{BindAddress: "0"},
	})
	Expect(err).NotTo(HaveOccurred())

	mgr.GetWebhookServer().Register("/mutate", &webhook.Admission{
		Handler: &Injector{
			Client: mgr.GetClient(),
			Config: &Config{
				Injectors: []InjectorData{
					{
						Selector: metav1.LabelSelector{
							MatchLabels: map[string]string{
								"sidecar-injector": "injector-test",
							},
						},
						Containers:     []string{"test-container"},
						InitContainers: []string{"test-initcontainer"},
						Volumes:        []string{"test-volume"},
					},
				},
				Containers: []corev1.Container{
					{
						Name:            "test-container",
						Image:           "test-image",
						ImagePullPolicy: corev1.PullIfNotPresent,
						Resources: corev1.ResourceRequirements{
							Requests: corev1.ResourceList{
								"cpu":    resource.MustParse("100m"),
								"memory": resource.MustParse("100Mi"),
							},
							Limits: corev1.ResourceList{
								"cpu":    resource.MustParse("200m"),
								"memory": resource.MustParse("200Mi"),
							},
						},
					},
					{
						Name:            "test-initcontainer",
						Image:           "test-initimage",
						ImagePullPolicy: corev1.PullIfNotPresent,
						Resources: corev1.ResourceRequirements{
							Requests: corev1.ResourceList{
								"cpu":    resource.MustParse("50m"),
								"memory": resource.MustParse("50Mi"),
							},
							Limits: corev1.ResourceList{
								"cpu":    resource.MustParse("50m"),
								"memory": resource.MustParse("50Mi"),
							},
						},
					},
				},
				Volumes: []corev1.Volume{
					{
						Name: "test-volume",
						VolumeSource: corev1.VolumeSource{
							Secret: &corev1.SecretVolumeSource{
								SecretName: "secret-config",
							},
						},
					},
				},
				EnvironmentVariables: []EnvironmentVariable{
					{
						Name:       "test-env-var",
						Container:  "test-container",
						Annotation: "sidecar-injector.ricoberger.de/test-env-var",
					},
				},
			},
			Decoder: admission.NewDecoder(mgr.GetScheme()),
		},
	})

	go func() {
		defer GinkgoRecover()
		err = mgr.Start(ctx)
		Expect(err).NotTo(HaveOccurred())
	}()

	dialer := &net.Dialer{Timeout: time.Second}
	addrPort := fmt.Sprintf("%s:%d", webhookInstallOptions.LocalServingHost, webhookInstallOptions.LocalServingPort)
	Eventually(func() error {
		//nolint:gosec
		conn, err := tls.DialWithDialer(dialer, "tcp", addrPort, &tls.Config{InsecureSkipVerify: true})
		if err != nil {
			return err
		}
		return conn.Close()
	}).Should(Succeed())
})

var _ = AfterSuite(func() {
	cancel()
	By("Tearing down the test environment")
	err := testEnv.Stop()
	Expect(err).NotTo(HaveOccurred())
})

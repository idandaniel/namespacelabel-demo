package controllers

import (
	"context"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	idandanielv1 "idandaniel.io/namespacelabel-demo/api/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

var _ = Describe("NamespaceLabel controller test", func() {

	const (
		NamespaceLabelName = "test-namepsacelabel"
		Namespace          = "default"
	)

	ctx := context.Background()

	managementLabels := map[string]string{
		"app.kubernetes.io/name": NamespaceLabelName,
	}

	namespaceLabelLookupKey := types.NamespacedName{Name: NamespaceLabelName, Namespace: Namespace}

	BeforeEach(func() {
		namespace := &corev1.Namespace{
			ObjectMeta: metav1.ObjectMeta{
				Name:   Namespace,
				Labels: managementLabels,
			},
		}
		Expect(k8sClient.Update(ctx, namespace)).NotTo(HaveOccurred())
	})

	AfterEach(func() {
		namespace := &corev1.Namespace{
			ObjectMeta: metav1.ObjectMeta{
				Name:      Namespace,
				Namespace: Namespace,
				Labels:    managementLabels,
			},
		}
		Expect(k8sClient.Update(ctx, namespace)).NotTo(HaveOccurred())
	})

	Context("When creating NamespaceLabel Label", func() {
		It("Should sync the Labels with the Namespace Labels.", func() {

			By("Creating the custom resource for the Kind NamespaceLabel")
			namespaceLabel := &idandanielv1.NamespaceLabel{
				ObjectMeta: metav1.ObjectMeta{
					Name:      NamespaceLabelName,
					Namespace: Namespace,
				},
				Spec: idandanielv1.NamespaceLabelSpec{
					Labels: map[string]string{
						"key_1": "value_1",
					},
				},
			}
			Expect(k8sClient.Create(ctx, namespaceLabel)).To(Not(HaveOccurred()))

			By("Checking if the custom resource was successfully created")
			Eventually(func() error {
				found := &idandanielv1.NamespaceLabel{}
				return k8sClient.Get(ctx, namespaceLabelLookupKey, found)
			}, time.Minute, time.Second).Should(Succeed())

			By("Reconciling the custom resource created")
			namespaceLabelReconciler := &NamespaceLabelReconciler{
				Client: k8sClient,
				Scheme: k8sClient.Scheme(),
			}
			_, err := namespaceLabelReconciler.Reconcile(ctx, reconcile.Request{NamespacedName: namespaceLabelLookupKey})
			Expect(err).To(Not(HaveOccurred()))

			By("Ensuring Namespace management Labels still exist")
			Eventually(func() bool {
				changedNamespace := &corev1.Namespace{}
				Expect(k8sClient.Get(ctx, types.NamespacedName{Name: Namespace}, changedNamespace)).To(Not(HaveOccurred()))

				currentLabels := changedNamespace.GetLabels()

				for k := range managementLabels {
					_, exists := currentLabels[k]
					if !exists {
						return false
					}
				}

				return true
			}, time.Second*15, time.Second).Should(BeTrue())

			By("Ensuring NamespaceLabel's Labels were added")
			Eventually(func() bool {
				changedNamespace := &corev1.Namespace{}
				Expect(k8sClient.Get(ctx, types.NamespacedName{Name: Namespace}, changedNamespace)).To(Not(HaveOccurred()))

				currentLabels := changedNamespace.GetLabels()
				newLabels := namespaceLabel.Spec.Labels

				for k, v := range newLabels {
					GinkgoWriter.Println(currentLabels[k] + " ---> " + currentLabels[v])
					GinkgoWriter.Println(k + " ---> " + v)
				}

				return true
			}, time.Second*15, time.Second).Should(BeTrue())
		})
	})
})

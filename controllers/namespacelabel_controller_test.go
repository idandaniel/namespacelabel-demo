package controllers

import (
	"context"
	"math/rand"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	idandanielv1 "idandaniel.io/namespacelabel-demo/api/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

func createNamespace(ctx context.Context, name string, labels map[string]string) {
	namespace := &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name:   name,
			Labels: labels,
		},
	}
	Expect(k8sClient.Create(ctx, namespace)).NotTo(HaveOccurred())
}

func ensureLabelsExists(ctx context.Context, namespace string, labels map[string]string) {
	Eventually(func() bool {
		logf.Log.Info("Ensurng labels")
		changedNamespace := &corev1.Namespace{}
		Expect(k8sClient.Get(ctx, types.NamespacedName{Name: namespace}, changedNamespace)).To(Not(HaveOccurred()))

		currentLabels := changedNamespace.GetLabels()

		for k, v := range labels {
			logf.Log.Info(k + " ---> " + v + ": [ " + currentLabels[k] + " ]")
			val, exists := currentLabels[k]
			if !exists || (exists && val != v) {
				return false
			}
		}

		return true
	}, time.Second*10, time.Millisecond*250).Should(BeTrue())
}

var _ = Describe("NamespaceLabel controller test", func() {

	NamespaceLabelName := "test-nl"
	Namespace := RandStringRunes(12)

	ctx := context.Background()

	managementLabels := map[string]string{
		"app.kubernetes.io/name": NamespaceLabelName,
	}

	namespaceLabelLookupKey := types.NamespacedName{Name: NamespaceLabelName, Namespace: Namespace}

	BeforeEach(func() {
		createNamespace(ctx, Namespace, managementLabels)
		k8sClient.DeleteAllOf(ctx, &idandanielv1.NamespaceLabel{})
	})

	AfterEach(func() {
		namespace := &corev1.Namespace{}
		Expect(k8sClient.Get(ctx, types.NamespacedName{Name: Namespace}, namespace)).ToNot(HaveOccurred())
		Expect(k8sClient.Delete(ctx, namespace)).NotTo(HaveOccurred())
		k8sClient.DeleteAllOf(ctx, &idandanielv1.NamespaceLabel{})
	})

	Context("When creating NamespaceLabel Label", func() {
		It("Should sync the Labels with the Namespace Labels.", func() {

			By("Creating the custom resource for the Kind NamespaceLabel")
			newLabels := map[string]string{
				"key_1": "value_1",
			}
			namespaceLabel := &idandanielv1.NamespaceLabel{
				ObjectMeta: metav1.ObjectMeta{
					Name:      NamespaceLabelName,
					Namespace: Namespace,
				},
				Spec: idandanielv1.NamespaceLabelSpec{
					Labels: newLabels,
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
			ensureLabelsExists(ctx, Namespace, managementLabels)

			By("Ensuring NamespaceLabel's Labels were added")
			ensureLabelsExists(ctx, Namespace, newLabels)
		})
	})
})

func init() {
	rand.Seed(time.Now().UnixNano())
}

var letterRunes = []rune("abcdefghijklmnopqrstuvwxyz123456789")

func RandStringRunes(n int) string {
	b := make([]rune, n)
	for i := range b {
		b[i] = letterRunes[rand.Intn(len(letterRunes))]
	}
	return string(b)
}

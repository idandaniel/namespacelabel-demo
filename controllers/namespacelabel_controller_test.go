package controllers

import (
	"context"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	idandanielv1 "idandaniel.io/namespacelabel-demo/api/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

var _ = Describe("NamespaceLabel controller test", func() {

	const (
		NamespaceLabelName = "test-namepsacelabel"
	)

	ctx := context.Background()

	managementLabels := map[string]string{
		"app.kubernetes.io/name": NamespaceLabelName,
	}

	namespace := &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name:   NamespaceLabelName,
			Labels: managementLabels,
		},
	}

	namespaceLabel := &idandanielv1.NamespaceLabel{
		ObjectMeta: metav1.ObjectMeta{
			Name:      NamespaceLabelName,
			Namespace: namespace.Name,
		},
		Spec: idandanielv1.NamespaceLabelSpec{
			Labels: map[string]string{
				"key_1": "value_1",
			},
		},
	}

	namespaceLabelLookupKey := types.NamespacedName{Name: NamespaceLabelName, Namespace: NamespaceLabelName}

	BeforeEach(func() {
		By("Creating the Namespace to perform the tests")
		err := k8sClient.Create(ctx, namespace)
		Expect(err).To(Not(HaveOccurred()))
	})

	AfterEach(func() {
		By("Deleting the Namespace to perform the tests")
		_ = k8sClient.Delete(ctx, namespace)
	})

	Context("When creating NamespaceLabel Label", func() {
		It("Should sync the Labels with the Namespace Labels.", func() {

			By("Creating the custom resource for the Kind NamespaceLabel")
			existingNamespaceLabel := &idandanielv1.NamespaceLabel{}
			err := k8sClient.Get(ctx, namespaceLabelLookupKey, existingNamespaceLabel)
			if err != nil && errors.IsNotFound(err) {
				err = k8sClient.Create(ctx, namespaceLabel)
				Expect(err).To(Not(HaveOccurred()))
			}

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
			_, err = namespaceLabelReconciler.Reconcile(ctx, reconcile.Request{NamespacedName: namespaceLabelLookupKey})
			Expect(err).To(Not(HaveOccurred()))

			By("Ensuring Namespace management Labels still exist")
			Eventually(func() bool {
				changedNamespace := &corev1.Namespace{}
				key := types.NamespacedName{Name: namespaceLabelLookupKey.Name, Namespace: namespaceLabelLookupKey.Namespace}
				err := k8sClient.Get(ctx, key, changedNamespace)
				Expect(err).To(Not(HaveOccurred()))

				currentLabels := changedNamespace.GetLabels()

				for k, v := range managementLabels {
					val, exists := currentLabels[k]
					if val != v || exists == false {
						return false
					}
				}

				return true
			}, time.Minute, time.Second).Should(BeTrue())

			By("Ensuring NamespaceLabel's Labels were added")
			Eventually(func() bool {
				changedNamespace := &corev1.Namespace{}
				key := types.NamespacedName{Name: namespaceLabelLookupKey.Name, Namespace: namespaceLabelLookupKey.Namespace}
				err := k8sClient.Get(ctx, key, changedNamespace)
				Expect(err).To(Not(HaveOccurred()))

				currentLabels := changedNamespace.GetLabels()
				newLabels := namespaceLabel.GetLabels()

				for k, v := range newLabels {
					val, exists := currentLabels[k]
					if val != v || exists == false {
						return false
					}
				}

				return true
			}, time.Minute, time.Second).Should(BeTrue())
		})
	})

})

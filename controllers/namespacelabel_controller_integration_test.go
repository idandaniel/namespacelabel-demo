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
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

const (
	Duration = time.Second * 10
	Interval = time.Millisecond * 250
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

func deleteNamespace(ctx context.Context, namespace string) {
	n := &corev1.Namespace{}
	Expect(k8sClient.Get(ctx, types.NamespacedName{Name: namespace}, n)).ToNot(HaveOccurred())
	Expect(k8sClient.Delete(ctx, n)).NotTo(HaveOccurred())
}

func startReconcile(ctx context.Context, request reconcile.Request) {
	namespaceLabelReconciler := &NamespaceLabelReconciler{
		Client: k8sClient,
		Scheme: k8sClient.Scheme(),
	}
	_, err := namespaceLabelReconciler.Reconcile(ctx, request)
	Expect(err).To(Not(HaveOccurred()))
}

func ensureLabelsExists(ctx context.Context, namespace string, labels map[string]string) {
	Eventually(func() bool {
		changedNamespace := &corev1.Namespace{}
		Expect(k8sClient.Get(ctx, types.NamespacedName{Name: namespace}, changedNamespace)).To(Not(HaveOccurred()))

		currentLabels := changedNamespace.GetLabels()

		for k, v := range labels {
			val, exists := currentLabels[k]
			if !exists || (exists && val != v) {
				return false
			}
		}

		return true
	}, Duration, Interval).Should(BeTrue())
}

func ensureLabelsDoesNotExist(ctx context.Context, namespace string, labels map[string]string) {
	Eventually(func() bool {
		changedNamespace := &corev1.Namespace{}
		Expect(k8sClient.Get(ctx, types.NamespacedName{Name: namespace}, changedNamespace)).To(Not(HaveOccurred()))

		currentLabels := changedNamespace.GetLabels()

		for k, v := range labels {
			val, exists := currentLabels[k]
			if exists && val == v {
				return false
			}
		}

		return true
	}, Duration, Interval).Should(BeTrue())
}

var _ = Describe("NamespaceLabel controller test", Ordered, func() {

	ctx := context.Background()

	Namespace := RandomString(16)
	managementLabels := map[string]string{
		"app.kubernetes.io/name": Namespace,
	}

	BeforeAll(func() {
		createNamespace(ctx, Namespace, managementLabels)
	})

	AfterAll(func() {
		k8sClient.DeleteAllOf(
			ctx,
			&idandanielv1.NamespaceLabel{},
			&client.DeleteAllOfOptions{ListOptions: client.ListOptions{Namespace: Namespace}},
		)
		deleteNamespace(ctx, Namespace)
	})

	Context("When creating NamespaceLabel Label", func() {

		NamespaceLabelName := "test-create-nl"
		namespaceLabelLookupKey := types.NamespacedName{Name: NamespaceLabelName, Namespace: Namespace}

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
			}, Duration, Interval).Should(Succeed())

			By("Reconciling the custom resource created")
			startReconcile(ctx, reconcile.Request{NamespacedName: namespaceLabelLookupKey})

			By("Ensuring Namespace management Labels still exist")
			ensureLabelsExists(ctx, Namespace, managementLabels)

			By("Ensuring NamespaceLabel's Labels were added")
			ensureLabelsExists(ctx, Namespace, newLabels)
		})
	})

	Context("When updating NamespaceLabel Label", func() {

		NamespaceLabelName := "test-update-nl"
		namespaceLabelLookupKey := types.NamespacedName{Name: NamespaceLabelName, Namespace: Namespace}

		It("Should update the Namespace Labels.", func() {
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

			By("Getting the NamespaceLabel to update")
			nlToUpdate := &idandanielv1.NamespaceLabel{}
			Expect(k8sClient.Get(ctx, namespaceLabelLookupKey, nlToUpdate)).ToNot(HaveOccurred())

			By("Updating the CR for the Kind NamespaceLabel")
			newLabels := map[string]string{
				"NewOne":     "NewOne",
				"AnotherOne": "AnotherOne",
			}
			nlToUpdate.Spec.Labels = newLabels
			Expect(k8sClient.Update(ctx, nlToUpdate)).To(Not(HaveOccurred()))

			By("Reconciling the CR updated")
			startReconcile(ctx, reconcile.Request{NamespacedName: namespaceLabelLookupKey})

			By("Ensuring Namespace management Labels still exist")
			ensureLabelsExists(ctx, Namespace, managementLabels)

			By("Ensuring NamespaceLabel's Labels were updated")
			ensureLabelsExists(ctx, Namespace, newLabels)
		})
	})

	Context("When deleting NamespaceLabel Label", func() {

		toDeleteName := "test-delete-nl"
		toKeepName := "test-keep-nl"
		toDeleteLookupKey := types.NamespacedName{Name: toDeleteName, Namespace: Namespace}
		toKeepLookupKey := types.NamespacedName{Name: toKeepName, Namespace: Namespace}

		const (
			Delete = "Delete"
			Keep   = "Keep"
		)

		It("Should delete the Labels from the Namespace Labels.", func() {
			ctx = context.Background()

			By("Creating the CR to delete")
			nlToDelete := &idandanielv1.NamespaceLabel{
				ObjectMeta: metav1.ObjectMeta{
					Name:      toDeleteName,
					Namespace: Namespace,
				},
				Spec: idandanielv1.NamespaceLabelSpec{
					Labels: map[string]string{
						Delete: Delete,
					},
				},
			}
			Expect(k8sClient.Create(ctx, nlToDelete)).To(Not(HaveOccurred()))

			By("Creating the CR to keep")
			nlToKeep := &idandanielv1.NamespaceLabel{
				ObjectMeta: metav1.ObjectMeta{
					Name:      toKeepName,
					Namespace: Namespace,
				},
				Spec: idandanielv1.NamespaceLabelSpec{
					Labels: map[string]string{
						Keep: Keep,
					},
				},
			}
			Expect(k8sClient.Create(ctx, nlToKeep)).To(Not(HaveOccurred()))

			By("Checking if the CR to delete was successfully created")
			nlToDelete = &idandanielv1.NamespaceLabel{}
			Eventually(func() error {
				return k8sClient.Get(ctx, toDeleteLookupKey, nlToDelete)
			}, Duration, Interval).Should(Succeed())

			By("Checking if the CR to keep was successfully created")
			nlToKeep = &idandanielv1.NamespaceLabel{}
			Eventually(func() error {
				return k8sClient.Get(ctx, toKeepLookupKey, nlToKeep)
			}, Duration, Interval).Should(Succeed())

			By("Reconciling to apply changes")
			startReconcile(ctx, reconcile.Request{NamespacedName: toDeleteLookupKey})
			startReconcile(ctx, reconcile.Request{NamespacedName: toKeepLookupKey})

			By("Ensuring Namespace management Labels still exist")
			ensureLabelsExists(ctx, Namespace, managementLabels)

			By("Ensure NamespaceLabels's Labels were updated")
			expectedLabels := map[string]string{
				Delete: Delete,
				Keep:   Keep,
			}
			ensureLabelsExists(ctx, Namespace, expectedLabels)

			By("Deleting the CR to delete")
			Expect(k8sClient.Delete(ctx, nlToDelete)).To(Not(HaveOccurred()))

			By("Checking if the CR was successfully deleted")
			Eventually(func() error {
				return k8sClient.Get(ctx, toDeleteLookupKey, &idandanielv1.NamespaceLabel{})
			}, Duration, Interval).Should(Not(Succeed()))

			By("Reconciling the CR deleted")
			startReconcile(ctx, reconcile.Request{NamespacedName: toDeleteLookupKey})

			By("Ensuring Namespace management Labels still exist")
			ensureLabelsExists(ctx, Namespace, managementLabels)

			By("Ensuring Namespacelabel to keep's Labels still exist")
			ensureLabelsExists(ctx, Namespace, nlToKeep.Spec.Labels)

			By("Ensuring NamespaceLabel's Labels were deleted")
			ensureLabelsDoesNotExist(ctx, Namespace, nlToDelete.Spec.Labels)
		})
	})

})

func init() {
	rand.Seed(time.Now().UnixNano())
}

var letterRunes = []rune("abcdefghijklmnopqrstuvwxyz123456789")

func RandomString(n int) string {
	b := make([]rune, n)
	for i := range b {
		b[i] = letterRunes[rand.Intn(len(letterRunes))]
	}
	return string(b)
}

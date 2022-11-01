package controllers

import (
	"context"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	idandanielv1 "idandaniel.io/namespacelabel-demo/api/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
)

const (
	NamespaceLabelName      = "test-namespacelabel"
	NamespaceLabelNamespace = "test-namespace"

	timeout  = time.Second * 10
	duration = time.Second * 10
	interval = time.Millisecond * 250
)

func testCreateNamespaceLabel() {
	ctx := context.Background()
	namespaceLabel := &idandanielv1.NamespaceLabel{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "idandaniel.idandaniel.io/v1",
			Kind:       "NamespaceLabel",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      NamespaceLabelName,
			Namespace: NamespaceLabelNamespace,
		},
		Spec: idandanielv1.NamespaceLabelSpec{
			Labels: map[string]string{
				"key_1": "value_1",
			},
		},
	}

	Expect(k8sClient.Create(ctx, namespaceLabel)).Should(Succeed())
	namespaceLabelLookupKey := types.NamespacedName{Name: NamespaceLabelName, Namespace: NamespaceLabelNamespace}
	createdNamespaceLabel := &idandanielv1.NamespaceLabel{}

	Expect(createdNamespaceLabel.Spec.Labels).Should(Equal(
		map[string]string{},
	))

	Eventually(func() bool {
		err := k8sClient.Get(ctx, namespaceLabelLookupKey, createdNamespaceLabel)
		return err != nil
	}, timeout, interval).Should(BeTrue())
}

func testAddLabelsToNamespace() {

}

var _ = Describe("NamespaceLabel controller", func() {

	// Define utility constants for object names and testing timeouts/durations and intervals.

	Context("When creating NamespaceLabel Labels", func() {
		It("Should sync the Labels with the Namespace Labels.", func() {
			By(
				"Creating new NamespaceLabel",
				testCreateNamespaceLabel,
			)
			By(
				"By adding the NamespaceLabel Labels and keep the mamagement labels",
				testAddLabelsToNamespace,
			)
		})
	})

})

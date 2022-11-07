package controllers

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	idandanielv1 "idandaniel.io/namespacelabel-demo/api/v1"
	"idandaniel.io/namespacelabel-demo/common/wrappers"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var _ = Describe("NamespaceLabel Controller", func() {

	const (
		NamespaceLabelName = "test-namepsacelabel"

		ManagementKey = "app.kubernetes.io/name"
	)

	prevLabels := map[string]string{
		ManagementKey: NamespaceLabelName,
	}
	namespace := &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name:   NamespaceLabelName,
			Labels: prevLabels,
		},
	}
	wrappedNamespace := &wrappers.NamespaceWrapper{Namespace: namespace}

	Context("With one new NamespaceLabel", func() {

		const NewLabelData = "NewLabel"

		newLabels := map[string]string{
			NewLabelData: NewLabelData,
		}

		namespaceLabel := &idandanielv1.NamespaceLabel{
			ObjectMeta: metav1.ObjectMeta{
				Name:      NamespaceLabelName,
				Namespace: namespace.Name,
			},
			Spec: idandanielv1.NamespaceLabelSpec{
				Labels: newLabels,
			},
		}

		It("Should safely append new Labels", func() {
			wrappedNamespace.UpdateLabels(true, namespaceLabel.Spec.Labels)

			Expect(wrappedNamespace.Namespace.Labels).Should(Equal(
				map[string]string{
					ManagementKey: NamespaceLabelName,
					NewLabelData:  NewLabelData,
				},
			))
		})
	})

	Context("With multiple new NamespaceLabels", func() {

		exampleNamespaceLabel1 := &idandanielv1.NamespaceLabel{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "NewNamespaceLabel2",
				Namespace: namespace.Name,
			},
			Spec: idandanielv1.NamespaceLabelSpec{
				Labels: map[string]string{
					"ExampleLabel1": "ExampleLabel1",
				},
			},
		}
		exampleNamespaceLabel2 := &idandanielv1.NamespaceLabel{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "NewNamespaceLabel1",
				Namespace: namespace.Name,
			},
			Spec: idandanielv1.NamespaceLabelSpec{
				Labels: map[string]string{
					"ExampleLabel2": "ExampleLabel2",
				},
			},
		}

		newNamespaceLabels := &idandanielv1.NamespaceLabelList{
			Items: []idandanielv1.NamespaceLabel{
				*exampleNamespaceLabel1,
				*exampleNamespaceLabel2,
			},
		}

		It("Should safely append new Labels", func() {
			wrappedNamespace.UpdateLabels(true, newNamespaceLabels.GetLabels())

			expectedLabels := make(map[string]string)
			for k, v := range exampleNamespaceLabel1.Spec.Labels {
				expectedLabels[k] = v
			}
			for k, v := range exampleNamespaceLabel2.Spec.Labels {
				expectedLabels[k] = v
			}
			expectedLabels[ManagementKey] = NamespaceLabelName

			Expect(wrappedNamespace.Namespace.Labels).Should(Equal(expectedLabels))
		})
	})

	Context("With delete NamespaceLabel", func() {

		const (
			Keep   = "KEEP"
			Remove = "REMOVE"
		)

		labelsToKeep := map[string]string{
			Keep: Keep,
		}

		labelsToRemove := map[string]string{
			Remove: Remove,
		}

		namespaceLabelToKeep := &idandanielv1.NamespaceLabel{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "NamepsaceLabelToKeep",
				Namespace: namespace.Name,
			},
			Spec: idandanielv1.NamespaceLabelSpec{
				Labels: labelsToKeep,
			},
		}

		namespaceLabelToRemove := &idandanielv1.NamespaceLabel{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "NamepsaceLabelToRemove",
				Namespace: namespace.Name,
			},
			Spec: idandanielv1.NamespaceLabelSpec{
				Labels: labelsToRemove,
			},
		}

		allNamespaceLabelsInNamespace := &idandanielv1.NamespaceLabelList{
			Items: []idandanielv1.NamespaceLabel{
				*namespaceLabelToKeep,
				*namespaceLabelToRemove,
			},
		}

		allLabelsToKeep := allNamespaceLabelsInNamespace.GetLabelsExcept(namespaceLabelToRemove)

		namespace := &corev1.Namespace{
			ObjectMeta: metav1.ObjectMeta{
				Name: NamespaceLabelName,
				Labels: map[string]string{
					ManagementKey: NamespaceLabelName,
					Remove:        Remove,
					Keep:          Keep,
				},
			},
		}
		wrappedNamespace := &wrappers.NamespaceWrapper{Namespace: namespace}

		It("Should safely remove Labels", func() {
			wrappedNamespace.RemoveLabelsExcept(
				namespaceLabelToRemove.Spec.Labels,
				allLabelsToKeep,
			)

			expectedLabels := map[string]string{
				ManagementKey: NamespaceLabelName,
				Keep:          Keep,
			}

			Expect(wrappedNamespace.Labels).Should(Equal(expectedLabels))
		})
	})

	Context("With update NamespaceLabel", func() {
		const (
			Old = "OLD"
			New = "NEW"
		)

		namespace := &corev1.Namespace{
			ObjectMeta: metav1.ObjectMeta{
				Name: NamespaceLabelName,
				Labels: map[string]string{
					ManagementKey: NamespaceLabelName,
					Old:           Old,
				},
			},
		}
		wrappedNamespace := &wrappers.NamespaceWrapper{Namespace: namespace}

		It("Should safely update Labels", func() {
			wrappedNamespace.UpdateLabels(
				true,
				map[string]string{New: New},
			)

			expectedLabels := map[string]string{
				ManagementKey: NamespaceLabelName,
				New:           New,
			}

			Expect(wrappedNamespace.Labels).Should(Equal(expectedLabels))
		})
	})
})

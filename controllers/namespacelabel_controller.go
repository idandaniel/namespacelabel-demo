/*
Copyright 2022.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package controllers

import (
	"context"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/log"

	idandanielv1 "idandaniel.io/namespacelabel-demo/api/v1"
	"idandaniel.io/namespacelabel-demo/common/wrappers"
)

type NamespaceLabelReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

const (
	finalizer       string = "idandaniel.idandaniel.io/finalizer"
	AddFinalizer    string = "ADD"
	RemoveFinalizer string = "REMOVE"
)

//+kubebuilder:rbac:groups=idandaniel.idandaniel.io,resources=namespacelabels,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=idandaniel.idandaniel.io,resources=namespacelabels/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=idandaniel.idandaniel.io,resources=namespacelabels/finalizers,verbs=update

// Handles removing safely NamespaceLabels labels from the associated Namespace labels when being deleted.
func (r *NamespaceLabelReconciler) removeLabelsFromAssociatedNamespace(ctx context.Context, namespaceLabel *idandanielv1.NamespaceLabel) error {
	labelsToRemove := namespaceLabel.Spec.Labels
	clientSet := kubernetes.NewForConfigOrDie(ctrl.GetConfigOrDie())

	// Get all NamespaceLabels in the namespace
	allInNamespace := &idandanielv1.NamespaceLabelList{}
	err := r.List(ctx, allInNamespace, &client.ListOptions{Namespace: namespaceLabel.Namespace})
	if client.IgnoreNotFound(err) != nil {
		return err
	}

	// Get all the NamespaceLabels Labels in Namespace except the one being deleted
	labelToIgnore := allInNamespace.GetLabelsExcept(namespaceLabel)

	// Get the namespace to remove labels from
	namespace, err := clientSet.CoreV1().Namespaces().Get(ctx, namespaceLabel.Namespace, metav1.GetOptions{})
	if err != nil {
		return err
	}

	// Update the Namespace
	wrappedNamespace := &wrappers.NamespaceWrapper{Namespace: namespace}
	wrappedNamespace.RemoveLabelsExcept(labelsToRemove, labelToIgnore)
	if _, err = clientSet.CoreV1().Namespaces().Update(ctx, wrappedNamespace.Namespace, metav1.UpdateOptions{}); err != nil {
		return err
	}

	return nil
}

// Generic function for either add or remove NamespaceLabel finalizer
func (r *NamespaceLabelReconciler) changeFinalizer(ctx context.Context, namespaceLabel *idandanielv1.NamespaceLabel, finalizer string, method string) error {
	changeMethods := map[string]interface{}{
		AddFinalizer:    controllerutil.AddFinalizer,
		RemoveFinalizer: controllerutil.RemoveFinalizer,
	}

	_ = changeMethods[method].(func(client.Object, string) bool)(namespaceLabel, finalizer)

	if err := r.Update(ctx, namespaceLabel); err != nil {
		return err
	}

	return nil

}

// Add finalizer to NamespaceLabel if ir doesn't have one
func (r *NamespaceLabelReconciler) addFinalizer(ctx context.Context, namespaceLabel *idandanielv1.NamespaceLabel, finalizer string) error {
	if !controllerutil.ContainsFinalizer(namespaceLabel, finalizer) {
		if err := r.changeFinalizer(ctx, namespaceLabel, finalizer, AddFinalizer); err != nil {
			return err
		}
	}

	return nil
}

// Handle NamespaceLabel deletion - clear the matching labels in Namespace
func (r *NamespaceLabelReconciler) handleDeletion(ctx context.Context, namespaceLabel *idandanielv1.NamespaceLabel, finalizer string) error {

	if controllerutil.ContainsFinalizer(namespaceLabel, finalizer) {
		if err := r.removeLabelsFromAssociatedNamespace(ctx, namespaceLabel); err != nil {
			return err
		}

		if err := r.changeFinalizer(ctx, namespaceLabel, finalizer, RemoveFinalizer); err != nil {
			return err
		}
	}

	return nil
}

// Main function of syncing Between NamespaceLabels to the actual associated Namespace labels
func (r *NamespaceLabelReconciler) sync(ctx context.Context, namespace string) error {

	// Get all the NamespaceLabels of the current request Namespace and retrieve their labels
	namespaceLabelList := &idandanielv1.NamespaceLabelList{}
	if err := r.List(ctx, namespaceLabelList, &client.ListOptions{Namespace: namespace}); err != nil {
		return client.IgnoreNotFound(err)
	}
	labelsToAdd := namespaceLabelList.GetLabels()

	// Get the Namespace
	clientSet := kubernetes.NewForConfigOrDie(ctrl.GetConfigOrDie())
	n, err := clientSet.CoreV1().Namespaces().Get(ctx, namespace, metav1.GetOptions{})
	if err != nil {
		return client.IgnoreNotFound(err)
	}

	// Update the Namespace labels safely (keeps the kubernetes managment tags)
	wrappedNamespace := &wrappers.NamespaceWrapper{Namespace: n}
	wrappedNamespace.UpdateLabels(true, labelsToAdd)
	_, err = clientSet.CoreV1().Namespaces().Update(ctx, wrappedNamespace.Namespace, metav1.UpdateOptions{})
	if err != nil {
		return client.IgnoreNotFound(err)
	}

	return nil
}

// Main reconcile loop
func (r *NamespaceLabelReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	_ = log.FromContext(ctx)

	// Get the associated Namespace
	namespaceLabel := &idandanielv1.NamespaceLabel{}
	if err := r.Get(ctx, req.NamespacedName, namespaceLabel, &client.GetOptions{}); err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	// Handle finalizer
	if !namespaceLabel.IsBeingDeleted() {
		if err := r.addFinalizer(ctx, namespaceLabel, finalizer); err != nil {
			return ctrl.Result{}, err
		}

	} else {
		if err := r.handleDeletion(ctx, namespaceLabel, finalizer); err != nil {
			return ctrl.Result{}, err
		}
		return ctrl.Result{}, nil
	}

	// Sync between NamespaceLabel CR to Namespace labels
	if err := r.sync(ctx, req.NamespacedName.Namespace); err != nil {
		return ctrl.Result{}, err
	}
	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *NamespaceLabelReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&idandanielv1.NamespaceLabel{}).
		Complete(r)
}

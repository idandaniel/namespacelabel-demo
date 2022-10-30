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
	"strings"

	"golang.org/x/exp/maps"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	idandanielv1 "idandaniel.io/namespacelabel-demo/api/v1"
)

// NamespaceLabelReconciler reconciles a NamespaceLabel object
type NamespaceLabelReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

//+kubebuilder:rbac:groups=idandaniel.idandaniel.io,resources=namespacelabels,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=idandaniel.idandaniel.io,resources=namespacelabels/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=idandaniel.idandaniel.io,resources=namespacelabels/finalizers,verbs=update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the NamespaceLabel object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.13.0/pkg/reconcile
func (r *NamespaceLabelReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	_ = log.FromContext(ctx)

	logger := ctrl.Log.WithName("reconcile")

	namespaceLabel := &idandanielv1.NamespaceLabel{}
	err := r.Get(ctx, req.NamespacedName, namespaceLabel)

	if err != nil {
		if errors.IsNotFound(err) {
			return reconcile.Result{}, nil
		}
		return ctrl.Result{}, err
	}

	clientSet := kubernetes.NewForConfigOrDie(ctrl.GetConfigOrDie())
	NamespaceGetter := clientSet.CoreV1().Namespaces

	namespace, err := NamespaceGetter().Get(ctx, namespaceLabel.Namespace, metav1.GetOptions{})

	if err != nil {
		if errors.IsNotFound(err) {
			err := r.Delete(ctx, namespaceLabel)
			if err != nil {
				logger.Info("Failed to delete NamespaceLabel " + namespaceLabel.Name)
				return reconcile.Result{}, err
			}
			logger.Info("Deleted NamespaceLabel " + namespaceLabel.Name)
		}
		return reconcile.Result{}, err
	}

	newLabels := getNamespaceProtectedLabels(namespace)
	maps.Copy(newLabels, namespaceLabel.Spec.Labels)
	namespace.Labels = newLabels

	namespace, err = NamespaceGetter().Update(ctx, namespace, metav1.UpdateOptions{})

	if err != nil {
		logger.Info("Failed to update namespace")
		return reconcile.Result{}, err
	}

	logger.Info(namespaceLabel.Name)
	for labelKey, labelValue := range namespace.Labels {
		logger.Info(labelKey + " ===> " + labelValue)
	}
	logger.Info("-----------------")

	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *NamespaceLabelReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&idandanielv1.NamespaceLabel{}).
		Complete(r)
}

func getNamespaceProtectedLabels(namespace *v1.Namespace) map[string]string {
	protectedLabels := make(map[string]string)

	for labelKey, labelValue := range namespace.Labels {
		if strings.Contains(labelKey, "kubernetes.io") {
			protectedLabels[labelKey] = labelValue
		}
	}

	return protectedLabels
}

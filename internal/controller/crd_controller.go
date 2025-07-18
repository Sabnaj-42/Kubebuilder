/*
Copyright 2025.

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

package controller

import (
	"context"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"time"

	//sabnaj "log"
	//v1 "k8s.io/client-go/applyconfigurations/apps/v1"
	"k8s.io/klog/v2"
	"strings"

	//"fmt"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	logf "sigs.k8s.io/controller-runtime/pkg/log"

	pcv1 "github.com/Sabnaj-42/kubebuilder/api/v1"
)

// CRDReconciler reconciles a CRD object
type CRDReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

// +kubebuilder:rbac:groups=pc.pc.com,resources=crds,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=pc.pc.com,resources=crds/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=pc.pc.com,resources=crds/finalizers,verbs=update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the CRD object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.21.0/pkg/reconcile
func (r *CRDReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	_ = logf.FromContext(ctx)

	// TODO(user): your logic here

	logger := klog.LoggerWithValues(klog.FromContext(ctx), "objectRef", req.NamespacedName)
	var crd pcv1.CRD
	if err := r.Get(ctx, req.NamespacedName, &crd); err != nil {
		if client.IgnoreNotFound(err) == nil {
			logger.Info("CRD resource not found. Ignoring since object must be deleted")
			return ctrl.Result{}, nil
		}
		logger.Error(err, "Failed to get CRD")
		return ctrl.Result{}, nil
	}
	logger.Info("Succesfully fetched CRD resource", "deploymentName", crd.Spec.DeploymentName, "replicas", crd.Spec.Replicas)

	deploymnetName := crd.Spec.DeploymentName
	if deploymnetName == "" {
		// We choose to absorb the error here as the worker would requeue the
		// resource otherwise. Instead, the next time the resource is updated
		// the resource will be queued again.
		utilruntime.HandleErrorWithContext(ctx, nil, "deployment name missing from object reference", "objectReference", req)
		return ctrl.Result{}, nil
	}

	var deploymentObj appsv1.Deployment
	objectKey := client.ObjectKey{
		Namespace: req.Namespace,
		Name:      crd.Spec.Name,
	}

	if objectKey.Name == "" {
		objectKey.Name = strings.Join([]string{crd.Name, pcv1.Service}, pcv1.Dash)
	}

	if err := r.Get(ctx, objectKey, &deploymentObj); err != nil {
		if client.IgnoreNotFound(err) == nil {
			logger.Info("Deployment ", objectKey.Name, "not not found. Creating a new one...")
			err := r.Create(ctx, newDeployment(&crd))
			if err != nil {
				logger.Info("error while creating deployment %\n", crd.Name)
				return ctrl.Result{}, err
			}
			logger.Info("Created")
			return ctrl.Result{}, nil

		}
	} else {
		if crd.Spec.Replicas != nil && *crd.Spec.Replicas != *deploymentObj.Spec.Replicas {
			//logger.Info(*crd.Spec.Replicas, deploymentObj.Spec.Replicas)
			logger.Info("deployment replicas don't match")
			*deploymentObj.Spec.Replicas = *crd.Spec.Replicas
			if err = r.Update(ctx, &deploymentObj); err != nil {
				logger.Info("error while updating deployment %\n", crd.Name)
				return ctrl.Result{}, err
			}
			logger.Info("deployment updated")

		}
		if crd.Spec.Replicas != nil && *deploymentObj.Spec.Replicas != crd.Status.AvailableReplicas {
			//logger.Info(*crd.Spec.Replicas, crd.Status.AvailableReplicas)

			var deepCopy *pcv1.CRD
			deepCopy = crd.DeepCopy()
			deepCopy.Status.AvailableReplicas = *deploymentObj.Spec.Replicas
			if err := r.Status().Update(ctx, deepCopy); err != nil {
				logger.Info("error updating status %s \n", err)
				return ctrl.Result{}, err
			}
			logger.Info("status updated")
		}
	}

	var serviceObject corev1.Service
	objectKey = client.ObjectKey{
		Namespace: req.Namespace,
		Name:      crd.Spec.Name,
	}
	logger.Info(objectKey.Namespace)
	//logger.Info(req)

	if objectKey.Name == "" {
		objectKey.Name = strings.Join([]string{crd.Name, pcv1.Service}, pcv1.Dash)
	}
	if err := r.Get(ctx, objectKey, &serviceObject); err != nil {
		if errors.IsNotFound(err) {
			err := r.Create(ctx, newService(&crd))
			if err != nil {
				logger.Info("error while creating service %s\n", err)
				return ctrl.Result{}, err
			}
			logger.Info("service created successfully")
			return ctrl.Result{}, nil
		}
		return ctrl.Result{}, err
	} else {
		if err := r.Update(ctx, &serviceObject); err != nil {
			logger.Info("error while updating service %s\n", err)
			return ctrl.Result{}, err
		}
		logger.Info("service updated")
	}
	return ctrl.Result{
		Requeue:      true,
		RequeueAfter: 1 * time.Minute,
	}, nil

}

// SetupWithManager sets up the controller with the Manager.
func (r *CRDReconciler) SetupWithManager(mgr ctrl.Manager) error {

	if err := mgr.GetFieldIndexer().IndexField(context.Background(), &appsv1.Deployment{}, pcv1.DeployOwnerKey, func(object client.Object) []string {
		deployment := object.(*appsv1.Deployment)
		owner := metav1.GetControllerOf(deployment)
		if owner == nil {
			return nil
		}
		if owner.APIVersion != pcv1.GroupVersion.String() || owner.Kind != pcv1.MyKind {
			return nil
		}
		return []string{owner.Name}
	}); err != nil {
		return err
	}

	if err := mgr.GetFieldIndexer().IndexField(context.Background(), &corev1.Service{}, pcv1.SvcOwnerKey, func(object client.Object) []string {
		service := object.(*corev1.Service)
		owner := metav1.GetControllerOf(service)
		if owner == nil {
			return nil
		}
		if owner.APIVersion != pcv1.GroupVersion.String() || owner.Kind != pcv1.MyKind {
			return nil
		}
		return []string{owner.Name}
	}); err != nil {
		return err
	}

	handlerForDeployment := handler.EnqueueRequestsFromMapFunc(
		func(ctx context.Context, obj client.Object) []reconcile.Request {
			crds := &pcv1.CRDList{}
			if err := r.List(ctx, crds); err != nil {
				// Log the error if needed
				return nil
			}

			var requests []reconcile.Request
			for _, crd := range crds.Items {
				deploymentName := crd.Spec.Name
				if deploymentName == "" {
					deploymentName = strings.Join([]string{crd.Name, pcv1.Service}, pcv1.Dash)
				}

				if deploymentName == obj.GetName() && crd.Namespace == obj.GetNamespace() {
					tempDeployment := &appsv1.Deployment{}
					if err := r.Get(ctx, types.NamespacedName{
						Namespace: obj.GetNamespace(),
						Name:      obj.GetName(),
					}, tempDeployment); err != nil {
						if errors.IsNotFound(err) {
							requests = append(requests, reconcile.Request{
								NamespacedName: types.NamespacedName{
									Namespace: crd.Namespace,
									Name:      crd.Name,
								},
							})
						}
						continue
					}

					if crd.Spec.Replicas != nil && tempDeployment.Spec.Replicas != nil &&
						*crd.Spec.Replicas != *tempDeployment.Spec.Replicas {
						requests = append(requests, reconcile.Request{
							NamespacedName: types.NamespacedName{
								Namespace: crd.Namespace,
								Name:      crd.Name,
							},
						})
					}
				}
			}
			return requests
		})

	return ctrl.NewControllerManagedBy(mgr).
		For(&pcv1.CRD{}).
		Watches(
			&appsv1.Deployment{},
			handlerForDeployment,
		).
		Owns(&corev1.Service{}).
		Complete(r)
}

/*
Copyright 2024.

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
	"fmt"
	clustergroupv1 "github.com/DenizY98/clusteroperator/api/v1"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/intstr"
	"reflect"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

// EdgeClusterReconciler reconciles a EdgeCluster object
type EdgeClusterReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

const edgeClusterFinalizer = "edgecluster.finalizer.clustergroupv1.trumpf.com"

// +kubebuilder:rbac:groups=clustergroup.trumpf.com,resources=edgeclusters,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=clustergroup.trumpf.com,resources=edgeclusters/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=clustergroup.trumpf.com,resources=edgeclusters/finalizers,verbs=update

func (r *EdgeClusterReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	// Fetch the EdgeCluster instance
	edgeCluster := clustergroupv1.EdgeCluster{}
	if err := r.Get(ctx, req.NamespacedName, &edgeCluster); err != nil {
		if errors.IsNotFound(err) {
			// EdgeCluster resource not found. This may be not an error in case it was deleted.
			return r.handleDeletion(ctx, req.Namespace, &edgeCluster)
		}
		// Error reading the object - requeue the request.
		return ctrl.Result{}, err
	}

	if !edgeCluster.ObjectMeta.DeletionTimestamp.IsZero() {
		// The object is being deleted
		if controllerutil.ContainsFinalizer(&edgeCluster, edgeClusterFinalizer) {
			// Perform deletion logic
			result, err := r.handleDeletion(ctx, req.Namespace, &edgeCluster)
			if err != nil {
				return ctrl.Result{}, err
			}

			// Remove the finalizer and update the EdgeCluster
			controllerutil.RemoveFinalizer(&edgeCluster, edgeClusterFinalizer)
			if err := r.Update(ctx, &edgeCluster); err != nil {
				return ctrl.Result{}, err
			}
			return result, nil
		}
		// Stop reconciliation as the item is being deleted
		return ctrl.Result{}, nil
	}
	// Add finalizer for this CR
	if !controllerutil.ContainsFinalizer(&edgeCluster, edgeClusterFinalizer) {
		controllerutil.AddFinalizer(&edgeCluster, edgeClusterFinalizer)
		if err := r.Update(ctx, &edgeCluster); err != nil {
			return ctrl.Result{}, err
		}
	}

	// The object is not being deleted, handle normal reconciliation
	return r.handleReconciliation(ctx, &edgeCluster)
}

func (r *EdgeClusterReconciler) handleReconciliation(ctx context.Context, edgeCluster *clustergroupv1.EdgeCluster) (ctrl.Result, error) {
	// Check if the Service exists
	serviceName := fmt.Sprintf("nginx-service-%d", edgeCluster.Spec.ID)
	service := corev1.Service{}
	serviceErr := r.Get(ctx, types.NamespacedName{Name: serviceName, Namespace: edgeCluster.Namespace}, &service)

	// Define the Nginx Deployment name
	deploymentName := fmt.Sprintf("nginx-deployment-%d", edgeCluster.Spec.ID)
	deployment := appsv1.Deployment{}
	deploymentErr := r.Get(ctx, types.NamespacedName{Name: deploymentName, Namespace: edgeCluster.Namespace}, &deployment)

	// If either Service or Deployment does not exist, handle creation
	if errors.IsNotFound(serviceErr) || errors.IsNotFound(deploymentErr) {
		return r.handleCreation(ctx, edgeCluster)
	}

	// Otherwise, handle update
	return r.handleUpdate(ctx, edgeCluster, &service)
}

func (r *EdgeClusterReconciler) handleCreation(ctx context.Context, edgeCluster *clustergroupv1.EdgeCluster) (ctrl.Result, error) {
	// Define the Nginx Service
	serviceName := fmt.Sprintf("nginx-service-%d", edgeCluster.Spec.ID)
	service := corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:        serviceName,
			Namespace:   edgeCluster.Namespace,
			Annotations: generateAnnotations(edgeCluster),
		},
		Spec: corev1.ServiceSpec{
			Selector: map[string]string{
				"app": fmt.Sprintf("nginx-%d", edgeCluster.Spec.ID),
			},
			Ports: []corev1.ServicePort{
				{
					Protocol:   corev1.ProtocolTCP,
					Port:       80,
					TargetPort: intstr.FromInt(80),
				},
			},
			Type: corev1.ServiceTypeClusterIP,
		},
	}

	// Create the Service
	if err := r.Create(ctx, &service); err != nil {
		return ctrl.Result{}, err
	}

	// Define the Nginx Deployment
	deploymentName := fmt.Sprintf("nginx-deployment-%d", edgeCluster.Spec.ID)
	deploymentLabels := map[string]string{
		"app": fmt.Sprintf("nginx-%d", edgeCluster.Spec.ID),
	}
	deployment := appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:        deploymentName,
			Namespace:   edgeCluster.Namespace,
			Annotations: generateAnnotations(edgeCluster),
		},
		Spec: appsv1.DeploymentSpec{
			Selector: &metav1.LabelSelector{
				MatchLabels: deploymentLabels,
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: deploymentLabels,
				},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Name:  "nginx",
							Image: "nginx:latest",
							Ports: []corev1.ContainerPort{
								{
									ContainerPort: 80,
								},
							},
						},
					},
				},
			},
		},
	}

	// Create the Deployment
	if err := r.Create(ctx, &deployment); err != nil {
		return ctrl.Result{}, err
	}
	// Specs changed, update Status accordingly
	if !reflect.DeepEqual(edgeCluster.Spec, edgeCluster.Status.Spec) {
		if err := r.updateEdgeClusterStatus(ctx, edgeCluster, serviceName, deploymentName); err != nil {
			return ctrl.Result{}, err
		}
	}
	return ctrl.Result{}, nil
}

func (r *EdgeClusterReconciler) handleUpdate(ctx context.Context, edgeCluster *clustergroupv1.EdgeCluster, service *corev1.Service) (ctrl.Result, error) {
	// Update Service if necessary
	updatedServiceAnnotations := generateAnnotations(edgeCluster)
	if !reflect.DeepEqual(service.Annotations, updatedServiceAnnotations) {
		service.Annotations = updatedServiceAnnotations
		if err := r.Update(ctx, service); err != nil {
			return ctrl.Result{}, err
		}
	}

	// Fetch the existing Deployment
	existingDeployment := appsv1.Deployment{}
	deploymentName := fmt.Sprintf("nginx-deployment-%d", edgeCluster.Spec.ID)
	err := r.Get(ctx, types.NamespacedName{Name: deploymentName, Namespace: edgeCluster.Namespace}, &existingDeployment)
	if err != nil {
		return ctrl.Result{}, err
	}

	// Prepare updated annotations
	updatedAnnotations := generateAnnotations(edgeCluster)

	// Update Deployment's metadata annotations if necessary
	if !reflect.DeepEqual(existingDeployment.Annotations, updatedAnnotations) {
		existingDeployment.Annotations = updatedAnnotations
	}

	// Update Pod template annotations within the Deployment if necessary
	if !reflect.DeepEqual(existingDeployment.Spec.Template.Annotations, updatedAnnotations) {
		existingDeployment.Spec.Template.Annotations = updatedAnnotations
		// Update the Deployment to apply changes to both its metadata and the Pod template
		if err := r.Update(ctx, &existingDeployment); err != nil {
			return ctrl.Result{}, err
		}
	}
	serviceName := fmt.Sprintf("nginx-service-%d", edgeCluster.Spec.ID)
	if err := r.updateEdgeClusterStatus(ctx, edgeCluster, serviceName, deploymentName); err != nil {
		return ctrl.Result{}, err
	}
	return ctrl.Result{}, nil
}

func (r *EdgeClusterReconciler) handleDeletion(ctx context.Context, namespace string, edgeCluster *clustergroupv1.EdgeCluster) (ctrl.Result, error) {
	// Define the dynamic names for the Nginx Service and Deployment
	serviceName := fmt.Sprintf("nginx-service-%d", edgeCluster.Spec.ID)
	deploymentName := fmt.Sprintf("nginx-deployment-%d", edgeCluster.Spec.ID)

	// Attempt to delete the Service
	service := corev1.Service{}
	if err := r.Get(ctx, types.NamespacedName{Name: serviceName, Namespace: namespace}, &service); err == nil {
		// Service exists, delete it
		if err := r.Delete(ctx, &service); err != nil {
			return ctrl.Result{}, err
		}
	} else if !errors.IsNotFound(err) {
		// Error occurred during fetching the service, and it's not a NotFound error
		return ctrl.Result{}, err
	}

	// Attempt to delete the Deployment
	deployment := appsv1.Deployment{}
	if err := r.Get(ctx, types.NamespacedName{Name: deploymentName, Namespace: namespace}, &deployment); err == nil {
		// Deployment exists, delete it
		if err := r.Delete(ctx, &deployment); err != nil {
			return ctrl.Result{}, err
		}
	} else if !errors.IsNotFound(err) {
		// Error occurred during fetching the deployment, and it's not a NotFound error
		return ctrl.Result{}, err
	}

	// Return successfully, indicating the resources have been cleaned up or were not found
	return ctrl.Result{}, nil
}

func (r *EdgeClusterReconciler) updateEdgeClusterStatus(ctx context.Context, edgeCluster *clustergroupv1.EdgeCluster, serviceName string, deploymentName string) error {
	// Update the status fields with the provided service and deployment names
	edgeCluster.Status.ServiceName = serviceName
	edgeCluster.Status.DeploymentName = deploymentName
	edgeCluster.Status.Spec = edgeCluster.Spec

	// Update the EdgeCluster status using the status writer
	return r.Status().Update(ctx, edgeCluster)
}

func generateAnnotations(edgeCluster *clustergroupv1.EdgeCluster) map[string]string {
	return map[string]string{
		"edgecluster.id":               fmt.Sprintf("%d", edgeCluster.Spec.ID),
		"edgecluster.providerId":       fmt.Sprintf("%d", edgeCluster.Spec.ProviderID),
		"edgecluster.kubeconf":         edgeCluster.Spec.Kubeconf,
		"edgecluster.imagePullSecrets": edgeCluster.Spec.ImagePullSecretNames,
		"edgecluster.registryPath":     edgeCluster.Spec.RegistryPath,
		"edgecluster.daprAppId":        edgeCluster.Spec.DaprAppId,
	}
}

func (r *EdgeClusterReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&clustergroupv1.EdgeCluster{}).
		Complete(r)
}

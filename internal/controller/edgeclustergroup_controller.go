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
	"encoding/json"
	"fmt"
	"net/http"
	"reflect"
	"time"

	clustergroupv1 "github.com/DenizY98/clusteroperator/api/v1"
	"github.com/go-logr/logr"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/intstr"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

// EdgeClusterGroupReconciler reconciles a EdgeClusterGroup object
type EdgeClusterGroupReconciler struct {
	client.Client
	Scheme *runtime.Scheme
	Log    logr.Logger
}

// +kubebuilder:rbac:groups=clustergroup.trumpf.com,resources=edgeclustergroups,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=clustergroup.trumpf.com,resources=edgeclustergroups/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=clustergroup.trumpf.com,resources=edgeclustergroups/finalizers,verbs=update

const edgeClusterGroupFinalizer = "edgeclustergroup.finalizer.clustergroupv1.trumpf.com"

// Reconcile the EdgeClusterGroup resources
func (r *EdgeClusterGroupReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	// First, sync with the db-mock service to ensure the ECG objects reflect the current state
	if err := r.syncWithDatabase(ctx); err != nil {
		// Handle errors and possibly requeue
		return ctrl.Result{}, err
	}

	// Check if there are any EdgeClusterGroup objects in the cluster
	ecgList := &clustergroupv1.EdgeClusterGroupList{}
	if err := r.List(ctx, ecgList); err != nil {
		return ctrl.Result{}, err
	}

	if len(ecgList.Items) == 0 {
		// No EdgeClusterGroup objects found, trigger a reconciliation after a certain interval
		return ctrl.Result{RequeueAfter: 1 * time.Minute}, nil
	}

	// Fetch the EdgeClusterGroup instance
	edgeClusterGroup := clustergroupv1.EdgeClusterGroup{}
	if err := r.Get(ctx, req.NamespacedName, &edgeClusterGroup); err != nil {
		if errors.IsNotFound(err) {
			// EdgeClusterGroup resource not found. This may not be an error in case it was deleted.
			return ctrl.Result{}, client.IgnoreNotFound(err)
		}
		// Error reading the object - requeue the request.
		return ctrl.Result{}, err
	}

	if !edgeClusterGroup.ObjectMeta.DeletionTimestamp.IsZero() {
		// The object is being deleted
		if controllerutil.ContainsFinalizer(&edgeClusterGroup, edgeClusterGroupFinalizer) {
			// Perform deletion logic here
			if err := r.finalizeEdgeClusterGroup(ctx, &edgeClusterGroup); err != nil {
				return ctrl.Result{}, err
			}
			// Remove the finalizer and update the EdgeClusterGroup
			controllerutil.RemoveFinalizer(&edgeClusterGroup, edgeClusterGroupFinalizer)
			if err := r.Update(ctx, &edgeClusterGroup); err != nil {
				return ctrl.Result{}, err
			}
		}
		// Stop reconciliation as the item is being deleted
		return ctrl.Result{}, nil
	}

	// Add finalizer for this CR if it does not exist
	if !controllerutil.ContainsFinalizer(&edgeClusterGroup, edgeClusterGroupFinalizer) {
		controllerutil.AddFinalizer(&edgeClusterGroup, edgeClusterGroupFinalizer)
		if err := r.Update(ctx, &edgeClusterGroup); err != nil {
			return ctrl.Result{}, err
		}
	}

	// The object is not being deleted, handle normal reconciliation
	result, err := r.reconcileEdgeClusterGroup(ctx, &edgeClusterGroup)
	if err != nil {
		return result, err
	}

	// Schedule the next reconciliation after a certain interval
	return ctrl.Result{RequeueAfter: 1 * time.Minute / 6}, nil
}

// syncWithDatabase syncs the EdgeClusterGroup objects with the database
func (r *EdgeClusterGroupReconciler) syncWithDatabase(ctx context.Context) error {
	// Discover the db-mock service
	dbMockService := &corev1.Service{}
	if err := r.Get(ctx, types.NamespacedName{Name: "db-mock-service", Namespace: "default"}, dbMockService); err != nil {
		return err
	}

	// For local development of the operator it's beneficial to port-forward the db-api port to your localhost
	// so that it's possible for the operator to be run locally with `make run` without the need to build and deploy foreach change
	// Uncomment the following line to fetch data from the db-mock service inside the running cluster
	// dbMockURL := fmt.Sprintf("http://%s:%d/apps", dbMockService.Spec.ClusterIP, 3001)
	dbMockURL := "http://localhost:3000/apps"
	resp, err := http.Get(dbMockURL)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	// Parse the JSON response
	var dbData struct {
		Providers []struct {
			ID           int    `json:"id"`
			ClientID     string `json:"clientId"`
			ClientSecret string `json:"clientSecret"`
		} `json:"providers"`
		Clusters map[string][]struct {
			ID                   int    `json:"id"`
			ProviderID           int    `json:"providerId"`
			Kubeconf             string `json:"kubeconf"`
			ImagePullSecretNames string `json:"imagePullSecretNames"`
			RegistryPath         string `json:"registryPath"`
			DaprAppId            string `json:"daprAppId"`
		} `json:"clusters"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&dbData); err != nil {
		return err
	}

	// Create/Update ECG objects based on the dbData
	for daprAppId, clusters := range dbData.Clusters {
		// Construct the EdgeClusterGroup object
		ecg := &clustergroupv1.EdgeClusterGroup{
			ObjectMeta: metav1.ObjectMeta{
				Name:      daprAppId,
				Namespace: "default", // Assuming a default namespace, adjust as necessary
			},
			Spec: clustergroupv1.EdgeClusterGroupSpec{
				DaprAppId: daprAppId,
				Clusters:  []clustergroupv1.EdgeClusterSpec{}, // Populate with actual cluster specs
			},
		}

		// Populate the Clusters slice in the spec with actual data
		for _, cluster := range clusters {
			ecg.Spec.Clusters = append(ecg.Spec.Clusters, clustergroupv1.EdgeClusterSpec{
				ID:                   cluster.ID,
				ProviderID:           cluster.ProviderID,
				Kubeconf:             cluster.Kubeconf,
				ImagePullSecretNames: cluster.ImagePullSecretNames,
				RegistryPath:         cluster.RegistryPath,
			})
		}

		// Check if the ECG object already exists
		found := &clustergroupv1.EdgeClusterGroup{}
		err := r.Get(ctx, types.NamespacedName{Name: ecg.Name, Namespace: ecg.Namespace}, found)
		if err != nil && errors.IsNotFound(err) {
			// ECG does not exist, create it
			if err := r.Create(ctx, ecg); err != nil {
				return err
			}
		} else if err == nil {
			// ECG exists, update it
			found.Spec = ecg.Spec
			if err := r.Update(ctx, found); err != nil {
				return err
			}
		} else {
			// Error getting the ECG object - requeue the request.
			return err
		}
	}

	// Handle deletions of ECG objects
	// Fetch all current ECGs
	ecgList := &clustergroupv1.EdgeClusterGroupList{}
	if err := r.List(ctx, ecgList); err != nil {
		return err
	}

	// Create a set of daprAppIds from the db-mock data
	dbDaprAppIds := make(map[string]struct{})
	for daprAppId := range dbData.Clusters {
		dbDaprAppIds[daprAppId] = struct{}{}
	}

	// Iterate over the current ECGs and identify which ones should be deleted
	for _, ecg := range ecgList.Items {
		if _, exists := dbDaprAppIds[ecg.Spec.DaprAppId]; !exists {
			// daprAppId is not present in the db-mock data, delete the ECG
			if err := r.Delete(ctx, &ecg); err != nil {
				return err
			}
		}
	}

	return nil
}

// finalizeEdgeClusterGroup runs when the EdgeClusterGroup is being deleted
func (r *EdgeClusterGroupReconciler) finalizeEdgeClusterGroup(ctx context.Context, edgeClusterGroup *clustergroupv1.EdgeClusterGroup) error {
	// Delete the single service associated with this EdgeClusterGroup
	serviceName := fmt.Sprintf("ecg-service-%s", edgeClusterGroup.Spec.DaprAppId)
	service := &corev1.Service{}
	if err := r.Get(ctx, types.NamespacedName{Name: serviceName, Namespace: edgeClusterGroup.Namespace}, service); err != nil {
		if !errors.IsNotFound(err) {
			// Error getting the service - requeue the request.
			return err
		}
	} else {
		// Service exists - delete it
		if err := r.Delete(ctx, service); err != nil {
			return err
		}
	}
	return nil
}

// reconcileEdgeClusterGroup handles the main reconciliation logic for the EdgeClusterGroup (creation or updating of the services)
func (r *EdgeClusterGroupReconciler) reconcileEdgeClusterGroup(ctx context.Context, edgeClusterGroup *clustergroupv1.EdgeClusterGroup) (ctrl.Result, error) {
	// First, sync with the db-mock service to ensure the ECG objects reflect the current state
	if err := r.syncWithDatabase(ctx); err != nil {
		// Handle errors and possibly requeue
		return ctrl.Result{}, err
	}

	// Check if the EdgeClusterGroup object exists
	found := &clustergroupv1.EdgeClusterGroup{}
	err := r.Get(ctx, types.NamespacedName{Name: edgeClusterGroup.Name, Namespace: edgeClusterGroup.Namespace}, found)
	if err != nil && errors.IsNotFound(err) {
		// EdgeClusterGroup does not exist, create it
		if err := r.Create(ctx, edgeClusterGroup); err != nil {
			return ctrl.Result{}, err
		}
	} else if err != nil {
		// Error getting the EdgeClusterGroup object - requeue the request.
		return ctrl.Result{}, err
	}

	// Generate the name for the service based on the EdgeClusterGroup's daprAppId
	serviceName := fmt.Sprintf("ecg-service-%s", edgeClusterGroup.Spec.DaprAppId)
	service := &corev1.Service{}

	err = r.Get(ctx, types.NamespacedName{Name: serviceName, Namespace: edgeClusterGroup.Namespace}, service)
	if err != nil && errors.IsNotFound(err) {
		// Service does not exist, create it
		service = generateService(edgeClusterGroup, serviceName)
		if err := r.Create(ctx, service); err != nil {
			return ctrl.Result{}, err
		}
	} else if err == nil {
		// Service exists, update it if necessary
		currentAnnotations := service.GetAnnotations()
		desiredAnnotations := generateAnnotationsForGroup(edgeClusterGroup)
		if !annotationsEqual(currentAnnotations, desiredAnnotations) {
			service.SetAnnotations(desiredAnnotations)
			if err := r.Update(ctx, service); err != nil {
				return ctrl.Result{}, err
			}
		}
	} else {
		// Error getting the service - requeue the request.
		return ctrl.Result{}, err
	}

	// Update the status field of the EdgeClusterGroup
	if err := r.updateEdgeClusterGroupStatus(ctx, edgeClusterGroup, service); err != nil {
		return ctrl.Result{}, err
	}

	// Continue with the rest of the reconciliation logic...
	return ctrl.Result{}, nil
}

// updateEdgeClusterGroupStatus updates the status of the given EdgeClusterGroup.
func (r *EdgeClusterGroupReconciler) updateEdgeClusterGroupStatus(ctx context.Context, edgeClusterGroup *clustergroupv1.EdgeClusterGroup, service *corev1.Service) error {
	totalClusters := len(edgeClusterGroup.Spec.Clusters)
	readyClusters := 0
	unreadyClusters := 0

	// Check if the annotations on the service match the desired annotations
	currentAnnotations := service.GetAnnotations()
	desiredAnnotations := generateAnnotationsForGroup(edgeClusterGroup)
	if annotationsEqual(currentAnnotations, desiredAnnotations) {
		// If annotations match, consider all clusters as ready
		readyClusters = totalClusters
	} else {
		// If annotations do not match, consider all clusters as unready
		unreadyClusters = totalClusters
	}

	// Update the EdgeClusterGroup status
	edgeClusterGroup.Status.TotalClusters = totalClusters
	edgeClusterGroup.Status.ReadyClusters = readyClusters
	edgeClusterGroup.Status.UnreadyClusters = unreadyClusters
	edgeClusterGroup.Status.LastUpdated = metav1.Now()

	// Determine the overall group status based on the ready/unready counts
	if unreadyClusters > 0 {
		edgeClusterGroup.Status.GroupStatus = "Degraded"
	} else {
		edgeClusterGroup.Status.GroupStatus = "Active"
	}

	// Write the updated status to the Kubernetes API
	return r.Status().Update(ctx, edgeClusterGroup)
}

// generateService generates a new Service resource for the EdgeClusterGroup
func generateService(edgeClusterGroup *clustergroupv1.EdgeClusterGroup, serviceName string) *corev1.Service {
	return &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:        serviceName,
			Namespace:   edgeClusterGroup.Namespace,
			Annotations: generateAnnotationsForGroup(edgeClusterGroup),
		},
		Spec: corev1.ServiceSpec{
			Selector: map[string]string{
				"app": serviceName,
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
}

// generateAnnotationsForGroup generates annotations for the service based on the EdgeClusterGroup
func generateAnnotationsForGroup(edgeClusterGroup *clustergroupv1.EdgeClusterGroup) map[string]string {
	annotations := map[string]string{
		"edgeclustergroup.daprAppId": edgeClusterGroup.Spec.DaprAppId,
	}

	// Append annotations for each EdgeCluster in the group
	for _, cluster := range edgeClusterGroup.Spec.Clusters {
		clusterAnnotations := generateECAnnotations(cluster)
		for key, value := range clusterAnnotations {
			annotations[key] = value
		}
	}

	return annotations
}

// generateAnnotations generates annotations for an individual EdgeCluster
func generateECAnnotations(cluster clustergroupv1.EdgeClusterSpec) map[string]string {
	return map[string]string{
		fmt.Sprintf("edgecluster.id.%d", cluster.ID):               fmt.Sprintf("%d", cluster.ID),
		fmt.Sprintf("edgecluster.providerId.%d", cluster.ID):       fmt.Sprintf("%d", cluster.ProviderID),
		fmt.Sprintf("edgecluster.kubeconf.%d", cluster.ID):         cluster.Kubeconf,
		fmt.Sprintf("edgecluster.imagePullSecrets.%d", cluster.ID): cluster.ImagePullSecretNames,
		fmt.Sprintf("edgecluster.registryPath.%d", cluster.ID):     cluster.RegistryPath,
	}
}
func annotationsEqual(a, b map[string]string) bool {
	return reflect.DeepEqual(a, b)
}

// SetupWithManager sets up the controller with the Manager.
func (r *EdgeClusterGroupReconciler) SetupWithManager(mgr ctrl.Manager) error {
	r.Log = ctrl.Log.WithName("controllers").WithName("EdgeClusterGroup")
	return ctrl.NewControllerManagedBy(mgr).
		For(&clustergroupv1.EdgeClusterGroup{}).
		Complete(r)
}

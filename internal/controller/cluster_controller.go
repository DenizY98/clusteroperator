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

	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	"github.com/DenizY98/clusteroperator.git/api/v1alpha1"
	clusteropv1alpha1 "github.com/DenizY98/clusteroperator.git/api/v1alpha1"
)

// ClusterReconciler reconciles a Cluster object
type ClusterReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

//+kubebuilder:rbac:groups=clusterop.example.com,resources=clusters,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=clusterop.example.com,resources=clusters/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=clusterop.example.com,resources=clusters/finalizers,verbs=update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the Cluster object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.15.0/pkg/reconcile
// ReconcileCluster reconciles the desired state of clusters based on fetched data.
func (r *ClusterReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := log.FromContext(ctx)

	// Fetch clusters from the API
	clusters, err := fetchClustersFromAPI()
	if err != nil {
		log.Error(err, "Failed to fetch clusters from API")
		return reconcile.Result{}, err
	}

	// Iterate over the fetched clusters
	for _, cluster := range clusters {
		// Create or update the CR cluster based on the fetched data
		crCluster := &clusteropv1alpha1.Cluster{
			ObjectMeta: metav1.ObjectMeta{
				Name:      fmt.Sprintf("cluster-%d", cluster.ID),
				Namespace: "default",
			},
			Spec: clusteropv1alpha1.ClusterSpec{
				ID:                   cluster.ID,
				ProviderID:           cluster.ProviderID,
				Kubeconf:             cluster.Kubeconf,
				ImagePullSecretNames: cluster.ImagePullSecretNames,
				RegistryPath:         cluster.RegistryPath,
				DaprAppID:            cluster.DaprAppID,
			},
		}

		// Check if the CR cluster already exists
		found := &clusteropv1alpha1.Cluster{}
		err := r.Get(ctx, types.NamespacedName{Name: crCluster.Name, Namespace: crCluster.Namespace}, found)
		if err != nil {
			if errors.IsNotFound(err) {
				// Create the CR cluster
				err := r.Create(ctx, crCluster)
				if err != nil {
					log.Error(err, "Failed to create CR cluster", "cluster", crCluster.Name)
					return reconcile.Result{}, err
				}
			} else {
				log.Error(err, "Failed to get CR cluster", "cluster", crCluster.Name)
				return reconcile.Result{}, err
			}
		} else {
			// Update the CR cluster
			found.Spec = crCluster.Spec
			err := r.Update(ctx, found)
			if err != nil {
				log.Error(err, "Failed to update CR cluster", "cluster", crCluster.Name)
				return reconcile.Result{}, err
			}
		}
	}

	return reconcile.Result{}, nil
}
func fetchClustersFromAPI() ([]v1alpha1.Cluster, error) {
	url := "http://localhost:3000/api/cluster"

	// Make the HTTP GET request
	response, err := http.Get(url)
	if err != nil {
		return nil, err
	}
	defer response.Body.Close()

	// Parse the response body as JSON
	var clusters []v1alpha1.Cluster
	if err := json.NewDecoder(response.Body).Decode(&clusters); err != nil {
		return nil, err
	}

	return clusters, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *ClusterReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&clusteropv1alpha1.Cluster{}).
		Complete(r)
}

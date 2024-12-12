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

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	kerrors "k8s.io/apimachinery/pkg/util/errors"
	"k8s.io/klog/v2"
	clusterv1 "sigs.k8s.io/cluster-api/api/v1beta1"
	"sigs.k8s.io/cluster-api/util"
	"sigs.k8s.io/cluster-api/util/patch"
	"sigs.k8s.io/cluster-api/util/predicates"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/log"

	infrav1alpha1 "github.com/miscord-dev/cluster-api-provider-incus/api/v1alpha1"
)

// IncusClusterReconciler reconciles a IncusCluster object
type IncusClusterReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

// +kubebuilder:rbac:groups=infrastructure.cluster.x-k8s.io,resources=incusclusters,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=infrastructure.cluster.x-k8s.io,resources=incusclusters/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=infrastructure.cluster.x-k8s.io,resources=incusclusters/finalizers,verbs=update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the IncusCluster object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.19.0/pkg/reconcile
func (r *IncusClusterReconciler) Reconcile(ctx context.Context, req ctrl.Request) (_ ctrl.Result, rerr error) {
	log := log.FromContext(ctx)

	// Fetch the IncusCluster instance
	incusCluster := &infrav1alpha1.IncusCluster{}
	if err := r.Client.Get(ctx, req.NamespacedName, incusCluster); err != nil {
		if apierrors.IsNotFound(err) {
			return ctrl.Result{}, nil
		}
		return ctrl.Result{}, err
	}

	// Fetch the Cluster.
	cluster, err := util.GetOwnerCluster(ctx, r.Client, incusCluster.ObjectMeta)
	if err != nil {
		return ctrl.Result{}, err
	}
	if cluster == nil {
		log.Info("Waiting for Cluster Controller to set OwnerRef on IncusCluster")
		return ctrl.Result{}, nil
	}

	log = log.WithValues("Cluster", klog.KObj(cluster))
	ctx = ctrl.LoggerInto(ctx, log)

	// Initialize the patch helper
	patchHelper, err := patch.NewHelper(incusCluster, r.Client)
	if err != nil {
		return ctrl.Result{}, err
	}

	// Always attempt to Patch the InMemoryCluster object and status after each reconciliation.
	defer func() {
		if err := patchHelper.Patch(ctx, incusCluster); err != nil {
			rerr = kerrors.NewAggregate([]error{rerr, err})
		}
	}()

	// Handle deleted clusters
	if !incusCluster.DeletionTimestamp.IsZero() {
		return ctrl.Result{}, r.reconcileDelete(ctx, cluster, incusCluster)
	}

	// Add finalizer first if not set to avoid the race condition between init and delete.
	// Note: Finalizers in general can only be added when the deletionTimestamp is not set.
	if !controllerutil.ContainsFinalizer(incusCluster, infrav1alpha1.ClusterFinalizer) {
		controllerutil.AddFinalizer(incusCluster, infrav1alpha1.ClusterFinalizer)
		return ctrl.Result{}, nil
	}

	// Handle non-deleted clusters
	return r.reconcileNormal(ctx, cluster, incusCluster)
}

func (r *IncusClusterReconciler) reconcileDelete(_ context.Context, _ *clusterv1.Cluster, incusCluster *infrav1alpha1.IncusCluster) error {
	controllerutil.RemoveFinalizer(incusCluster, infrav1alpha1.ClusterFinalizer)

	return nil
}

func (r *IncusClusterReconciler) reconcileNormal(ctx context.Context, _ *clusterv1.Cluster, incusCluster *infrav1alpha1.IncusCluster) (ctrl.Result, error) {
	log := log.FromContext(ctx)

	if incusCluster.Spec.ControlPlaneEndpoint.Host == "" {
		log.Info("IncusCluster is not ready as controlPlaneEndpoint.host is missing")

		return ctrl.Result{Requeue: true}, nil
	}
	if incusCluster.Spec.ControlPlaneEndpoint.Port == 0 {
		log.Info("IncusCluster is not ready as controlPlaneEndpoint.port is missing")

		return ctrl.Result{Requeue: true}, nil
	}

	// IncusCluster is ready
	incusCluster.Status.Ready = true

	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *IncusClusterReconciler) SetupWithManager(ctx context.Context, mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&infrav1alpha1.IncusCluster{}).
		Watches(
			&clusterv1.Cluster{},
			handler.EnqueueRequestsFromMapFunc(util.ClusterToInfrastructureMapFunc(ctx, infrav1alpha1.GroupVersion.WithKind("IncusCluster"), mgr.GetClient(), &infrav1alpha1.IncusCluster{})),
			builder.WithPredicates(
				predicates.ClusterUnpaused(ctrl.LoggerFrom(ctx)),
			),
		).
		Named("incuscluster").
		Complete(r)
}

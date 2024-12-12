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
	"errors"
	"fmt"

	"github.com/lxc/incus/shared/api"
	infrav1alpha1 "github.com/miscord-dev/cluster-api-provider-incus/api/v1alpha1"
	"github.com/miscord-dev/cluster-api-provider-incus/pkg/incus"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/klog/v2"
	clusterv1 "sigs.k8s.io/cluster-api/api/v1beta1"
	bootstrapv1 "sigs.k8s.io/cluster-api/bootstrap/kubeadm/api/v1beta1"
	"sigs.k8s.io/cluster-api/util"
	"sigs.k8s.io/cluster-api/util/annotations"
	clog "sigs.k8s.io/cluster-api/util/log"
	"sigs.k8s.io/cluster-api/util/patch"
	"sigs.k8s.io/cluster-api/util/predicates"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/handler"
)

// IncusMachineReconciler reconciles a IncusMachine object
type IncusMachineReconciler struct {
	client.Client
	Scheme *runtime.Scheme

	incusClient incus.Client
}

// +kubebuilder:rbac:groups=infrastructure.cluster.x-k8s.io,resources=incusmachines,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=infrastructure.cluster.x-k8s.io,resources=incusmachines/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=infrastructure.cluster.x-k8s.io,resources=incusmachines/finalizers,verbs=update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the IncusMachine object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.19.0/pkg/reconcile
func (r *IncusMachineReconciler) Reconcile(ctx context.Context, req ctrl.Request) (result ctrl.Result, rerr error) {
	log := ctrl.LoggerFrom(ctx)
	ctx = ctrl.LoggerInto(ctx, log)

	// Fetch the IncusMachine instance.
	incusMachine := &infrav1alpha1.IncusMachine{}
	if err := r.Client.Get(ctx, req.NamespacedName, incusMachine); err != nil {
		if apierrors.IsNotFound(err) {
			return ctrl.Result{}, nil
		}
		return ctrl.Result{}, err
	}

	// AddOwners adds the owners of IncusMachine as k/v pairs to the logger.
	// Specifically, it will add KubeadmControlPlane, MachineSet and MachineDeployment.
	ctx, log, err := clog.AddOwners(ctx, r.Client, incusMachine)
	if err != nil {
		return ctrl.Result{}, err
	}

	// Fetch the Machine.
	machine, err := util.GetOwnerMachine(ctx, r.Client, incusMachine.ObjectMeta)
	if err != nil {
		return ctrl.Result{}, err
	}
	if machine == nil {
		log.Info("Waiting for Machine Controller to set OwnerRef on IncusMachine")
		return ctrl.Result{}, nil
	}

	log = log.WithValues("Machine", klog.KObj(machine))
	ctx = ctrl.LoggerInto(ctx, log)

	// Fetch the Cluster.
	cluster, err := util.GetClusterFromMetadata(ctx, r.Client, machine.ObjectMeta)
	if err != nil {
		log.Info("IncusMachine owner Machine is missing cluster label or cluster does not exist")
		return ctrl.Result{}, err
	}
	if cluster == nil {
		log.Info(fmt.Sprintf("Please associate this machine with a cluster using the label %s: <name of cluster>", clusterv1.ClusterNameLabel))
		return ctrl.Result{}, nil
	}

	log = log.WithValues("Cluster", klog.KObj(cluster))
	ctx = ctrl.LoggerInto(ctx, log)

	// Return early if the object or Cluster is paused.
	if annotations.IsPaused(cluster, incusMachine) {
		log.Info("Reconciliation is paused for this object")
		return ctrl.Result{}, nil
	}

	if cluster.Spec.InfrastructureRef == nil {
		log.Info("Cluster infrastructureRef is not available yet")
		return ctrl.Result{}, nil
	}

	// Fetch the Incus Cluster.
	incusCluster := &infrav1alpha1.IncusCluster{}
	incusClusterName := client.ObjectKey{
		Namespace: incusMachine.Namespace,
		Name:      cluster.Spec.InfrastructureRef.Name,
	}
	if err := r.Client.Get(ctx, incusClusterName, incusCluster); err != nil {
		log.Info("IncusCluster is not available yet")
		return ctrl.Result{}, nil
	}

	// Initialize the patch helper
	patchHelper, err := patch.NewHelper(incusMachine, r.Client)
	if err != nil {
		return ctrl.Result{}, err
	}
	// Always attempt to Patch the IncusMachine object and status after each reconciliation.
	defer func() {
		if err := patchIncusMachine(ctx, patchHelper, incusMachine); err != nil {
			log.Error(err, "Failed to patch IncusMachine")
			if rerr == nil {
				rerr = err
			}
		}
	}()

	// Add finalizer first if not set to avoid the race condition between init and delete.
	// Note: Finalizers in general can only be added when the deletionTimestamp is not set.
	if incusMachine.ObjectMeta.DeletionTimestamp.IsZero() && !controllerutil.ContainsFinalizer(incusMachine, infrav1alpha1.MachineFinalizer) {
		controllerutil.AddFinalizer(incusMachine, infrav1alpha1.MachineFinalizer)
		return ctrl.Result{}, nil
	}

	// Handle deleted machines
	if !incusMachine.ObjectMeta.DeletionTimestamp.IsZero() {
		return r.reconcileDelete(ctx, incusCluster, machine, incusMachine)
	}

	// Handle non-deleted machines
	return r.reconcileNormal(ctx, cluster, incusCluster, machine, incusMachine)
}

func patchIncusMachine(ctx context.Context, patchHelper *patch.Helper, incusMachine *infrav1alpha1.IncusMachine) error {
	// Always update the readyCondition by summarizing the state of other conditions.
	// A step counter is added to represent progress during the provisioning process (instead we are hiding the step counter during the deletion process).
	// conditions.SetSummary(incusMachine,
	// 	conditions.WithConditions(
	// 		infrav1alpha1.ContainerProvisionedCondition,
	// 		infrav1alpha1.BootstrapExecSucceededCondition,
	// 	),
	// 	conditions.WithStepCounterIf(incusMachine.ObjectMeta.DeletionTimestamp.IsZero() && incusMachine.Spec.ProviderID == nil),
	// )

	// Patch the object, ignoring conflicts on the conditions owned by this controller.
	return patchHelper.Patch(
		ctx,
		incusMachine,
		// patch.WithOwnedConditions{Conditions: []clusterv1.ConditionType{
		// 	clusterv1.ReadyCondition,
		// 	infrav1alpha1.ContainerProvisionedCondition,
		// 	infrav1alpha1.BootstrapExecSucceededCondition,
		// }},
	)
}

func (r *IncusMachineReconciler) reconcileDelete(ctx context.Context, _ *infrav1alpha1.IncusCluster, _ *clusterv1.Machine, incusMachine *infrav1alpha1.IncusMachine) (ctrl.Result, error) {
	log := ctrl.LoggerFrom(ctx)

	// If the IncusMachine is being deleted, handle its deletion.
	// log.Info("Handling deleted IncusMachine")
	// if err := r.deleteMachine(ctx, incusCluster, machine, incusMachine, externalMachine, externalLoadBalancer); err != nil {
	// 	log.Error(err, "Failed to delete IncusMachine")
	// 	return err
	// }

	output, err := r.incusClient.GetInstance(ctx, incusMachine.Name)
	if errors.Is(err, incus.ErrorInstanceNotFound) {
		// Instance is already deleted so remove the finalizer.
		controllerutil.RemoveFinalizer(incusMachine, infrav1alpha1.MachineFinalizer)
		return ctrl.Result{}, nil
	}
	if err != nil {
		return ctrl.Result{}, fmt.Errorf("failed to get instance: %w", err)
	}

	if output.StatusCode != api.Stopped &&
		output.StatusCode != api.Stopping {
		if err := r.incusClient.StopInstance(ctx, incusMachine.Name); err != nil {
			log.Info("Failed to stop instance", "error", err)
		}
	} else if output.StatusCode != api.OperationCreated {
		if err := r.incusClient.DeleteInstance(ctx, incusMachine.Name); err != nil {
			return ctrl.Result{}, fmt.Errorf("failed to delete instance: %w", err)
		}
	}

	return ctrl.Result{
		Requeue: true,
	}, nil
}

//nolint:unparam
func (r *IncusMachineReconciler) reconcileNormal(ctx context.Context, cluster *clusterv1.Cluster, _ *infrav1alpha1.IncusCluster, machine *clusterv1.Machine, incusMachine *infrav1alpha1.IncusMachine) (ctrl.Result, error) {
	log := ctrl.LoggerFrom(ctx)

	// Check if the infrastructure is ready, otherwise return and wait for the cluster object to be updated
	if !cluster.Status.InfrastructureReady {
		log.Info("Waiting for IncusCluster Controller to create cluster infrastructure")
		return ctrl.Result{}, nil
	}

	if machine.Spec.Bootstrap.DataSecretName == nil {
		log.Info("Waiting for the Bootstrap provider controller to set bootstrap data")
		return ctrl.Result{}, nil
	}
	dataSecretName := *machine.Spec.Bootstrap.DataSecretName

	_, err := r.incusClient.GetInstance(ctx, incusMachine.Name)
	if err == nil {
		return ctrl.Result{}, nil
	}
	if !errors.Is(err, incus.ErrorInstanceNotFound) {
		return ctrl.Result{}, fmt.Errorf("failed to get instance: %w", err)
	}

	bootstrapData, bootstrapFormat, err := r.getBootstrapData(ctx, incusMachine.Namespace, dataSecretName)
	if err != nil {
		return ctrl.Result{}, err
	}

	// Create the instance
	err = r.incusClient.CreateInstance(ctx, incus.CreateInstanceInput{
		Name: incusMachine.Name,
		BootstrapData: incus.BootstrapData{
			Data:   bootstrapData,
			Format: string(bootstrapFormat),
		},
		InstanceSpec: incusMachine.Spec.InstanceSpec,
	})
	if err != nil {
		return ctrl.Result{}, fmt.Errorf("failed to create instance: %w", err)
	}

	return ctrl.Result{}, nil
}

func (r *IncusMachineReconciler) getBootstrapData(ctx context.Context, namespace string, dataSecretName string) (string, bootstrapv1.Format, error) {
	s := &corev1.Secret{}
	key := client.ObjectKey{Namespace: namespace, Name: dataSecretName}
	if err := r.Client.Get(ctx, key, s); err != nil {
		return "", "", fmt.Errorf("failed to retrieve bootstrap data secret %s: %w", dataSecretName, err)
	}

	value, ok := s.Data["value"]
	if !ok {
		return "", "", errors.New("error retrieving bootstrap data: secret value key is missing")
	}

	format := s.Data["format"]
	if len(format) == 0 {
		format = []byte(bootstrapv1.CloudConfig)
	}

	return string(value), bootstrapv1.Format(format), nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *IncusMachineReconciler) SetupWithManager(ctx context.Context, mgr ctrl.Manager) error {
	clusterToIncusMachines, err := util.ClusterToTypedObjectsMapper(mgr.GetClient(), &infrav1alpha1.IncusMachineList{}, mgr.GetScheme())
	if err != nil {
		return err
	}

	return ctrl.NewControllerManagedBy(mgr).
		For(&infrav1alpha1.IncusMachine{}).
		Watches(
			&clusterv1.Machine{},
			handler.EnqueueRequestsFromMapFunc(util.MachineToInfrastructureMapFunc(infrav1alpha1.GroupVersion.WithKind("IncusMachine"))),
		).
		Watches(
			&infrav1alpha1.IncusCluster{},
			handler.EnqueueRequestsFromMapFunc(r.IncusClusterToIncusMachines),
		).
		Watches(
			&clusterv1.Cluster{},
			handler.EnqueueRequestsFromMapFunc(clusterToIncusMachines),
			builder.WithPredicates(
				predicates.ClusterUnpausedAndInfrastructureReady(ctrl.LoggerFrom(ctx)),
			),
		).
		Named("incusmachine").
		Complete(r)
}

// IncusClusterToIncusMachines is a handler.ToRequestsFunc to be used to enqueue
// requests for reconciliation of IncusMachines.
func (r *IncusMachineReconciler) IncusClusterToIncusMachines(ctx context.Context, o client.Object) []ctrl.Request {
	result := []ctrl.Request{}
	c, ok := o.(*infrav1alpha1.IncusCluster)
	if !ok {
		panic(fmt.Sprintf("Expected a IncusCluster but got a %T", o))
	}

	cluster, err := util.GetOwnerCluster(ctx, r.Client, c.ObjectMeta)
	switch {
	case apierrors.IsNotFound(err) || cluster == nil:
		return result
	case err != nil:
		return result
	}

	labels := map[string]string{clusterv1.ClusterNameLabel: cluster.Name}
	machineList := &clusterv1.MachineList{}
	if err := r.Client.List(ctx, machineList, client.InNamespace(c.Namespace), client.MatchingLabels(labels)); err != nil {
		return nil
	}
	for _, m := range machineList.Items {
		if m.Spec.InfrastructureRef.Name == "" {
			continue
		}
		name := client.ObjectKey{Namespace: m.Namespace, Name: m.Name}
		result = append(result, ctrl.Request{NamespacedName: name})
	}

	return result
}

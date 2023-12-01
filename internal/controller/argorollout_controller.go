package controller

import (
	"context"

	argorolloutsapiv1alpha1 "github.com/argoproj/argo-rollouts/pkg/apis/rollouts/v1alpha1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

	optimizerv1alpha1 "github.com/oviceinc/rollout-optimizer-controller/api/v1alpha1"
)

type ArgoRolloutReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

//+kubebuilder:rbac:groups=argoproj.io,resources=rollouts,verbs=get;list;watch
//+kubebuilder:rbac:groups=argoproj.io,resources=rollouts/status,verbs=get
//+kubebuilder:rbac:groups=rollout.ovice.com,resources=rolloutscaledowns,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=rollout.ovice.com,resources=rolloutscaledowns/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=rollout.ovice.com,resources=rolloutscaledowns/finalizers,verbs=update

func (r *ArgoRolloutReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := log.FromContext(ctx)
	scaleDownList := &optimizerv1alpha1.RolloutScaleDownList{}
	err := r.List(ctx, scaleDownList)
	if err != nil {
		logger.V(1).Error(err, "Faild to list RolloutScaleDown")
		return ctrl.Result{}, err
	}
	var targetScaleDown *optimizerv1alpha1.RolloutScaleDown
	for i := range scaleDownList.Items {
		item := scaleDownList.Items[i]
		if item.Namespace == req.Namespace && item.Spec.TargetRollout == req.Name {
			targetScaleDown = &item
		}
	}
	if targetScaleDown == nil {
		logger.V(2).Info("Not target of ScaleDown", req.Namespace, req.Name)
		return ctrl.Result{}, nil
	}
	argoRollout := &argorolloutsapiv1alpha1.Rollout{}
	err = r.Get(ctx, types.NamespacedName{Name: req.Name, Namespace: req.Namespace}, argoRollout)
	if err != nil && errors.IsNotFound(err) {
		logger.V(1).Info("Not found ArgoRollout", req.Namespace, req.Name)
		return ctrl.Result{}, nil
	} else if err != nil {
		logger.V(1).Error(err, "Failed to get ArgoRollout", req.Namespace, req.Name)
		return ctrl.Result{}, err
	}

	logger.V(1).Info("Target ArgoRollout", argoRollout.Namespace, argoRollout.Name)
	// TODO: Check ArgoRollout status and scaledown if needed

	return ctrl.Result{}, nil
}

func (r *ArgoRolloutReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).For(&argorolloutsapiv1alpha1.Rollout{}).Complete(r)
}

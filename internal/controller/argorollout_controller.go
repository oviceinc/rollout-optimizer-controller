package controller

import (
	"context"
	"time"

	argorolloutsapiv1alpha1 "github.com/argoproj/argo-rollouts/pkg/apis/rollouts/v1alpha1"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/record"
	klog "k8s.io/klog/v2"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	optimizerv1alpha1 "github.com/oviceinc/rollout-optimizer-controller/api/v1alpha1"
)

type ArgoRolloutReconciler struct {
	client.Client
	Scheme   *runtime.Scheme
	Recorder record.EventRecorder
}

//+kubebuilder:rbac:groups=argoproj.io,resources=rollouts,verbs=get;list;watch
//+kubebuilder:rbac:groups=argoproj.io,resources=rollouts/status,verbs=get
//+kubebuilder:rbac:groups=rollout.ovice.com,resources=rolloutscaledowns,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=rollout.ovice.com,resources=rolloutscaledowns/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=rollout.ovice.com,resources=rolloutscaledowns/finalizers,verbs=update
//+kubebuilder:rbac:groups=apps,resources=replicasets,verbs=get;list;watch;update

func (r *ArgoRolloutReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	scaleDownList := &optimizerv1alpha1.RolloutScaleDownList{}
	err := r.List(ctx, scaleDownList)
	if err != nil {
		klog.Errorf("Failed to list RolloutScaleDown: %v", err)
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
		klog.V(1).Infof("No target of ScaleDown: %s/%s", req.Namespace, req.Name)
		return ctrl.Result{}, nil
	}
	argoRollout := &argorolloutsapiv1alpha1.Rollout{}
	err = r.Get(ctx, types.NamespacedName{Name: req.Name, Namespace: req.Namespace}, argoRollout)
	if err != nil && errors.IsNotFound(err) {
		klog.V(1).Infof("Not found ArgoRollout: %s/%s", req.Namespace, req.Name)
		return ctrl.Result{}, nil
	} else if err != nil {
		klog.Errorf("Failed to get ArgoRollout: %v", err)
		return ctrl.Result{}, err
	}

	klog.Infof("Target ArgoRollout: %s/%s", argoRollout.Namespace, argoRollout.Name)

	if isCompleted(&argoRollout.Status) {
		klog.V(1).Infof("Rollout is completed: %s/%s", argoRollout.Namespace, argoRollout.Name)

		if argoRollout.Status.Replicas == argoRollout.Status.ReadyReplicas {
			if targetScaleDown.Status.Phase != optimizerv1alpha1.RolloutScaleDownPhaseHealthy {
				newStatus := targetScaleDown.Status.DeepCopy()
				newStatus.Phase = optimizerv1alpha1.RolloutScaleDownPhaseHealthy
				targetScaleDown.Status = *newStatus
				if err := r.Status().Update(ctx, targetScaleDown); err != nil {
					klog.Errorf("Failed to update RolloutScaleDown: %v", err)
					return ctrl.Result{}, err
				}
				klog.Infof("RolloutScaleDown is healthy: %s/%s", targetScaleDown.Namespace, targetScaleDown.Name)
				r.Recorder.Eventf(targetScaleDown, corev1.EventTypeNormal, "Healthy", "RolloutScaleDown is healthy")
			}
			return ctrl.Result{}, nil
		}

		rsHash := argoRollout.Status.StableRS
		oldRS := []*appsv1.ReplicaSet{}
		RSList := &appsv1.ReplicaSetList{}
		if err := r.List(ctx, RSList, &client.ListOptions{LabelSelector: labels.SelectorFromSet(map[string]string{"app": argoRollout.Name})}); err != nil {
			klog.Errorf("Failed to list ReplicaSet: %v", err)
			return ctrl.Result{}, err
		}

		for i := range RSList.Items {
			rs := RSList.Items[i]
			if rs.Name == argoRollout.Name+"-"+rsHash {
				continue
			}
			if rs.Status.Replicas > 0 {
				oldRS = append(oldRS, &rs)
			}
		}
		if len(oldRS) == 0 {
			klog.V(1).Infof("Old ReplicaSet is not found: %s/%s", argoRollout.Namespace, argoRollout.Name)
			return ctrl.Result{}, nil
		}

		for i := range oldRS {
			rs := oldRS[i]
			if rs.Status.Replicas == 0 {
				klog.V(1).Infof("Old ReplicaSet is already scaled down: %s/%s", rs.Namespace, rs.Name)
				continue
			}
			klog.V(1).Infof("Old ReplicaSet: %s/%s", rs.Namespace, rs.Name)

			retry, err := r.scaleDown(ctx, rs, targetScaleDown, argoRollout)
			if err != nil {
				klog.Errorf("Failed to scale down ReplicaSet: %v", err)
				return ctrl.Result{}, err
			}
			if retry > 0 {
				if targetScaleDown.Status.Phase != optimizerv1alpha1.RolloutScaleDownPhaseScaling {
					newStatus := targetScaleDown.Status.DeepCopy()
					newStatus.Phase = optimizerv1alpha1.RolloutScaleDownPhaseScaling
					targetScaleDown.Status = *newStatus
					if err := r.Status().Update(ctx, targetScaleDown); err != nil {
						klog.Errorf("Failed to update RolloutScaleDown: %v", err)
						return ctrl.Result{}, err
					}
				}
				return ctrl.Result{RequeueAfter: retry, Requeue: true}, nil
			}
		}
	}

	return ctrl.Result{}, nil
}

func (r *ArgoRolloutReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).For(&argorolloutsapiv1alpha1.Rollout{}).Complete(r)
}

func isCompleted(status *argorolloutsapiv1alpha1.RolloutStatus) bool {
	if status.Phase != argorolloutsapiv1alpha1.RolloutPhaseHealthy {
		return false
	}
	completed := false
	progressing := false
	for _, cond := range status.Conditions {
		if cond.Type == argorolloutsapiv1alpha1.RolloutCompleted && cond.Status == corev1.ConditionTrue {
			completed = true
		}
		if cond.Type == argorolloutsapiv1alpha1.RolloutProgressing && cond.Status == corev1.ConditionTrue {
			progressing = true
		}
	}
	if completed == true && progressing == true && status.Replicas >= status.ReadyReplicas {
		return true
	}
	return false
}

func (r *ArgoRolloutReconciler) scaleDown(ctx context.Context, rs *appsv1.ReplicaSet, scaleDown *optimizerv1alpha1.RolloutScaleDown, rollout *argorolloutsapiv1alpha1.Rollout) (time.Duration, error) {
	rolloutUpdated := time.Now()
	for _, cond := range rollout.Status.Conditions {
		if cond.Type == argorolloutsapiv1alpha1.RolloutCompleted {
			rolloutUpdated = cond.LastUpdateTime.Time
		}
	}
	if rolloutUpdated.Add(time.Duration(scaleDown.Spec.CoolTimeSeconds) * time.Second).After(time.Now()) {
		return 30 * time.Second, nil
	}

	lastUpdated := scaleDown.Status.LastScaleDownTime
	if lastUpdated.Add(time.Duration(scaleDown.Spec.CoolTimeSeconds) * time.Second).After(time.Now()) {
		return 30 * time.Second, nil
	}

	if *rs.Spec.Replicas == 0 {
		return 0, nil
	}
	replicas := *rs.Spec.Replicas - int32(scaleDown.Spec.TerminatePerOnce)

	newStatus := scaleDown.Status.DeepCopy()
	newStatus.LastScaleDownTime = metav1.Now()
	newStatus.Phase = optimizerv1alpha1.RolloutScaleDownPhaseScaling
	scaleDown.Status = *newStatus
	if err := r.Status().Update(ctx, scaleDown); err != nil {
		klog.Errorf("Failed to update RolloutScaleDown: %v", err)
		return 0, err
	}

	r.Recorder.Eventf(scaleDown, corev1.EventTypeNormal, "ScaleDown", "Scaling down %s/%s replicas %d -> %d", rs.Namespace, rs.Name, *rs.Spec.Replicas, replicas)
	klog.Infof("Scaling down ReplicaSet %s/%s replicas %d -> %d", rs.Namespace, rs.Name, *rs.Spec.Replicas, replicas)
	rs.Spec.Replicas = &replicas
	if err := r.Update(ctx, rs); err != nil {
		klog.Errorf("Failed to update ReplicaSet: %v", err)
		return 0, err
	}

	return 0, nil
}

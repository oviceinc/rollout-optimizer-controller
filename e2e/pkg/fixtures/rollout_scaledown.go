package fixtures

import (
	"context"

	optimizerv1alpha1 "github.com/oviceinc/rollout-optimizer-controller/api/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func ApplyRolloutScaledown(ctx context.Context, k8sClient client.Client, ns, name, rollout string, perOnce, coolTime int) error {
	scaledown := &optimizerv1alpha1.RolloutScaleDown{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: ns,
		},
		Spec: optimizerv1alpha1.RolloutScaleDownSpec{
			TargetRollout:    rollout,
			TerminatePerOnce: perOnce,
			CoolTimeSeconds:  coolTime,
		},
	}

	return k8sClient.Create(ctx, scaledown)
}

func DeleteRolloutScaledown(ctx context.Context, k8sClient client.Client, ns, name string) error {
	return k8sClient.Delete(ctx, &optimizerv1alpha1.RolloutScaleDown{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: ns,
		},
	})
}

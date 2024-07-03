package fixtures

import (
	"context"

	argorolloutsapiv1alpha1 "github.com/argoproj/argo-rollouts/pkg/apis/rollouts/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/klog/v2"
	utilpointer "k8s.io/utils/pointer"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

var (
	activeService  = "active"
	previewService = "preview"
)

func ApplyRollout(ctx context.Context, k8sClient client.Client, ns, name, image string, replicas int) error {
	if err := service(ctx, k8sClient, ns, activeService, name); err != nil {
		return err
	}

	if err := service(ctx, k8sClient, ns, previewService, name); err != nil {
		return err
	}

	rollout := &argorolloutsapiv1alpha1.Rollout{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: ns,
			Labels: map[string]string{
				"app": name,
			},
		},
		Spec: argorolloutsapiv1alpha1.RolloutSpec{
			Replicas: utilpointer.Int32(int32(replicas)),
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{
					"app": name,
				},
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{
						"app": name,
					},
				},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Name:  name,
							Image: image,
							Ports: []corev1.ContainerPort{
								{
									ContainerPort: 80,
								},
							},
						},
					},
				},
			},
			Strategy: argorolloutsapiv1alpha1.RolloutStrategy{
				BlueGreen: &argorolloutsapiv1alpha1.BlueGreenStrategy{
					ActiveService:         activeService,
					PreviewService:        previewService,
					AutoPromotionEnabled:  utilpointer.Bool(true),
					ScaleDownDelaySeconds: utilpointer.Int32(0),
				},
			},
		},
	}

	return k8sClient.Create(ctx, rollout)
}

func UpdateRollout(ctx context.Context, k8sClient client.Client, ns, name, image string) error {
	found := &argorolloutsapiv1alpha1.Rollout{}
	err := k8sClient.Get(ctx, client.ObjectKey{Namespace: ns, Name: name}, found)
	if err != nil {
		return err
	}

	found.Spec.Template.Spec.Containers[0].Image = image
	return k8sClient.Update(ctx, found)
}

func DeleteRollout(ctx context.Context, k8sClient client.Client, ns, name string) error {
	if err := k8sClient.Delete(ctx, &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      activeService,
			Namespace: ns,
		},
	}); err != nil {
		klog.Error(err)
	}
	if err := k8sClient.Delete(ctx, &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      previewService,
			Namespace: ns,
		},
	}); err != nil {
		klog.Error(err)
	}
	if err := k8sClient.Delete(ctx, &argorolloutsapiv1alpha1.Rollout{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: ns,
		},
	}); err != nil {
		klog.Error(err)
	}
	return nil
}

func service(ctx context.Context, k8sClient client.Client, ns, name, rollout string) error {
	svc := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: ns,
		},
		Spec: corev1.ServiceSpec{
			Selector: map[string]string{
				"app": rollout,
			},
			Ports: []corev1.ServicePort{
				{
					Port: 80,
				},
			},
		},
	}

	return k8sClient.Create(ctx, svc)

}

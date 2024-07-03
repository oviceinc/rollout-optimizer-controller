package fixtures

import (
	"context"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/klog/v2"
	utilpointer "k8s.io/utils/pointer"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func ApplyManager(ctx context.Context, k8sClient client.Client, ns, name, image string) error {
	SAName := "manager"
	roleName := "manager-role"

	sa := serviceAccount(ns, SAName)
	if err := k8sClient.Create(ctx, sa); err != nil {
		return err
	}
	role := role(roleName)
	if err := k8sClient.Create(ctx, role); err != nil {
		return err
	}
	roleBinding := clusterRoleBinding(ns, SAName, roleName)
	if err := k8sClient.Create(ctx, roleBinding); err != nil {
		return err
	}
	leaderElectionRole := leaderElectionRole(ns, "leader-election")
	if err := k8sClient.Create(ctx, leaderElectionRole); err != nil {
		return err
	}
	leaderElectionRoleBinding := leaderElectionRoleBinding(ns, "leader-election", SAName)
	if err := k8sClient.Create(ctx, leaderElectionRoleBinding); err != nil {
		return err
	}
	deploy := deployment(ns, name, image, SAName)
	if err := k8sClient.Create(ctx, deploy); err != nil {
		return err
	}
	return nil
}

func DeleteManager(ctx context.Context, k8sClient client.Client, ns, name string) error {
	SAName := "manager"
	roleName := "manager-role"

	if err := k8sClient.Delete(ctx, &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: ns,
		},
	}); err != nil {
		klog.Error(err)
	}
	if err := k8sClient.Delete(ctx, &rbacv1.RoleBinding{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "leader-election",
			Namespace: ns,
		},
	}); err != nil {
		klog.Error(err)
	}
	if err := k8sClient.Delete(ctx, &rbacv1.Role{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "leader-election",
			Namespace: ns,
		},
	}); err != nil {
		klog.Error(err)
	}
	if err := k8sClient.Delete(ctx, &rbacv1.ClusterRoleBinding{
		ObjectMeta: metav1.ObjectMeta{
			Name: roleName,
		},
	}); err != nil {
		klog.Error(err)
	}
	if err := k8sClient.Delete(ctx, &rbacv1.ClusterRole{
		ObjectMeta: metav1.ObjectMeta{
			Name: roleName,
		},
	}); err != nil {
		klog.Error(err)
	}
	if err := k8sClient.Delete(ctx, &corev1.ServiceAccount{
		ObjectMeta: metav1.ObjectMeta{
			Name:      SAName,
			Namespace: ns,
		},
	}); err != nil {
		klog.Error(err)
	}

	return nil
}

func serviceAccount(ns, name string) *corev1.ServiceAccount {
	return &corev1.ServiceAccount{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: ns,
		},
	}
}

func role(name string) *rbacv1.ClusterRole {
	return &rbacv1.ClusterRole{
		ObjectMeta: metav1.ObjectMeta{
			Name: name,
		},
		Rules: []rbacv1.PolicyRule{
			{
				APIGroups: []string{"apps"},
				Resources: []string{"replicasets"},
				Verbs:     []string{"get", "list", "watch", "update"},
			},
			{
				APIGroups: []string{"argoproj.io"},
				Resources: []string{"rollouts"},
				Verbs:     []string{"get", "list", "watch"},
			},
			{
				APIGroups: []string{"argoproj.io"},
				Resources: []string{"rollouts/status"},
				Verbs:     []string{"get"},
			},
			{
				APIGroups: []string{"rollout.ovice.com"},
				Resources: []string{"rolloutscaledowns"},
				Verbs:     []string{"get", "list", "watch", "create", "update", "patch", "delete"},
			},
			{
				APIGroups: []string{"rollout.ovice.com"},
				Resources: []string{"rolloutscaledowns/status"},
				Verbs:     []string{"get", "update", "patch"},
			},
		},
	}
}

func clusterRoleBinding(ns, SAName, role string) *rbacv1.ClusterRoleBinding {
	return &rbacv1.ClusterRoleBinding{
		ObjectMeta: metav1.ObjectMeta{
			Name: role,
		},
		Subjects: []rbacv1.Subject{
			{
				Kind:      "ServiceAccount",
				Name:      SAName,
				Namespace: ns,
			},
		},
		RoleRef: rbacv1.RoleRef{
			Kind: "ClusterRole",
			Name: role,
		},
	}
}

func leaderElectionRole(ns, name string) *rbacv1.Role {
	return &rbacv1.Role{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: ns,
		},
		Rules: []rbacv1.PolicyRule{
			{
				Verbs: []string{
					"get",
					"list",
					"watch",
					"create",
					"update",
					"patch",
					"delete",
				},
				APIGroups: []string{""},
				Resources: []string{"configmaps"},
			},
			{
				Verbs: []string{
					"get",
					"update",
					"patch",
				},
				APIGroups: []string{""},
				Resources: []string{"configmaps/status"},
			},
			{
				Verbs: []string{
					"get",
					"list",
					"watch",
					"create",
					"update",
					"patch",
					"delete",
				},
				APIGroups: []string{"coordination.k8s.io"},
				Resources: []string{"leases"},
			},
		},
	}
}

func leaderElectionRoleBinding(ns, roleName, SAName string) *rbacv1.RoleBinding {
	return &rbacv1.RoleBinding{
		ObjectMeta: metav1.ObjectMeta{
			Name:      roleName,
			Namespace: ns,
		},
		Subjects: []rbacv1.Subject{
			{
				Kind:      rbacv1.ServiceAccountKind,
				Name:      SAName,
				Namespace: ns,
			},
		},
		RoleRef: rbacv1.RoleRef{
			APIGroup: "rbac.authorization.k8s.io",
			Kind:     "Role",
			Name:     roleName,
		},
	}
}

func deployment(ns, name, image, SAName string) *appsv1.Deployment {
	return &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: ns,
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: utilpointer.Int32(1),
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
						},
					},
					ServiceAccountName: SAName,
				},
			},
		},
	}

}

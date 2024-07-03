package e2e_test

import (
	"context"
	"fmt"
	"os"
	"time"

	argorolloutsapiv1alpha1 "github.com/argoproj/argo-rollouts/pkg/apis/rollouts/v1alpha1"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	optimizerv1alpha1 "github.com/oviceinc/rollout-optimizer-controller/api/v1alpha1"
	"github.com/oviceinc/rollout-optimizer-controller/e2e/pkg/fixtures"
	"github.com/oviceinc/rollout-optimizer-controller/e2e/pkg/util"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/wait"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	klog "k8s.io/klog/v2"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

var (
	cfg       *rest.Config
	k8sClient client.Client
	scheme    = runtime.NewScheme()
	namespace = "default"
	rollout   = "example"
	scaledown = "example-scaledown"
	perOnce   = 1
	coolTime  = 120
	replicas  = 4
)

var _ = BeforeSuite(func() {
	configFile := os.Getenv("KUBECONFIG")
	if configFile == "" {
		configFile = "$HOME/.kube/config"
	}
	var err error
	cfg, err = clientcmd.BuildConfigFromFlags("", os.ExpandEnv(configFile))
	Expect(err).ShouldNot(HaveOccurred())

	err = clientgoscheme.AddToScheme(scheme)
	Expect(err).ShouldNot(HaveOccurred())
	err = argorolloutsapiv1alpha1.AddToScheme(scheme)
	Expect(err).ShouldNot(HaveOccurred())
	err = optimizerv1alpha1.AddToScheme(scheme)
	Expect(err).ShouldNot(HaveOccurred())

	k8sClient, err = client.New(cfg, client.Options{Scheme: scheme})
	Expect(err).ShouldNot(HaveOccurred())

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
	defer cancel()

	err = waitUntilReady(ctx, k8sClient)
	Expect(err).ShouldNot(HaveOccurred())
})

func waitUntilReady(ctx context.Context, k8sClient client.Client) error {
	klog.Info("Waiting until kubernetes cluster is ready")
	err := wait.Poll(10*time.Second, 10*time.Minute, func() (bool, error) {
		nodeList := &corev1.NodeList{}
		err := k8sClient.List(ctx, nodeList)
		if err != nil {
			return false, fmt.Errorf("failed to list nodes: %v", err)
		}
		if len(nodeList.Items) == 0 {
			klog.Warningf("node does not exist yet")
			return false, nil
		}
		for i := range nodeList.Items {
			n := &nodeList.Items[i]
			if !nodeIsReady(n) {
				klog.Warningf("node %s is not ready yet", n.Name)
				return false, nil
			}
		}
		klog.Info("all nodes are ready")
		return true, nil
	})
	return err
}

func nodeIsReady(node *corev1.Node) bool {
	for i := range node.Status.Conditions {
		con := &node.Status.Conditions[i]
		if con.Type == corev1.NodeReady && con.Status == corev1.ConditionTrue {
			return true
		}
	}
	return false
}

var _ = Describe("E2E", func() {
	BeforeEach(func() {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
		defer cancel()

		if err := util.ApplyCRD(ctx, cfg); err != nil {
			panic(err)
		}

		// Apply Rollout
		if err := fixtures.ApplyRollout(ctx, k8sClient, namespace, rollout, "nginx:1.25.0", replicas); err != nil {
			klog.Error(err)
		}
		// Apply RolloutScaleDown
		if err := fixtures.ApplyRolloutScaledown(ctx, k8sClient, namespace, scaledown, rollout, perOnce, coolTime); err != nil {
			klog.Error(err)
		}
		// Apply manager
		if err := fixtures.ApplyManager(ctx, k8sClient, namespace, "manager", os.Getenv("MANAGER_IMAGE")); err != nil {
			klog.Error(err)
		}
	})
	AfterEach(func() {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
		defer cancel()

		if err := fixtures.DeleteManager(ctx, k8sClient, namespace, "manager"); err != nil {
			klog.Error(err)
		}

		if err := fixtures.DeleteRolloutScaledown(ctx, k8sClient, namespace, scaledown); err != nil {
			klog.Error(err)
		}

		if err := fixtures.DeleteRollout(ctx, k8sClient, namespace, rollout); err != nil {
			klog.Error(err)
		}
	})
	It("E2E", func() {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
		defer cancel()

		r := &argorolloutsapiv1alpha1.Rollout{}
		err := wait.Poll(10*time.Second, 10*time.Minute, func() (bool, error) {
			err := k8sClient.Get(ctx, types.NamespacedName{Name: rollout, Namespace: namespace}, r)
			if err != nil {
				return false, err
			}
			if r.Status.Phase == argorolloutsapiv1alpha1.RolloutPhaseHealthy {
				return true, nil
			}
			return false, nil
		})
		Expect(err).ShouldNot(HaveOccurred())

		oldHash := r.Status.StableRS
		oldRS := &appsv1.ReplicaSet{}
		err = wait.Poll(10*time.Second, 10*time.Minute, func() (bool, error) {
			err := k8sClient.Get(ctx, types.NamespacedName{Name: rollout + "-" + oldHash, Namespace: namespace}, oldRS)
			if err != nil {
				return false, err
			}
			if oldRS.Status.Replicas == oldRS.Status.ReadyReplicas {
				return true, nil
			}
			return false, nil
		})
		Expect(err).ShouldNot(HaveOccurred())
		Expect(oldRS.Status.Replicas).Should(Equal(int32(replicas)))

		// Update Rollout
		err = fixtures.UpdateRollout(ctx, k8sClient, namespace, rollout, "nginx:latest")
		Expect(err).ShouldNot(HaveOccurred())

		// Wait until rollout is healthy
		err = wait.Poll(10*time.Second, 10*time.Minute, func() (bool, error) {
			err := k8sClient.Get(ctx, types.NamespacedName{Name: rollout, Namespace: namespace}, r)
			if err != nil {
				return false, err
			}
			if r.Status.Phase == argorolloutsapiv1alpha1.RolloutPhaseHealthy {
				return true, nil
			}
			return false, nil
		})
		Expect(err).ShouldNot(HaveOccurred())

		time.Sleep(10 * time.Second)

		// The old replica set should be scaled down
		err = k8sClient.Get(ctx, types.NamespacedName{Name: oldRS.Name, Namespace: namespace}, oldRS)
		Expect(err).ShouldNot(HaveOccurred())
		Expect(oldRS.Status.Replicas).Should(Equal(int32(replicas - perOnce)))

		// Wait 120 seconds
		time.Sleep(120 * time.Second)
		// The old replica set should be scaled down
		err = k8sClient.Get(ctx, types.NamespacedName{Name: oldRS.Name, Namespace: namespace}, oldRS)
		Expect(err).ShouldNot(HaveOccurred())
		Expect(oldRS.Status.Replicas).Should(Equal(int32(replicas - perOnce*2)))

		// Wait 120 seconds
		time.Sleep(120 * time.Second)
		// The old replica set should be scaled down
		err = k8sClient.Get(ctx, types.NamespacedName{Name: oldRS.Name, Namespace: namespace}, oldRS)
		Expect(err).ShouldNot(HaveOccurred())
		Expect(oldRS.Status.Replicas).Should(Equal(int32(replicas - perOnce*3)))

		// Wait 120 seconds
		time.Sleep(120 * time.Second)
		// The old replica set should be scaled down
		err = k8sClient.Get(ctx, types.NamespacedName{Name: oldRS.Name, Namespace: namespace}, oldRS)
		Expect(err).ShouldNot(HaveOccurred())
		Expect(oldRS.Status.Replicas).Should(Equal(int32(replicas - perOnce*4)))
	})
})

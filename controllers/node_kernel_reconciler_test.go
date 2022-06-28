package controllers

import (
	"context"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	runtimectrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

var _ = Describe("NodeKernelReconciler", func() {
	Describe("Reconcile", func() {
		const (
			kernelVersion = "1.2.3"
			labelName     = "label-name"
			nodeName      = "node-name"
		)

		It("should return an error if the node cannot be found anymore", func() {
			nkr := NewNodeKernelReconciler(fake.NewClientBuilder().Build(), labelName, nil)
			req := runtimectrl.Request{
				NamespacedName: types.NamespacedName{Name: nodeName},
			}

			_, err := nkr.Reconcile(context.TODO(), req)
			Expect(err).To(HaveOccurred())
		})

		It("should set the label if it does not exist", func() {
			node := v1.Node{
				ObjectMeta: metav1.ObjectMeta{Name: nodeName},
				Status: v1.NodeStatus{
					NodeInfo: v1.NodeSystemInfo{KernelVersion: kernelVersion},
				},
			}

			client := fake.NewClientBuilder().WithObjects(&node).Build()

			nkr := NewNodeKernelReconciler(client, labelName, nil)
			req := runtimectrl.Request{
				NamespacedName: types.NamespacedName{Name: nodeName},
			}

			ctx := context.TODO()

			res, err := nkr.Reconcile(ctx, req)
			Expect(err).NotTo(HaveOccurred())
			Expect(res).To(Equal(res))

			updatedNode := v1.Node{}

			err = client.Get(ctx, types.NamespacedName{Name: nodeName}, &updatedNode)
			Expect(err).NotTo(HaveOccurred())
			Expect(updatedNode.Labels).To(HaveKeyWithValue(labelName, kernelVersion))
		})

		It("should set the label if it already exists", func() {
			node := v1.Node{
				ObjectMeta: metav1.ObjectMeta{
					Name:   nodeName,
					Labels: map[string]string{kernelVersion: "4.5.6"},
				},
				Status: v1.NodeStatus{
					NodeInfo: v1.NodeSystemInfo{KernelVersion: kernelVersion},
				},
			}

			client := fake.NewClientBuilder().WithObjects(&node).Build()

			nkr := NewNodeKernelReconciler(client, labelName, nil)
			req := runtimectrl.Request{
				NamespacedName: types.NamespacedName{Name: nodeName},
			}

			ctx := context.TODO()

			res, err := nkr.Reconcile(ctx, req)
			Expect(err).NotTo(HaveOccurred())
			Expect(res).To(Equal(res))

			updatedNode := v1.Node{}

			err = client.Get(ctx, types.NamespacedName{Name: nodeName}, &updatedNode)
			Expect(err).NotTo(HaveOccurred())
			Expect(updatedNode.Labels).To(HaveKeyWithValue(labelName, kernelVersion))
		})
	})
})

package taskruntoolcall

import (
	"context"

	kubechainv1alpha1 "github.com/humanlayer/smallchain/kubechain/api/v1alpha1"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/record"
)

// Test tool instances for different types
var addTool = &TestTool{
	name:     "add",
	toolType: "function",
}

var trtcForAddTool = &TestTaskRunToolCall{
	name:      "test-taskruntoolcall",
	toolName:  addTool.name,
	arguments: `{"a": 2, "b": 3}`,
}

// TestTool represents a test Tool resource
type TestTool struct {
	name     string
	toolType string
	tool     *kubechainv1alpha1.Tool
}

// TestTaskRunToolCall represents a test TaskRunToolCall resource
type TestTaskRunToolCall struct {
	name            string
	toolName        string
	arguments       string
	taskRunToolCall *kubechainv1alpha1.TaskRunToolCall
}

func (t *TestTaskRunToolCall) SetupWithStatus(ctx context.Context, status kubechainv1alpha1.TaskRunToolCallStatus) *kubechainv1alpha1.TaskRunToolCall {
	taskRunToolCall := t.Setup(ctx)
	taskRunToolCall.Status = status
	Expect(k8sClient.Status().Update(ctx, taskRunToolCall)).To(Succeed())
	t.taskRunToolCall = taskRunToolCall
	return taskRunToolCall
}

func (t *TestTaskRunToolCall) Setup(ctx context.Context) *kubechainv1alpha1.TaskRunToolCall {
	By("creating the taskruntoolcall")
	taskRunToolCall := &kubechainv1alpha1.TaskRunToolCall{
		ObjectMeta: metav1.ObjectMeta{
			Name:      t.name,
			Namespace: "default",
		},
		Spec: kubechainv1alpha1.TaskRunToolCallSpec{
			TaskRunRef: kubechainv1alpha1.LocalObjectReference{
				Name: "parent-taskrun",
			},
			ToolRef: kubechainv1alpha1.LocalObjectReference{
				Name: t.toolName,
			},
			Arguments: t.arguments,
		},
	}
	_ = k8sClient.Delete(ctx, taskRunToolCall) // Delete if exists
	err := k8sClient.Create(ctx, taskRunToolCall)
	Expect(err).NotTo(HaveOccurred())
	Expect(k8sClient.Get(ctx, types.NamespacedName{Name: t.name, Namespace: "default"}, taskRunToolCall)).To(Succeed())
	t.taskRunToolCall = taskRunToolCall
	return taskRunToolCall
}

func (t *TestTool) Teardown(ctx context.Context) {
	By("deleting the tool")
	_ = k8sClient.Delete(ctx, t.tool)
}

func (t *TestTool) SetupWithStatus(ctx context.Context, status kubechainv1alpha1.ToolStatus) *kubechainv1alpha1.Tool {
	tool := t.Setup(ctx)
	tool.Status = status
	Expect(k8sClient.Status().Update(ctx, tool)).To(Succeed())
	t.tool = tool
	return tool
}

func (t *TestTool) Setup(ctx context.Context) *kubechainv1alpha1.Tool {
	By("creating the tool")
	tool := &kubechainv1alpha1.Tool{
		ObjectMeta: metav1.ObjectMeta{
			Name:      t.name,
			Namespace: "default",
		},
		Spec: kubechainv1alpha1.ToolSpec{
			ToolType:    t.toolType,
			Name:        t.name,
			Description: "Test tool for " + t.toolType,
			Execute: kubechainv1alpha1.ToolExecute{
				Builtin: &kubechainv1alpha1.BuiltinToolSpec{
					Name: t.name,
				},
			},
		},
	}
	_ = k8sClient.Delete(ctx, tool) // Delete if exists
	err := k8sClient.Create(ctx, tool)
	Expect(err).NotTo(HaveOccurred())
	Expect(k8sClient.Get(ctx, types.NamespacedName{Name: t.name, Namespace: "default"}, tool)).To(Succeed())
	t.tool = tool
	return tool
}

// reconciler creates a new reconciler for testing
// Note, at the time of this writing there's not a lot going on in here
// but future updates will add a MockMCPManager to do things specific to the TRTC flow
func reconciler() (*TaskRunToolCallReconciler, *record.FakeRecorder) {
	By("creating a test reconciler")
	recorder := record.NewFakeRecorder(10)

	reconciler := &TaskRunToolCallReconciler{
		Client:   k8sClient,
		Scheme:   k8sClient.Scheme(),
		recorder: recorder,
	}

	return reconciler, recorder
}

// setupTestTools sets up all the tools needed for testing
func setupTestAddTool(ctx context.Context) func() {
	addTool.SetupWithStatus(ctx, kubechainv1alpha1.ToolStatus{
		Ready:  true,
		Status: "Ready",
	})

	return func() {
		addTool.Teardown(ctx)
	}
}

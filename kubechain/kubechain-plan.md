# Implementation Plan – Enhancing Printer Columns for Tasks and TaskRuns

Below are the concrete steps needed to support two additional printer columns:
1. A “Messages” column (number of items in the TaskRun’s context window)
2. A “UserPreview” column (the first 50 characters of the user’s message) for both Task and TaskRun CRDs

This plan details the changes in the CRDs and controllers.

---

## 1. CRD Changes

### A. For Task CRD
1. In file: **kubechain/api/v1alpha1/task_types.go**, inside the TaskStatus struct, add a new field:
   • Add:
   ```
   // UserPreview stores the first 50 characters of the task’s input.
   UserPreview string `json:"userPreview,omitempty"`
   ```
2. Update the kubebuilder printcolumn annotation (above the Task type definition) to add a new column. For example, modify the annotation:
   • Add a new line:
   ```
   // +kubebuilder:printcolumn:name="UserPreview",type="string",JSONPath=".status.userPreview"
   ```
3. The controller which updates a Task (the TaskReconciler or Task controller) should update the status – set the new field by taking the first 50 characters of `.spec.input` (or the full string if shorter).

### B. For TaskRun CRD
1. In file: **kubechain/api/v1alpha1/taskrun_types.go**, inside the TaskRunStatus struct, add two new fields:
   • Add:
   ```
   // MessageCount contains the number of messages in the context window.
   MessageCount int `json:"messageCount,omitempty"`

   // UserMsgPreview stores the first 50 characters of the user’s message from the context window.
   UserMsgPreview string `json:"userMsgPreview,omitempty"`
   ```
2. Update the kubebuilder printcolumn annotation for TaskRun. For example, modify the existing annotations to add:
   • Add:
   ```
   // +kubebuilder:printcolumn:name="Messages",type="integer",JSONPath=".status.messageCount",priority=1
   // +kubebuilder:printcolumn:name="UserPreview",type="string",JSONPath=".status.userMsgPreview",priority=1
   ```
3. Once the CRD type changes are done, run `make generate` and `make manifests` to update the generated DeepCopy functions and YAML files, then apply the updated CRDs.

---

## 2. Controller Changes

### A. In Task Controller (for Task CRD)
1. In the reconciliation logic (for example, in the TaskReconciler), after validating the task, update the status:
   • Compute:
   ```go
   var preview string
   if len(task.Spec.Input) > 50 {
       preview = task.Spec.Input[:50]
   } else {
       preview = task.Spec.Input
   }
   task.Status.UserPreview = preview
   ```
2. Ensure the Task status is updated with this information.

### B. In TaskRun Controller (for TaskRun CRD)
1. In the TaskRunReconciler (in **kubechain/internal/controller/taskrun_controller.go**), after you populate the ContextWindow (which is a slice of Message), compute:
   • The message count:
   ```go
   statusUpdate.Status.MessageCount = len(statusUpdate.Status.ContextWindow)
   ```
   • The first 50 characters of the user’s message – search through `statusUpdate.Status.ContextWindow` for the message where `Role=="user"`. For example:
   ```go
   userPreview := ""
   for _, msg := range statusUpdate.Status.ContextWindow {
       if msg.Role == "user" {
           if len(msg.Content) > 50 {
               userPreview = msg.Content[:50]
           } else {
               userPreview = msg.Content
           }
           break
       }
   }
   statusUpdate.Status.UserMsgPreview = userPreview
   ```
2. Update the TaskRun status with these new fields.
3. Ensure the TaskRun controller’s reconciliation logic then calls `r.Status().Update(ctx, statusUpdate)`.

---

## 3. Regeneration and Testing

1. Run the following commands to rebuild generated code and update CRDs:
   • `cd kubechain && make generate`
   • `cd kubechain && make manifests`
   • `kubectl apply -f config/crd/bases/kubechain.humanlayer.dev_tasks.yaml -f config/crd/bases/kubechain.humanlayer.dev_taskruns.yaml`
2. Restart controllers and verify using:
   • `kubectl get tasks,taskruns -o wide`
3. Confirm that the Task printer shows a “UserPreview” column (first 50 characters of the user input) and that the TaskRun printer shows both the “Messages” (computed message count) and “UserPreview” column.

This plan provides the precise steps and code changes necessary to meet the feature request.

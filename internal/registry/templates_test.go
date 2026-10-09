package registry

import "testing"

func TestDeployWorkflowTemplate_NonEmpty(t *testing.T) {
	if len(DeployWorkflowTemplate) == 0 {
		t.Error("DeployWorkflowTemplate is empty")
	}
}

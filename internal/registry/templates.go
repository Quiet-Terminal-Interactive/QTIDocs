package registry

import _ "embed"

//go:embed templates/deploy.yml
var DeployWorkflowTemplate []byte

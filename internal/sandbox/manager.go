package sandbox

type SandboxManager interface {
	Setup(workspaceID string) (*Sandbox, error)
	Teardown(workspaceID string) error
	GetSandbox(workspaceID string) (*Sandbox, error)
	ExecuteInSandbox(workspaceID string, cmd string, args []string, envVars map[string]string) (string, error)
}

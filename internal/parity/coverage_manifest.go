package parity

type Coverage struct {
	NativeTags          []string          `json:"native_tags,omitempty"`
	Targets             []string          `json:"targets,omitempty"`
	TargetRun           string            `json:"target_run,omitempty"`
	Package             string            `json:"package"`
	Status              string            `json:"status"`
	PlatformDiagnostics map[string]string `json:"platform_diagnostics,omitempty"`
	Diagnostic          string            `json:"diagnostic,omitempty"`
}

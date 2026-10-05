package preview

import (
	"grubmgr/internal/output"
	"grubmgr/internal/system"
)

// Adapter is the future boundary for a separately installed, constrained VM renderer.
type Adapter interface {
	Preview(themeDirectory string) error
}
type External struct{ System system.Report }

func (e External) Preview(_ string) error {
	if len(e.System.MissingOptional) > 0 {
		return output.Fail(output.Unsupported, "PREVIEW_DEPENDENCIES", "optional dependencies missing: %v", e.System.MissingOptional)
	}
	return output.Fail(output.Unsupported, "PREVIEW_DISABLED", "dependencies detected; constrained external adapter has not passed disposable-VM review")
}

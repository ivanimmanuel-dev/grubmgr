package preview

import (
	"grubmgr/internal/output"
	"grubmgr/internal/system"
)

// Adapter reports preview availability for synthetic fixtures.
type Adapter interface {
	Preview(themeDirectory string) error
}
type External struct{ System system.Report }

func (e External) Preview(_ string) error {
	if len(e.System.MissingOptional) > 0 {
		return output.Fail(output.Unsupported, "PREVIEW_DEPENDENCIES", "optional dependencies missing: %v", e.System.MissingOptional)
	}
	return output.Fail(output.Unsupported, "PREVIEW_DISABLED", "preview is unavailable in fixture mode; use the installed Linux CLI without --root")
}

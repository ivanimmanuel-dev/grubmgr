package cli

import (
	"bufio"
	"fmt"
	"grubmgr/internal/fetch"
	"grubmgr/internal/output"
	"grubmgr/internal/planner"
	"grubmgr/internal/state"
	"grubmgr/internal/system"
	"io"
	"os"
	"runtime"
	"strings"
)

func direct(o options, in io.Reader, out, diagnostics io.Writer) (any, error) {
	if len(o.args) != 2 {
		return nil, output.Fail(output.Usage, "ARGUMENT", "%s expects one target", o.args[0])
	}
	if o.json && !o.yes {
		return nil, output.Fail(output.Usage, "CONFIRMATION", "use --yes with --json for direct operations, or use plan and apply")
	}
	action, target := o.args[0], o.args[1]
	if action == "install" {
		p, err := system.Locations(o.root)
		if err != nil {
			return nil, err
		}
		if !p.Fixture && runtime.GOOS != "windows" && os.Geteuid() == 0 {
			return nil, output.Fail(output.Unsupported, "UNPRIVILEGED_REQUIRED", "run the CLI as an ordinary user")
		}
		db, err := state.Open(p, false)
		if err != nil {
			return nil, err
		}
		packages, err := db.Packages()
		db.Close()
		if err != nil {
			return nil, err
		}
		pkg, selectionErr := state.Select(packages, canonical(target, p.Config))
		if selectionErr != nil {
			for _, existing := range packages {
				if existing.Manifest.ID == canonical(target, p.Config) {
					return nil, selectionErr
				}
			}
			pkg, err = fetch.Import(p, target, o.recipe)
			if err != nil {
				return nil, err
			}
			if !pkg.Validation.Valid {
				return nil, output.Fail(output.Invalid, "VALIDATION_FAILED", "%v", pkg.Validation.Findings)
			}
		}
		target = pkg.Manifest.ID + "@" + pkg.Manifest.Revision
	}
	planOptions := o
	planOptions.args = []string{"plan", action, target}
	value, err := execute(planOptions)
	if err != nil {
		return nil, err
	}
	plan, ok := value.(planner.Plan)
	if !ok || !plan.Applicable {
		return nil, output.Fail(output.Unsupported, "PLAN_REFUSED", "%s", plan.Reason)
	}
	if !o.json {
		label := plan.ThemeID
		if label == "" {
			label = target
		}
		fmt.Fprintf(out, "%s %s\n", strings.ToUpper(action[:1])+action[1:], label)
		if plan.Destination != "" {
			fmt.Fprintf(out, "Destination: %s\n", plan.Destination)
		}
		if plan.AfterTheme != plan.BeforeTheme {
			fmt.Fprintf(out, "Theme: %q → %q\n", plan.BeforeTheme, plan.AfterTheme)
		}
		if plan.ThemeBackup != "" {
			fmt.Fprintf(out, "Previous theme backup: %s\n", plan.ThemeBackup)
		}
		if len(plan.Generator) != 0 {
			fmt.Fprintln(out, "The GRUB menu will be regenerated with the installed kernels.")
		}
	}
	if !o.yes {
		fmt.Fprint(diagnostics, "Apply this change? [y/N] ")
		answer, _ := bufio.NewReader(in).ReadString('\n')
		if answer = strings.ToLower(strings.TrimSpace(answer)); answer != "y" && answer != "yes" {
			return nil, output.Fail(output.Usage, "CANCELLED", "change cancelled")
		}
	}
	applyOptions := o
	applyOptions.args = []string{"apply", plan.ID}
	return execute(applyOptions)
}

package preview

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"grubmgr/internal/fsx"
	"grubmgr/internal/model"
	"grubmgr/internal/output"
	"grubmgr/internal/validate"
	"image"
	"image/png"
	"io"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"time"
)

const menu = `set timeout=-1
insmod serial
serial --unit=0 --speed=115200
terminal_output --append serial
echo GRUBMGR_PREVIEW_READY
menuentry 'Linux preview' { echo 'Preview only'; sleep 60; }
menuentry 'Recovery preview' { echo 'Preview only'; sleep 60; }
`
const renderer = "/usr/local/bin/grub2-theme-preview"
const qemuAdapter = "/usr/libexec/grubmgr-preview-qemu"

type Result struct {
	Kind        string `json:"kind"`
	Image       string `json:"image"`
	Description string `json:"description"`
}

func Capture(source, cache string, m model.Manifest, variantID string) (Result, error) {
	var result Result
	if runtime.GOOS != "linux" || os.Geteuid() == 0 {
		return result, output.Fail(output.Unsupported, "PREVIEW_PLATFORM", "preview requires an ordinary Linux user")
	}
	missing := []string{}
	code, vars, err := firmware()
	if err != nil {
		return result, output.Fail(output.Unsupported, "PREVIEW_DEPENDENCIES", "%v", err)
	}
	for _, name := range []string{renderer, qemuAdapter, "/usr/bin/bwrap", "/usr/bin/prlimit", "/usr/bin/qemu-system-x86_64", "/usr/bin/grub-mkrescue", "/usr/bin/xorriso", "/usr/bin/mformat"} {
		if _, err := os.Stat(name); err != nil {
			missing = append(missing, name)
		}
	}
	if len(missing) > 0 {
		return result, output.Fail(output.Unsupported, "PREVIEW_DEPENDENCIES", "missing: %s", strings.Join(missing, ", "))
	}
	if v := validate.Run(source, m); !v.Valid {
		return result, output.Fail(output.Invalid, "PREVIEW_PACKAGE", "theme validation failed")
	}
	variant, err := m.Variant(variantID)
	if err != nil {
		return result, err
	}
	if variant.Entry != "theme.txt" {
		return result, fmt.Errorf("external preview requires theme.txt")
	}
	if err = fsx.NoLinks(cache); err != nil {
		return result, err
	}
	if err = os.MkdirAll(cache, 0700); err != nil {
		return result, err
	}
	dir, err := os.MkdirTemp(cache, "preview-")
	if err != nil {
		return result, err
	}
	if err = os.WriteFile(filepath.Join(dir, "menu.cfg"), []byte(menu), 0600); err != nil {
		return result, err
	}
	args := []string{"--unshare-all", "--die-with-parent", "--new-session", "--dir", "/firmware", "--ro-bind", code, "/firmware/code.fd", "--ro-bind", vars, "/firmware/vars.fd", "--ro-bind", "/usr", "/usr", "--symlink", "usr/lib", "/lib", "--symlink", "usr/lib64", "/lib64", "--proc", "/proc", "--dev", "/dev", "--tmpfs", "/tmp", "--dir", "/run", "--dir", "/home", "--dir", "/sys", "--dir", "/sys/firmware", "--dir", "/sys/firmware/efi", "--dir", "/etc", "--ro-bind", "/etc/ld.so.cache", "/etc/ld.so.cache", "--ro-bind", filepath.Join(source, variant.Root), "/theme", "--bind", dir, "/output", "--clearenv", "--setenv", "G2TP_OVMF_IMAGE", "/firmware/code.fd", "--setenv", "PATH", "/usr/local/bin:/usr/bin:/bin", "--setenv", "HOME", "/tmp", "--setenv", "LANG", "C", "--setenv", "TMPDIR", "/tmp", "--chdir", "/tmp", "/usr/bin/prlimit", "--as=4294967296", "--cpu=90", "--fsize=268435456", "--nofile=128", "--", renderer, "--grub-cfg", "/output/menu.cfg", "--qemu", qemuAdapter, "--grub2-mkrescue", "/usr/bin/grub-mkrescue", "--xorriso", "/usr/bin/xorriso", "--display", "none", "--no-kvm", "--timeout", "-1", "--resolution", "1024x768", "/theme"}
	ctx, cancel := context.WithTimeout(context.Background(), 360*time.Second)
	// Upstream otherwise hides child stderr, including adapter diagnostics.
	args = append(args[:len(args)-1], "--verbose", args[len(args)-1])
	for i := range args {
		if args[i] == "--cpu=90" {
			args[i] = "--cpu=300"
		}
	}
	defer cancel()
	cmd := exec.CommandContext(ctx, "/usr/bin/bwrap", args...)
	cmd.Env = []string{"PATH=/usr/bin:/bin", "LANG=C"}
	var log limitedLog
	cmd.Stdout = &log
	cmd.Stderr = &log
	err = cmd.Run()
	_ = os.WriteFile(filepath.Join(dir, "preview.log"), log.Bytes(), 0600)
	if err != nil {
		return result, output.Fail(output.Unsupported, "PREVIEW_FAILED", "sandboxed preview failed: %v; log: %s", err, filepath.Join(dir, "preview.log"))
	}
	if err = ppmToPNG(filepath.Join(dir, "preview.ppm"), filepath.Join(dir, "preview.png")); err != nil {
		return result, err
	}
	return Result{Kind: "grub-qemu", Image: filepath.Join(dir, "preview.png"), Description: "GRUB rendered a generated menu in an isolated VM; this does not verify a physical boot"}, nil
}

type limitedLog struct{ bytes.Buffer }

func (b *limitedLog) Write(p []byte) (int, error) {
	if b.Len()+len(p) > 1<<20 {
		return 0, fmt.Errorf("preview log limit")
	}
	return b.Buffer.Write(p)
}

// QEMU runs inside the renderer's unprivileged mount/network namespace.
// The external adapter can select only its bounded temporary raw image.
func QEMU(args []string) error {
	if runtime.GOOS != "linux" || os.Geteuid() == 0 {
		return fmt.Errorf("preview runs as an ordinary Linux user")
	}
	imagePath := ""
	for i := 0; i < len(args); i += 2 {
		if i+1 >= len(args) {
			return fmt.Errorf("unexpected renderer argument")
		}
		switch args[i] {
		case "-m":
			if args[i+1] != "256" {
				return fmt.Errorf("unexpected memory size")
			}
		case "-display":
			if args[i+1] != "none" {
				return fmt.Errorf("unexpected display")
			}
		case "-drive":
			value := args[i+1]
			if value == "if=pflash,format=raw,readonly=on,file=/firmware/code.fd" {
				continue
			}
			if imagePath != "" || !strings.HasPrefix(value, "file=/tmp/") || !strings.HasSuffix(value, ",index=0,media=disk,format=raw") {
				return fmt.Errorf("unexpected drive")
			}
			imagePath = strings.TrimSuffix(strings.TrimPrefix(value, "file="), ",index=0,media=disk,format=raw")
		default:
			return fmt.Errorf("unexpected renderer argument")
		}
	}
	if imagePath == "" || filepath.Clean(imagePath) != imagePath || !strings.HasPrefix(imagePath, "/tmp/") || fsx.NoLinks(imagePath) != nil {
		return fmt.Errorf("invalid preview image")
	}
	info, err := os.Stat(imagePath)
	if err != nil || !info.Mode().IsRegular() || info.Size() > 256<<20 {
		return fmt.Errorf("invalid preview image size")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 300*time.Second)
	defer cancel()
	// OVMF needs a matching variable store. This private copy is discarded with
	// the namespace and never reads or writes the host firmware variable store.
	vars, err := os.ReadFile("/firmware/vars.fd")
	if err != nil {
		return err
	}
	if len(vars) > 1<<20 {
		return fmt.Errorf("unexpected firmware variable template")
	}
	if err = os.WriteFile("/tmp/grubmgr-preview-vars.fd", vars, 0600); err != nil {
		return err
	}
	cmd := exec.CommandContext(ctx, "/usr/bin/qemu-system-x86_64", "-machine", "q35,smm=on", "-accel", "tcg", "-m", "256", "-net", "none", "-display", "none", "-monitor", "none", "-serial", "file:/output/grub-serial.log", "-drive", "file="+imagePath+",format=raw,media=cdrom,readonly=on", "-drive", "if=pflash,format=raw,readonly=on,file=/firmware/code.fd", "-drive", "if=pflash,format=raw,file=/tmp/grubmgr-preview-vars.fd", "-qmp", "unix:/tmp/grubmgr-preview.sock,server=on,wait=off")
	cmd.Env = []string{"PATH=/usr/bin:/bin", "LANG=C"}
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err = cmd.Start(); err != nil {
		return err
	}
	defer func() { _ = cmd.Process.Kill(); _ = cmd.Wait() }()
	var conn net.Conn
	for until := time.Now().Add(8 * time.Second); time.Now().Before(until); time.Sleep(100 * time.Millisecond) {
		conn, err = net.Dial("unix", "/tmp/grubmgr-preview.sock")
		if err == nil {
			break
		}
	}
	if err != nil {
		return err
	}
	defer conn.Close()
	_ = conn.SetDeadline(time.Now().Add(290 * time.Second))
	decoder, encoder := json.NewDecoder(conn), json.NewEncoder(conn)
	var response map[string]any
	if err = decoder.Decode(&response); err != nil {
		return err
	}
	request := func(value any) error {
		if err := encoder.Encode(value); err != nil {
			return err
		}
		for {
			var answer map[string]any
			if err := decoder.Decode(&answer); err != nil {
				return err
			}
			if e, ok := answer["error"]; ok {
				return fmt.Errorf("QMP: %v", e)
			}
			if _, ok := answer["return"]; ok {
				return nil
			}
		}
	}
	if err = request(map[string]any{"execute": "qmp_capabilities"}); err != nil {
		return err
	}
	ready := false
	for until := time.Now().Add(240 * time.Second); time.Now().Before(until); time.Sleep(time.Second) {
		data, err := os.ReadFile("/output/grub-serial.log")
		if err == nil && bytes.Contains(data, []byte("GRUBMGR_PREVIEW_READY")) && bytes.Contains(data, []byte("Linux preview")) {
			ready = true
			break
		}
		if ctx.Err() != nil {
			return ctx.Err()
		}
	}
	if !ready {
		return fmt.Errorf("GRUB did not reach the generated menu; see grub-serial.log")
	}
	// Allow the renderer's theme assignment and initial menu paint to complete.
	time.Sleep(15 * time.Second)
	if err = request(map[string]any{"execute": "screendump", "arguments": map[string]any{"filename": "/output/preview.ppm"}}); err != nil {
		return err
	}
	return request(map[string]any{"execute": "quit"})
}

func ppmToPNG(source, dest string) error {
	f, err := os.Open(source)
	if err != nil {
		return err
	}
	defer f.Close()
	r := bufio.NewReader(io.LimitReader(f, 50<<20))
	magic, _ := r.ReadString('\n')
	size, _ := r.ReadString('\n')
	maximum, _ := r.ReadString('\n')
	fields := strings.Fields(size)
	if strings.TrimSpace(magic) != "P6" || strings.TrimSpace(maximum) != "255" || len(fields) != 2 {
		return fmt.Errorf("invalid QEMU screenshot")
	}
	w, e1 := strconv.Atoi(fields[0])
	h, e2 := strconv.Atoi(fields[1])
	if e1 != nil || e2 != nil || w < 1 || h < 1 || w > 4096 || h > 4096 {
		return fmt.Errorf("screenshot size limit")
	}
	pixels := make([]byte, w*h*3)
	if _, err = io.ReadFull(r, pixels); err != nil {
		return err
	}
	if w >= 640 && h >= 480 {
		visible := false
		for i := 3; i < len(pixels); i += 3 {
			if !bytes.Equal(pixels[:3], pixels[i:i+3]) {
				visible = true
				break
			}
		}
		if !visible {
			return fmt.Errorf("QEMU returned a blank frame; preview is not verified")
		}
	}
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	for i := 0; i < w*h; i++ {
		copy(img.Pix[i*4:i*4+3], pixels[i*3:i*3+3])
		img.Pix[i*4+3] = 255
	}
	out, err := os.OpenFile(dest, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0600)
	if err != nil {
		return err
	}
	err = png.Encode(out, img)
	closeErr := out.Close()
	if err != nil {
		return err
	}
	return closeErr
}

package goeditor

import (
	"errors"
	"os/exec"
	"runtime"
	"strings"
)

// ErrNothingToOpen is returned by OpenWithSystem for an empty target.
var ErrNothingToOpen = errors.New("nothing to open")

// OpenWithSystem opens target, such as the one in an OpenMsg, with the
// system's default handler: open on macOS, xdg-open on Linux and the BSDs,
// and the URL handler on Windows. It returns once the handler has started.
// A relative file name is resolved against the process's working directory.
//
// The handler runs on the machine running the program, so for an editor
// served over SSH it opens nothing on the user's machine.
func OpenWithSystem(target string) error {
	if target == "" {
		return ErrNothingToOpen
	}
	// Keep a file name such as "-a" from being read as a flag.
	if strings.HasPrefix(target, "-") {
		target = "./" + target
	}

	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "darwin":
		cmd = exec.Command("open", target)
	case "windows":
		cmd = exec.Command("rundll32", "url.dll,FileProtocolHandler", target)
	default:
		cmd = exec.Command("xdg-open", target)
	}
	if err := cmd.Start(); err != nil {
		return err
	}
	go func() { _ = cmd.Wait() }() // reap the handler process
	return nil
}

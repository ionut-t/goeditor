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
//
// The handler does whatever the system does with target: it can start the
// app registered for any URL scheme (zoommtg://, vscode://, …) and run a
// local program or script (.app, .command, .exe, .bat). If the buffer may
// hold untrusted text, check the target first, for example:
//
//	if u, err := url.Parse(msg.Target); err == nil && (u.Scheme == "http" || u.Scheme == "https") {
//		err = goeditor.OpenWithSystem(msg.Target)
//	}
func OpenWithSystem(target string) error {
	if target == "" {
		return ErrNothingToOpen
	}
	cmd := openCommand(runtime.GOOS, target)
	if err := cmd.Start(); err != nil {
		return err
	}
	go func() { _ = cmd.Wait() }() // reap the handler process
	return nil
}

// openCommand returns the command that opens target on goos. target is
// passed as a single argument, never through a shell.
func openCommand(goos, target string) *exec.Cmd {
	if goos == "windows" {
		return exec.Command("rundll32", "url.dll,FileProtocolHandler", target)
	}
	// open and xdg-open read an argument starting with '-' as a flag, and
	// xdg-open doesn't accept "--", so make such a file name a path.
	if strings.HasPrefix(target, "-") {
		target = "./" + target
	}
	if goos == "darwin" {
		return exec.Command("open", target)
	}
	return exec.Command("xdg-open", target)
}

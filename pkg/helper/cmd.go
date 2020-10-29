package helper

import (
	"os"
	"os/exec"
	"strings"
	"syscall"

	"github.com/eko/monday/pkg/log"
)

// BuildCmd builds a *exec.Cmd struct with some unified rules such as setting stdout/err, directory path,
// list of commands to run under and environment variables
func BuildCmd(commandsList []string, path string, stdout, stderr *log.Streamer) *exec.Cmd {
	var commands = strings.Join(commandsList, "; ")

	commands = strings.Replace(commands, "~", "$HOME", -1)
	commands = os.ExpandEnv(commands)

	cmd := exec.Command("/bin/sh", "-c", strings.Replace(commands, "'", "\\'", -1))
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	cmd.Dir = path
	cmd.Stdout = stdout
	cmd.Stderr = stderr
	cmd.Env = os.Environ()

	return cmd
}

package run

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"regexp"
)

// addEnvVariables adds environment variables given as key/value pair
func (r *Runner) addEnvVariables(cmd *exec.Cmd, envs map[string]string) {
	for key, value := range envs {
		cmd.Env = append(cmd.Env, fmt.Sprintf("%s=%s", key, value))
	}
}

// addEnvVariablesFromFile adds environment variables given as a filename
func (r *Runner) addEnvVariablesFromFile(cmd *exec.Cmd, filename string) {
	if filename == "" {
		return
	}

	filename = os.ExpandEnv(filename)

	file, err := os.OpenFile(filename, os.O_RDONLY, os.ModePerm)
	if err != nil {
		r.view.Writef("❌  Unable to open environment file '%s': %v\n", filename, err)
		return
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()

		r, _ := regexp.Compile("([a-zA-Z0-9_]+)=(.*)")
		matches := r.FindStringSubmatch(line)

		if len(matches) < 3 {
			continue
		}

		cmd.Env = append(cmd.Env, fmt.Sprintf("%s=%s", matches[1], matches[2]))
	}

	if err := scanner.Err(); err != nil {
		r.view.Writef("❌  An error has occured while reading environment file '%s': %v\n", filename, err)
		return
	}
}

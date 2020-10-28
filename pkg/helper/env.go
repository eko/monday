package helper

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"regexp"
)

// AddEnvVariables adds environment variables given as key/value pair
func AddEnvVariables(cmd *exec.Cmd, envs map[string]string) {
	for key, value := range envs {
		cmd.Env = append(cmd.Env, fmt.Sprintf("%s=%s", key, value))
	}
}

// AddEnvVariablesFromFile adds environment variables given as a filename
func AddEnvVariablesFromFile(cmd *exec.Cmd, filename string) error {
	if filename == "" {
		return nil
	}

	filename = os.ExpandEnv(filename)

	file, err := os.OpenFile(filename, os.O_RDONLY, os.ModePerm)
	if err != nil {
		return fmt.Errorf("unable to open environment file '%s': %v", filename, err)
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
		return fmt.Errorf("an error has occured while reading environment file '%s': %v", filename, err)
	}

	return nil
}

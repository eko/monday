package main

import (
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"runtime"

	"github.com/spf13/cobra"
)

const (
	latestReleaseURL = "https://api.github.com/repos/policygenius/monday/releases/latest"
	binaryURLPattern = "https://github.com/policygenius/monday/releases/latest/download/monday-%s-%s"
	binaryFilepath   = "/usr/local/bin/monday"
)

type GithubAPIResponse struct {
	TagName string `json:"tag_name"`
}

var upgradeCmd = &cobra.Command{
	Use:   "upgrade",
	Short: "This command upgrades Monday to its latest version",
	Long:  `In case a new version of Monday is available, this command will download it and install it locally for you.`,
	Run: func(cmd *cobra.Command, args []string) {
		response, err := http.Get(latestReleaseURL)
		if err != nil {
			fmt.Printf("❌  An error has occured while contacting GitHub API: %v\n", err)
			return
		}

		data, _ := ioutil.ReadAll(response.Body)

		githubResponse := &GithubAPIResponse{}
		json.Unmarshal(data, githubResponse)

		if githubResponse.TagName == Version {
			fmt.Printf("✅  You are already on the latest version: %s\n", githubResponse.TagName)
			return
		}

		fmt.Printf("🐢  A new version is available. Current is %s and %s is available\n", Version, githubResponse.TagName)

		// Download the right binary depending on OS and architecture
		fmt.Printf("👉  Downloading binary for your OS (%s/%s)...\n", runtime.GOOS, runtime.GOARCH)

		binaryURL := fmt.Sprintf(binaryURLPattern, runtime.GOOS, runtime.GOARCH)

		resp, err := http.Get(binaryURL)
		if err != nil {
			fmt.Printf("❌  An error has occured while trying to download binary from URL: %s\n", binaryURL)
			return
		}
		defer resp.Body.Close()

		// Create the binary file
		out, err := os.Create(binaryFilepath)
		if err != nil {
			fmt.Printf("❌  An error has occured while trying to create binary file to: %s\n", binaryFilepath)
			return
		}
		defer out.Close()

		// Write binary
		_, err = io.Copy(out, resp.Body)
		if err != nil {
			fmt.Printf("❌  An error has occured while trying to copy binary file to: %s\n", binaryFilepath)
			return
		}

		fmt.Printf("✅  Monday has been successfully upgraded. You are now on latest version: %s\n", githubResponse.TagName)
		return
	},
}

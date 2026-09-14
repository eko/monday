package main

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/spf13/cobra"
)

const (
	latestReleaseURL = "https://api.github.com/repos/eko/monday/releases/latest"
	releaseAssetURL  = "https://github.com/eko/monday/releases/download/%s/monday-%s-%s"

	apiTimeout      = 15 * time.Second
	downloadTimeout = 10 * time.Minute
)

var errNoChecksum = errors.New("no checksum published for this release")

type githubRelease struct {
	TagName string `json:"tag_name"`
}

// downloadedFile is a file downloaded on disk with its SHA-256 checksum
type downloadedFile struct {
	path     string
	checksum string
}

var upgradeCmd = &cobra.Command{
	Use:   "upgrade",
	Short: "Upgrade Monday to its latest version",
	Long: `In case a new version of Monday is available, this command downloads the binary
built for your system, verifies its checksum and replaces the current binary.`,
	Run: func(cmd *cobra.Command, args []string) {
		if err := upgrade(); err != nil {
			fmt.Printf("❌  %v\n", err)
			os.Exit(1)
		}
	},
}

func upgrade() error {
	ctx, cancel := context.WithTimeout(context.Background(), downloadTimeout)
	defer cancel()

	client := &http.Client{}

	latest, err := fetchLatestTag(ctx, client, latestReleaseURL)
	if err != nil {
		return err
	}

	if latest == Version {
		fmt.Printf("✅  You are already on the latest version: %s\n", latest)
		return nil
	}

	fmt.Printf("🐢  A new version is available: %s (current is %s)\n", latest, Version)

	target, err := currentBinaryPath()
	if err != nil {
		return fmt.Errorf("unable to locate the current binary: %w", err)
	}

	if isHomebrewPath(target) {
		fmt.Println("🍺  Monday is installed with Homebrew, please upgrade it with: brew upgrade monday")
		return nil
	}

	assetURL := fmt.Sprintf(releaseAssetURL, latest, runtime.GOOS, runtime.GOARCH)

	fmt.Printf("👉  Downloading binary for your system (%s/%s)...\n", runtime.GOOS, runtime.GOARCH)

	download, err := downloadFile(ctx, client, assetURL, filepath.Dir(target))
	if err != nil {
		return withPrivilegesHint(err)
	}
	defer os.Remove(download.path)

	expected, err := fetchChecksum(ctx, client, assetURL+".sha256")
	switch {
	case errors.Is(err, errNoChecksum):
		fmt.Println("⚠️   No checksum is published for this release, skipping the verification")

	case err != nil:
		return err

	case expected != download.checksum:
		return fmt.Errorf("checksum mismatch for the downloaded binary (expected %s, got %s)", expected, download.checksum)

	default:
		fmt.Println("🔏  Checksum verified")
	}

	if err := os.Chmod(download.path, 0o755); err != nil {
		return withPrivilegesHint(err)
	}

	if err := os.Rename(download.path, target); err != nil {
		return withPrivilegesHint(fmt.Errorf("unable to replace %s: %w", target, err))
	}

	fmt.Printf("✅  Monday has been successfully upgraded to %s (%s)\n", latest, target)

	return nil
}

// fetchLatestTag returns the tag name of the latest published release
func fetchLatestTag(
	ctx context.Context,
	client *http.Client,
	url string,
) (string, error) {
	ctx, cancel := context.WithTimeout(ctx, apiTimeout)
	defer cancel()

	response, err := get(ctx, client, url, "application/vnd.github+json")
	if err != nil {
		return "", fmt.Errorf("unable to contact the GitHub API: %w", err)
	}
	defer response.Body.Close()

	if response.StatusCode != http.StatusOK {
		return "", fmt.Errorf("unexpected response from the GitHub API: %s", response.Status)
	}

	var release githubRelease
	if err := json.NewDecoder(response.Body).Decode(&release); err != nil {
		return "", fmt.Errorf("unable to read the GitHub API response: %w", err)
	}

	if release.TagName == "" {
		return "", errors.New("the GitHub API did not return any release tag")
	}

	return release.TagName, nil
}

// downloadFile downloads the given URL into a temporary file of the given
// directory, computing its checksum on the fly
func downloadFile(
	ctx context.Context,
	client *http.Client,
	url string,
	dir string,
) (*downloadedFile, error) {
	response, err := get(ctx, client, url, "application/octet-stream")
	if err != nil {
		return nil, fmt.Errorf("unable to download %s: %w", url, err)
	}
	defer response.Body.Close()

	if response.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unable to download %s: %s", url, response.Status)
	}

	file, err := os.CreateTemp(dir, ".monday-upgrade-*")
	if err != nil {
		return nil, fmt.Errorf("unable to create the temporary file in %s: %w", dir, err)
	}
	defer file.Close()

	hash := sha256.New()

	if _, err := io.Copy(io.MultiWriter(file, hash), response.Body); err != nil {
		os.Remove(file.Name())
		return nil, fmt.Errorf("unable to download %s: %w", url, err)
	}

	return &downloadedFile{
		path:     file.Name(),
		checksum: hex.EncodeToString(hash.Sum(nil)),
	}, nil
}

// fetchChecksum returns the SHA-256 checksum published next to a release
// asset, in the "<checksum>  <filename>" format of the shasum command
func fetchChecksum(
	ctx context.Context,
	client *http.Client,
	url string,
) (string, error) {
	ctx, cancel := context.WithTimeout(ctx, apiTimeout)
	defer cancel()

	response, err := get(ctx, client, url, "text/plain")
	if err != nil {
		return "", fmt.Errorf("unable to download the checksum %s: %w", url, err)
	}
	defer response.Body.Close()

	if response.StatusCode == http.StatusNotFound {
		return "", errNoChecksum
	}

	if response.StatusCode != http.StatusOK {
		return "", fmt.Errorf("unable to download the checksum %s: %s", url, response.Status)
	}

	body, err := io.ReadAll(io.LimitReader(response.Body, 1024))
	if err != nil {
		return "", fmt.Errorf("unable to read the checksum %s: %w", url, err)
	}

	fields := strings.Fields(string(body))
	if len(fields) == 0 || len(fields[0]) != sha256.Size*2 {
		return "", fmt.Errorf("invalid checksum content at %s", url)
	}

	return strings.ToLower(fields[0]), nil
}

func get(
	ctx context.Context,
	client *http.Client,
	url string,
	accept string,
) (*http.Response, error) {
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}

	request.Header.Set("Accept", accept)
	request.Header.Set("User-Agent", "monday/"+Version)

	return client.Do(request)
}

// currentBinaryPath returns the real path of the running binary, symlinks resolved
func currentBinaryPath() (string, error) {
	path, err := os.Executable()
	if err != nil {
		return "", err
	}

	return filepath.EvalSymlinks(path)
}

// isHomebrewPath returns true when the binary lives in a Homebrew cellar,
// whatever the prefix (/usr/local, /opt/homebrew, /home/linuxbrew/.linuxbrew)
func isHomebrewPath(path string) bool {
	return strings.Contains(path, "/Cellar/")
}

func withPrivilegesHint(err error) error {
	if errors.Is(err, fs.ErrPermission) {
		return fmt.Errorf("%w (try running the upgrade with elevated privileges: sudo monday upgrade)", err)
	}

	return err
}

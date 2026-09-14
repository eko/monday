package main

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNormalizeVersion(t *testing.T) {
	testCases := []struct {
		version string
		want    string
	}{
		{version: "refs/tags/v2.6.1", want: "v2.6.1"},
		{version: "v2.6.1", want: "v2.6.1"},
		{version: "sources-f4681", want: "sources-f4681"},
		{version: " v2.6.1\n", want: "v2.6.1"},
		{version: "", want: ""},
	}

	for _, testCase := range testCases {
		t.Run(testCase.version, func(t *testing.T) {
			assert.Equal(t, testCase.want, normalizeVersion(testCase.version))
		})
	}
}

func TestIsHomebrewPath(t *testing.T) {
	testCases := []struct {
		path string
		want bool
	}{
		{path: "/opt/homebrew/Cellar/monday/2.6.1/bin/monday", want: true},
		{path: "/usr/local/Cellar/monday/2.6.1/bin/monday", want: true},
		{path: "/home/linuxbrew/.linuxbrew/Cellar/monday/2.6.1/bin/monday", want: true},
		{path: "/usr/local/bin/monday", want: false},
		{path: "/Users/vincent/go/bin/monday", want: false},
	}

	for _, testCase := range testCases {
		t.Run(testCase.path, func(t *testing.T) {
			assert.Equal(t, testCase.want, isHomebrewPath(testCase.path))
		})
	}
}

func TestFetchLatestTag(t *testing.T) {
	testCases := []struct {
		name        string
		status      int
		body        string
		want        string
		expectedErr string
	}{
		{
			name:   "release found",
			status: http.StatusOK,
			body:   `{"tag_name": "v2.7.0", "name": "v2.7.0"}`,
			want:   "v2.7.0",
		},
		{
			name:        "api error",
			status:      http.StatusForbidden,
			body:        `{"message": "rate limited"}`,
			expectedErr: "unexpected response from the GitHub API: 403 Forbidden",
		},
		{
			name:        "invalid json",
			status:      http.StatusOK,
			body:        `not json`,
			expectedErr: "unable to read the GitHub API response: invalid character 'o' in literal null (expecting 'u')",
		},
		{
			name:        "missing tag",
			status:      http.StatusOK,
			body:        `{}`,
			expectedErr: "the GitHub API did not return any release tag",
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				assert.Equal(t, "application/vnd.github+json", r.Header.Get("Accept"))
				assert.Contains(t, r.Header.Get("User-Agent"), "monday/")

				w.WriteHeader(testCase.status)
				_, _ = w.Write([]byte(testCase.body))
			}))
			defer server.Close()

			tag, err := fetchLatestTag(context.Background(), server.Client(), server.URL)

			if testCase.expectedErr != "" {
				assert.EqualError(t, err, testCase.expectedErr)
				return
			}

			require.NoError(t, err)
			assert.Equal(t, testCase.want, tag)
		})
	}
}

func TestFetchChecksum(t *testing.T) {
	valid := "9f86d081884c7d659a2feaa0c55ad015a3bf4f1b2b0b822cd15d6c15b0f00a08"

	testCases := []struct {
		name        string
		status      int
		body        string
		want        string
		expectedErr error
		errContains string
	}{
		{
			name:   "shasum format",
			status: http.StatusOK,
			body:   valid + "  monday-darwin-arm64\n",
			want:   valid,
		},
		{
			name:   "bare checksum, uppercase",
			status: http.StatusOK,
			body:   "9F86D081884C7D659A2FEAA0C55AD015A3BF4F1B2B0B822CD15D6C15B0F00A08",
			want:   valid,
		},
		{
			name:        "not published",
			status:      http.StatusNotFound,
			expectedErr: errNoChecksum,
		},
		{
			name:        "server error",
			status:      http.StatusInternalServerError,
			errContains: "500 Internal Server Error",
		},
		{
			name:        "invalid content",
			status:      http.StatusOK,
			body:        "<html>",
			errContains: "invalid checksum content",
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(testCase.status)
				_, _ = w.Write([]byte(testCase.body))
			}))
			defer server.Close()

			checksum, err := fetchChecksum(context.Background(), server.Client(), server.URL)

			switch {
			case testCase.expectedErr != nil:
				assert.ErrorIs(t, err, testCase.expectedErr)
			case testCase.errContains != "":
				require.Error(t, err)
				assert.Contains(t, err.Error(), testCase.errContains)
			default:
				require.NoError(t, err)
				assert.Equal(t, testCase.want, checksum)
			}
		})
	}
}

func TestDownloadFile(t *testing.T) {
	// Given
	content := []byte("#!/bin/sh\necho monday\n")
	sum := sha256.Sum256(content)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write(content)
	}))
	defer server.Close()

	dir := t.TempDir()

	// When
	download, err := downloadFile(context.Background(), server.Client(), server.URL, dir)

	// Then
	require.NoError(t, err)
	assert.Equal(t, hex.EncodeToString(sum[:]), download.checksum)
	assert.Equal(t, dir, filepath.Dir(download.path))

	written, err := os.ReadFile(download.path)
	require.NoError(t, err)
	assert.Equal(t, content, written)
}

func TestDownloadFileWhenNotFound(t *testing.T) {
	// Given
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	// When
	download, err := downloadFile(context.Background(), server.Client(), server.URL, t.TempDir())

	// Then
	assert.Nil(t, download)
	assert.EqualError(t, err, "unable to download "+server.URL+": 404 Not Found")
}

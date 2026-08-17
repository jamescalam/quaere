// Package gcp handles the Google Cloud authentication needed for the Cloud SQL
// IAM connection method: detecting whether usable Application Default
// Credentials exist, and driving the browser-based `gcloud` login when they
// don't.
package gcp

import (
	"context"
	"os/exec"

	"golang.org/x/oauth2/google"
)

// cloudPlatformScope is the OAuth scope Cloud SQL IAM auth requires.
const cloudPlatformScope = "https://www.googleapis.com/auth/cloud-platform"

// HasADC reports whether Application Default Credentials are available and can
// mint a token. This is the same credential source the Cloud SQL dialer uses,
// so a true result means an IAM connection can be attempted without a login.
func HasADC(ctx context.Context) bool {
	creds, err := google.FindDefaultCredentials(ctx, cloudPlatformScope)
	if err != nil || creds.TokenSource == nil {
		return false
	}
	_, err = creds.TokenSource.Token()
	return err == nil
}

// LoginCmd returns the command that performs an interactive Application Default
// Credentials login. Running it opens a browser (exactly like invoking gcloud
// by hand); it should be run via tea.Exec so the TUI yields the terminal while
// gcloud prints its prompts. Requires the gcloud CLI to be installed.
func LoginCmd() *exec.Cmd {
	return exec.Command("gcloud", "auth", "application-default", "login")
}

// GcloudInstalled reports whether the gcloud CLI is on PATH.
func GcloudInstalled() bool {
	_, err := exec.LookPath("gcloud")
	return err == nil
}

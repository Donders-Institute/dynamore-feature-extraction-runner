package util

import (
	"bytes"
	"fmt"
	"os/exec"
	"syscall"
)

// Payload is the data structure for the feature extraction payload.
type Payload struct {
	// EndPointRadarbase is the endpoint of the radarbase platform.
	EndPointRadarbase string `json:"radarbaseURL"`
	// UserID is the user id the raw data concerns.
	UserID string `json:"userID"`
	// RawDataPath is the filesystem path referring to the raw data
	// of the user.
	RawDataPath string `json:"rawDataPath"`
}

// String is a string representation of the payload.
func (p Payload) String() string {
	return fmt.Sprintf("%s:%s", p.UserID, p.RawDataPath)
}

// Run executes the feature extraction payload on local host, using the
// `runas` user credential.
func (p Payload) Run(runas *syscall.Credential) (string, error) {

	cmd := exec.Command("whoami")

	cmd.SysProcAttr = &syscall.SysProcAttr{
		Credential: runas,
	}

	var out bytes.Buffer
	cmd.Stdout = &out

	err := cmd.Run()
	if err != nil {
		return "", err
	}
	return out.String(), nil
}

// Submit sends the feature extraction payload to be run on the HPC cluster.
// The job is submitted under the credential `runas`.
func (p Payload) Submit(runas *syscall.Credential) (string, error) {

	// compose job script

	// submit job using `Run`, return stdout of it as job id.

	return "", fmt.Errorf("Not implemented")
}

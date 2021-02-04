package util

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path"
	"syscall"

	log "github.com/Donders-Institute/tg-toolset-golang/pkg/logger"
)

const (
	runFeatureStatsWrapper = "/opt/dynamore/run-feature-extract.sh"
	qsubExec               = "/bin/qsub"
)

// Payload is the data structure for the feature extraction payload.
type Payload struct {
	// // EndPointRadarbase is the endpoint of the radarbase platform.
	// EndPointRadarbase string `json:"radarbaseURL"`
	// UserID is the user id the raw data concerns.
	UserID string `json:"userID"`
	// SessionID is the experiment session id the raw data concerns.
	SessionID string `json:"sessionID"`
	// OutputDir is the filesystem path where the output data is to
	// be stored.
	OutputDir string `json:"outputDir"`
}

// String is a string representation of the payload.
func (p Payload) String() string {
	return fmt.Sprintf("%s:%s:%s", p.UserID, p.SessionID, p.OutputDir)
}

// Run executes the feature extraction payload on local host, using the
// `runas` user credential.
func (p Payload) Run(runas *syscall.Credential) (string, error) {

	cmd := exec.Command(runFeatureStatsWrapper, p.UserID, p.SessionID)

	cmd.SysProcAttr = &syscall.SysProcAttr{
		Credential: runas,
	}

	// stdout file
	of, err := os.Create(path.Join(p.OutputDir, fmt.Sprintf("%s.%s.out", p.UserID, p.SessionID)))
	if err != nil {
		return "", err
	}
	defer of.Close()
	cmd.Stdout = of

	// stderr file
	ef, err := os.Create(path.Join(p.OutputDir, fmt.Sprintf("%s.%s.err", p.UserID, p.SessionID)))
	if err != nil {
		return "", err
	}
	defer ef.Close()
	cmd.Stderr = ef

	err = cmd.Run()
	if err != nil {
		return "", err
	}

	// TODO: return something meaningful
	return "success", nil
}

// Submit sends the feature extraction payload to be run on the HPC cluster.
// The job is submitted under the credential `runas`.
func (p Payload) Submit(runas *syscall.Credential) (string, error) {

	// run job submission script
	cmd := exec.Command(qsubExec,
		"-l", "walltime=1:00:00,mem=4gb",
		"-N", fmt.Sprintf("dynamore-%s-%s", p.UserID, p.SessionID),
		"-o", p.OutputDir,
		"-e", p.OutputDir,
		"-F", fmt.Sprintf("%s %s", p.UserID, p.SessionID),
		runFeatureStatsWrapper)

	log.Debugf("command: %s", cmd)

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

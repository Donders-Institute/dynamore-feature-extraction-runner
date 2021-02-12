package util

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"net"
	"os"
	"os/exec"
	"os/user"
	"path"
	"strconv"
	"strings"
	"syscall"
	"time"

	log "github.com/Donders-Institute/tg-toolset-golang/pkg/logger"
	"golang.org/x/crypto/ssh"
)

var (
	featureStatsExec string
	qsubExec         string
	outRoot          string
	trqJobName       string
)

func init() {

	// set executable paths from env. vars.
	featureStatsExec = os.Getenv("FEATURE_STATS_EXEC")
	qsubExec = os.Getenv("QSUB_EXEC")
	trqJobName = os.Getenv("TORQUE_JOB_NAME")
	outRoot = os.Getenv("PAYLOAD_OUTPUT_ROOT")

	// use default if executables are not set.
	if featureStatsExec == "" {
		featureStatsExec = "/opt/dynamore/run-feature-stats.sh"
	}
	if qsubExec == "" {
		qsubExec = "/bin/qsub"
	}
	if trqJobName == "" {
		trqJobName = "dynamore-feature-state"
	}
}

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
	OutputDir string `json:"outputDirectory"`
}

// String is a string representation of the payload.
func (p Payload) String() string {
	return fmt.Sprintf("%s:%s:%s", p.UserID, p.SessionID, p.OutputDir)
}

// Run executes the feature extraction payload on local host, using the
// `runas` user credential.
func (p Payload) Run(runas string) (string, error) {

	// prepare runner credential
	cred, err := GetSyscallCredential(runas)
	if err != nil {
		return "", fmt.Errorf("cannot resolve credential of runner %s: %s", runas, err)
	}

	// get and prepare job's stdout/stderr file paths.
	outdir, err := p.prepareOutputDir(runas)
	if err != nil {
		return "", fmt.Errorf("cannot prepare outdir for payload: %s", err)
	}

	cmd := exec.Command(featureStatsExec, p.UserID, p.SessionID, p.OutputDir)

	cmd.SysProcAttr = &syscall.SysProcAttr{
		Credential: cred,
	}

	// stdout file
	of, err := os.Create(path.Join(outdir, fmt.Sprintf("%s.%s.out", p.UserID, p.SessionID)))
	if err != nil {
		return "", err
	}
	defer of.Close()
	cmd.Stdout = of

	// stderr file
	ef, err := os.Create(path.Join(outdir, fmt.Sprintf("%s.%s.err", p.UserID, p.SessionID)))
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
func (p Payload) Submit(runas, jobReq, jobQueue string) (string, error) {

	// prepare runner credential
	cred, err := GetSyscallCredential(runas)
	if err != nil {
		return "", fmt.Errorf("cannot resolve credential of runner %s: %s", runas, err)
	}

	// get and prepare job's stdout/stderr file paths.
	outdir, err := p.prepareOutputDir(runas)
	if err != nil {
		return "", fmt.Errorf("cannot prepare outdir for payload: %s", err)
	}

	// arguments for `qsubExec`
	args := []string{
		"-l", jobReq,
		"-N", trqJobName,
		"-o", outdir,
		"-e", outdir,
		"-F", fmt.Sprintf("%s %s %s", p.UserID, p.SessionID, p.OutputDir),
	}
	if jobQueue != "" {
		args = append(args, "-q", jobQueue)
	}
	args = append(args, featureStatsExec)

	// run job submission script
	cmd := exec.Command(qsubExec, args...)

	log.Debugf("command: %s", cmd)

	cmd.SysProcAttr = &syscall.SysProcAttr{
		Credential: cred,
	}

	var out bytes.Buffer
	cmd.Stdout = &out

	err = cmd.Run()
	if err != nil {
		return "", err
	}

	return out.String(), nil
}

// SSHSubmit submits jobs via HPC's access node, using SSH. It requires
// pubkey authentication to be established between server and the remote
// user account.
func (p Payload) SSHSubmit(username, jobReq, jobQueue, sshHost, privateKeyFile string) (string, error) {

	// get and prepare job's stdout/stderr file paths.
	outdir, err := p.prepareOutputDir(username)
	if err != nil {
		return "", fmt.Errorf("cannot prepare outdir for payload: %s", err)
	}

	// configure the SSH connection
	privateKey, err := ioutil.ReadFile(privateKeyFile)
	if err != nil {
		return "", err
	}
	signer, _ := ssh.ParsePrivateKey(privateKey)

	clientConfig := &ssh.ClientConfig{
		User: username,
		Auth: []ssh.AuthMethod{ssh.PublicKeys(signer)},
		HostKeyCallback: func(hostname string, remote net.Addr, key ssh.PublicKey) error {
			return nil
		},
		Timeout: time.Duration(10) * time.Second, // timeout of 10 seconds
	}

	// initiate ssh connection
	conn := SSHConnector{}
	client, err := conn.NewClient(sshHost, clientConfig)
	if err != nil {
		return "", err
	}
	defer conn.CloseConnection(client)

	// ssh client session.
	session, err := conn.NewSession(client)
	if err != nil {
		return "", err
	}
	defer conn.CloseSession(session)

	var out bytes.Buffer
	session.Stdout = &out

	// remote ssh command
	var cmd string

	if jobQueue != "" {
		// command with specific job queue
		cmd = fmt.Sprintf(
			`bash -l -c "qsub -q %s -l %s -N %s -o %s -e %s -F '%s %s %s' %s"`,
			jobQueue,
			jobReq,
			trqJobName,
			outdir,
			outdir,
			p.UserID,
			p.SessionID,
			p.OutputDir,
			featureStatsExec,
		)
	} else {
		// command without specific job queue
		cmd = fmt.Sprintf(
			`bash -l -c "qsub -l %s -N %s -o %s -e %s -F '%s %s %s' %s"`,
			jobReq,
			trqJobName,
			outdir,
			outdir,
			p.UserID,
			p.SessionID,
			p.OutputDir,
			featureStatsExec,
		)
	}

	log.Debugf("ssh command: %s", cmd)

	// run ssh command
	if err := session.Run(cmd); err != nil {
		return "", err
	}

	return out.String(), nil
}

// prepareOutputFiles makes sure the existence of the output directory for
// storing the stdout/stderr of this payload execution.  The output directory
// is either within `p.OutputDirRoot` or `${HOME}/dynamore-feature-extraction`
// if `p.OutputDirRoot` is an empty string.
func (p Payload) prepareOutputDir(username string) (string, error) {

	// get user object
	u, err := user.Lookup(username)
	if err != nil {
		return "", err
	}

	// convert uid/gid to integer.
	uid, _ := strconv.Atoi(u.Uid)
	gid, _ := strconv.Atoi(u.Gid)

	// top-level directory
	rdir := outRoot
	if rdir == "" {
		rdir = path.Join(u.HomeDir, "dynamore-feature-extraction")
	}
	if err := os.MkdirAll(rdir, 0750); err != nil {
		return "", err
	}
	if err := os.Chown(rdir, uid, gid); err != nil {
		return "", err
	}

	// sub-directories for outputs of this payload
	for _, p := range strings.Split(p.OutputDir, "/") {
		rdir = path.Join(rdir, p)
		if err := os.MkdirAll(rdir, 0750); err != nil {
			return "", err
		}
		if err := os.Chown(rdir, uid, gid); err != nil {
			return "", err
		}
	}

	return rdir, nil
}

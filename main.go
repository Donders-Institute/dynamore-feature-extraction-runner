package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"math/rand"
	"os"
	"os/user"
	"path"

	"github.com/go-redis/redis/v8"

	"github.com/Donders-Institute/dynamore-feature-extraction-runner/util"

	log "github.com/Donders-Institute/tg-toolset-golang/pkg/logger"
)

var (
	rdb             *redis.Client
	optRedisURL     *string
	optRedisChannel *string
	optRunnerUser   *string
	optSSHKeyDir    *string
	optsJobQue      *string
	optsJobReq      *string
	verbose         *bool

	// list of submission hosts (port number is necessary) of the HPC cluster.
	hpcSubmitHosts = []string{
		"mentat001.dccn.nl:22",
		"mentat002.dccn.nl:22",
		"mentat003.dccn.nl:22",
		"mentat004.dccn.nl:22",
		"mentat005.dccn.nl:22",
	}
)

func usage() {
	fmt.Printf("\nUsage: %s [OPTIONS]\n", os.Args[0])
	fmt.Printf("\nOPTIONS:\n")
	flag.PrintDefaults()
}

func init() {

	u, err := user.Current()
	if err != nil {
		log.Fatalf("%s", err)
	}

	// load env variables for default values
	defaultRedisURL := os.Getenv("REDIS_URL")
	if defaultRedisURL == "" {
		defaultRedisURL = "redis://localhost:6379/0"
	}

	defaultRedisChannel := os.Getenv("REDIS_PAYLOAD_CHANNEL")
	if defaultRedisChannel == "" {
		defaultRedisChannel = "dynamore_feature_extraction"
	}

	defaultExecUser := os.Getenv("EXEC_USER")
	if defaultExecUser == "" {
		defaultExecUser = u.Username
	}

	defaultSSHKeyDir := os.Getenv("SSH_KEY_DIR")
	if defaultSSHKeyDir == "" {
		defaultSSHKeyDir = path.Join(u.HomeDir, ".ssh", "dfe_runner")
	}

	defaultJobReq := os.Getenv("TORQUE_JOB_REQUIREMENT")
	if defaultJobReq == "" {
		defaultJobReq = "walltime=1:00:00,mem=4gb"
	}

	// parse commandline arguments
	optRedisURL = flag.String("d", defaultRedisURL, "set endpoint `url` of the Redis server.")
	optRedisChannel = flag.String("c", defaultRedisChannel, "set redis `channel` for feature-extraction payloads.")
	optRunnerUser = flag.String("u", defaultExecUser, "run feature-extraction process/job as the `user`.")
	optSSHKeyDir = flag.String("k", defaultSSHKeyDir, "`path` in which the the SSH pub/priv keys are created.")
	optsJobReq = flag.String("l", defaultJobReq, "specify the HPC torque job `requirement`.")
	optsJobQue = flag.String("q", os.Getenv("TORQUE_JOB_QUEUE"), "specify the HPC torque job `queue`.")

	verbose = flag.Bool("v", false, "show debug messages.")
	flag.Usage = usage
	flag.Parse()

	// create dir for ssh keys
	if err := os.MkdirAll(*optSSHKeyDir, 0700); err != nil {
		log.Fatalf("%s", err)
	}

	// config logger
	cfg := log.Configuration{
		EnableConsole:     true,
		ConsoleJSONFormat: false,
		ConsoleLevel:      log.Info,
	}

	if *verbose {
		cfg.ConsoleLevel = log.Debug
	}

	// initialize logger
	log.NewLogger(cfg, log.InstanceLogrusLogger)

	// initiate connection to redis server
	opt, err := redis.ParseURL(*optRedisURL)
	if err != nil {
		log.Fatalf("%s", err)
	}

	rdb = redis.NewClient(opt)
}

// main function
func main() {
	ctx := context.Background()
	ctx, cancel := context.WithCancel(ctx)

	chanPayload := rdb.Subscribe(ctx, *optRedisChannel)

	defer func() {
		chanPayload.Close()
		cancel()
	}()

	if err := serve(ctx, chanPayload); err != nil {
		log.Fatalf("%s", err)
	}
}

// serve runs indefinitely and listens to incoming payload message from the redis.
func serve(ctx context.Context, payloads *redis.PubSub) error {

	// provision SSH when sshPrivKey does not exist.
	sshPrivKey := path.Join(*optSSHKeyDir, "id_rsa")
	sshPubKey := path.Join(*optSSHKeyDir, "id_rsa.pub")

	if _, err := os.Stat(sshPrivKey); os.IsNotExist(err) {
		if err := util.GenerateRSAKeyPair(sshPrivKey, sshPubKey); err != nil {
			return fmt.Errorf("cannot initiate RSA keys for ssh connection: %s", err)
		}

		if err := util.AddAuthorizedPublicKey(*optRunnerUser, sshPubKey); err != nil {
			return fmt.Errorf("cannot update authorized_keys: %s", err)
		}
	}

	ch := payloads.Channel()

	for {
		select {
		case <-ctx.Done():
			return nil
		case m := <-ch:
			go runPayload(m, sshPrivKey)
		}
	}

}

func runPayload(m *redis.Message, sshPrivKey string) {
	log.Infof("payload: %s", m.Payload)

	// unmarshal redis message to payload struct
	p := util.Payload{}
	json.Unmarshal([]byte(m.Payload), &p)

	// submit payload in one of the following methods:
	// TODO: make method switch configurable
	//
	// method 1: run singularity on local server.
	// _, err := p.Run(*optRunnerUser)
	//
	// method 2: submit job to run singularity using direct qsub.
	//           The server is the submit host.
	// jid, err := p.Submit(*optRunnerUser, *optJobReq, *optJobQue)
	//
	// method 3: submit job to run singularity via a remote submit host.
	//           SSH keypair auth required.
	submitHost := hpcSubmitHosts[rand.Int()%len(hpcSubmitHosts)]
	jid, err := p.SSHSubmit(*optRunnerUser, *optsJobReq, *optsJobQue, submitHost, sshPrivKey)

	if err != nil {
		log.Errorf("[%s] cannot submit payload: %s", p, err)
	}

	log.Infof("[%s] payload submitted as job %s", p, jid)
}

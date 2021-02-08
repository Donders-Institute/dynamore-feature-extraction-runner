package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
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
	verbose         *bool

	// list of submission hosts of the HPC cluster.
	hpcSubmitHosts = []string{
		"mentat001.dccn.nl",
		"mentat002.dccn.nl",
		"mentat003.dccn.nl",
		"mentat004.dccn.nl",
		"mentat005.dccn.nl",
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
	if defaultRedisChannel == "" {
		defaultExecUser = u.Username
	}

	// parse commandline arguments
	optRedisURL = flag.String("d", defaultRedisURL, "set endpoint `url` of the Redis server.")
	optRedisChannel = flag.String("c", defaultRedisChannel, "set redis `channel` for feature-extraction payloads.")
	optRunnerUser = flag.String("u", defaultExecUser, "run feature-extraction process/job as the `user`.")
	verbose = flag.Bool("v", false, "show debug messages.")
	flag.Usage = usage
	flag.Parse()

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

	// prepare runner credential
	cred, err := util.GetSyscallCredential(*optRunnerUser)
	if err != nil {
		return fmt.Errorf("cannot resolve credential of runner %s: %s", *optRunnerUser, err)
	}

	// provision SSH
	if err := provisionSSH(); err != nil {
		return err
	}

	ch := payloads.Channel()

	for {
		select {
		case <-ctx.Done():
			return nil
		case m := <-ch:
			log.Infof("payload: %s", m.Payload)

			// unmarshal redis message to payload struct
			p := util.Payload{}
			json.Unmarshal([]byte(m.Payload), &p)

			// submit payload
			jid, err := p.Submit(cred)

			if err != nil {
				log.Errorf("[%s] cannot submit payload: %s", p, err)
				continue
			}

			log.Infof("[%s] payload submitted as job %s", p, jid)
		}
	}

}

// provisionSSH prepares the ssh keys in service runner's
// home directory to allow remote job submission.
func provisionSSH() error {
	me, err := user.Current()
	if err != nil {
		return fmt.Errorf("cannot identify service user: %s", err)
	}

	keydir := path.Join(me.HomeDir, ".ssh", "def_runner")
	if err := os.MkdirAll(keydir, 0700); err != nil {
		return err
	}

	sshPrivKey := path.Join(keydir, "id_rsa")
	sshPubKey := path.Join(keydir, "id_rsa.pub")

	if err := util.GenerateRSAKeyPair(sshPrivKey, sshPubKey); err != nil {
		return fmt.Errorf("cannot initiate RSA keys for ssh connection: %s", err)
	}

	if err := util.AddAuthorizedPublicKey(*optRunnerUser, sshPubKey); err != nil {
		return fmt.Errorf("cannot update authorized_keys: %s", err)
	}

	return nil
}

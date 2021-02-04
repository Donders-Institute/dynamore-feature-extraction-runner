package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"os/user"

	"github.com/go-redis/redis/v8"

	"github.com/Donders-Institute/dynamore-feature-extraction-runner/util"

	log "github.com/Donders-Institute/tg-toolset-golang/pkg/logger"
)

const (
	defaultRedisURL     = "redis://localhost:6379/0"
	defaultRedisPass    = ""
	defaultRedisChannel = "dynamore_feature_extraction"
)

var (
	rdb             *redis.Client
	optRedisURL     *string
	optRedisChannel *string
	optRunnerUser   *string
	verbose         *bool
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

	// parse commandline arguments
	optRedisURL = flag.String("d", defaultRedisURL, "set endpoint `url` of the Redis server.")
	optRedisChannel = flag.String("c", defaultRedisChannel, "set redis `channel` for feature-extraction payloads.")
	optRunnerUser = flag.String("u", u.Username, "run feature-extraction process/job as the `user`.")
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

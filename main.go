package main

import (
	"context"
	"flag"
	"fmt"
	"os"

	"github.com/go-redis/redis/v8"

	log "github.com/sirupsen/logrus"
)

const (
	defaultRedisURL  = "redis://localhost:6379/0"
	defaultRedisPass = ""
)

var (
	rdb         *redis.Client
	optRedisURL *string
)

func usage() {
	fmt.Printf("\nUsage: %s [OPTIONS]\n", os.Args[0])
	fmt.Printf("\nOPTIONS:\n")
	flag.PrintDefaults()
}

func init() {

	// parse commandline arguments
	optRedisURL = flag.String("d", defaultRedisURL, "set endpoint `url` of the Redis server.")
	flag.Usage = usage
	flag.Parse()

	// initiate connection to redis server
	opt, err := redis.ParseURL(*optRedisURL)
	if err != nil {
		log.Fatalln(err)
	}

	rdb = redis.NewClient(opt)
}

// main function
func main() {
	ctx := context.Background()
	ctx, cancel := context.WithCancel(ctx)

	chanPayload := rdb.Subscribe(ctx, "payload-feature-extraction")

	defer func() {
		chanPayload.Close()
		cancel()
	}()

	if err := serve(ctx, chanPayload); err != nil {
		log.Fatalf("%s", err)
	}
}

func serve(ctx context.Context, payloads *redis.PubSub) error {

	ch := payloads.Channel()

	for {
		select {
		case <-ctx.Done():
			return nil
		case m := <-ch:
			log.Info("payload: %+v", m)

			p := Payload{}

			jid, err := submitJob(p)

			if err != nil {
				log.Errorf("[%s] cannot submit payload: %s", p, err)
			}

			log.Infof("[%s]payload submitted as job %s", p, jid)
		}
	}

}

// Payload is the data structure for the feature extraction payload.
type Payload struct {
	// EndPointRadarbase is the endpoint of the radarbase platform.
	EndPointRadarbase string
	// UserID is the user id the raw data concerns.
	UserID string
	// RawDataPath is the filesystem path referring to the raw data
	// of the user.
	RawDataPath string
}

// String is a string representation of the payload.
func (p Payload) String() string {
	return p.UserID
}

// submitJob submits a HPC job and returns a job id as a string.
func submitJob(payload Payload) (string, error) {
	return "", fmt.Errorf("Not implemented")
}

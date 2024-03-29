package main

import (
	"context"
	"log"
	"os"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/starton-io/go-dispatch/dcron"
	examplesCommon "github.com/starton-io/go-dispatch/dcron/examples/common"
	"github.com/starton-io/go-dispatch/driver"
	dlog "github.com/starton-io/go-dispatch/logger"
)

type EnvConfig struct {
	RedisAddr     string
	RedisPassword string
	ServerName    string
	JobStoreKey   string
}

func defaultString(targetStr string, defaultStr string) string {
	if targetStr == "" {
		return defaultStr
	}
	return targetStr
}

func (ec *EnvConfig) LoadEnv() {
	ec.RedisAddr = defaultString(os.Getenv("redis_addr"), "127.0.0.1:6379")
	ec.RedisPassword = defaultString(os.Getenv("redis_password"), "123456")
	ec.ServerName = defaultString(os.Getenv("server_name"), "dcronsvr")
	ec.JobStoreKey = defaultString(os.Getenv("job_store_key"), "job_store_key")
}

var IEnv EnvConfig
var localJobStore map[string]string

func addOrUpdateJob(d *dcron.Dcron, logger *log.Logger, jobName, jobData string) {
	// Deserialize jobData to get cronSpec and possibly other job details
	// Assuming jobData is serialized in a way that can be deserialized into an ExecJob instance
	job := &examplesCommon.ExecJob{}
	err := job.UnSerialize([]byte(jobData)) // Ensure this method exists
	if err != nil {
		logger.Printf("Error unserializing job: %v", err)
		return
	}
	err = d.AddJob(jobName, job.GetCron(), job) // Ensure this method exists
	if err != nil {
		logger.Printf("Error adding/updating job: %v", err)
	}
}

func syncJobsWithLocalStore(d *dcron.Dcron, logger *log.Logger, redisOpts *redis.Options) {
	cli := redis.NewClient(redisOpts)
	defer cli.Close()
	redisJobs, err := cli.HGetAll(context.Background(), IEnv.JobStoreKey).Result()
	if err != nil {
		logger.Println("Error fetching jobs from Redis:", err)
		return
	}

	// Initialize localJobStore if it's the first run
	if localJobStore == nil {
		localJobStore = make(map[string]string)
		for jobName, jobData := range redisJobs {
			localJobStore[jobName] = jobData
			addOrUpdateJob(d, logger, jobName, jobData)

		}
		log.Println("localJobStore", localJobStore)
		return
	}

	// For subsequent runs, sync jobs with the local store
	for jobName, jobData := range redisJobs {
		if localData, exists := localJobStore[jobName]; !exists || localData != jobData {
			localJobStore[jobName] = jobData
			addOrUpdateJob(d, logger, jobName, jobData)
		}
	}

	log.Println("localJobStore", localJobStore)
	// Remove jobs from dcron that no longer exist in Redis
	for jobName := range localJobStore {
		if _, exists := redisJobs[jobName]; !exists {
			delete(localJobStore, jobName)
			// d.RemoveJob(jobName) // Ensure this method exists
			d.Remove(jobName)
			logger.Printf("Job removed: %s", jobName)
		}
	}

	log.Println("localJobStore", localJobStore)
}

func main() {
	logger := log.New(os.Stdout, "", log.LstdFlags)
	IEnv.LoadEnv()

	redisOpts := &redis.Options{
		Addr:     IEnv.RedisAddr,
		Password: IEnv.RedisPassword,
	}

	redisCli := redis.NewClient(redisOpts)
	drv := driver.NewRedisDriver(redisCli)
	dcronInstance := dcron.NewDcronWithOption(IEnv.ServerName, drv,
		dcron.WithLogger(dlog.DefaultLogger()),
		dcron.CronOptionSeconds(),
		dcron.WithHashReplicas(20),
		dcron.WithNodeUpdateDuration(10*time.Second),
		//dcron.WithClusterStable(5*time.Second),
		dcron.WithRecoverFunc(func(d *dcron.Dcron) {
			go func() {
				// Initial sync
				syncJobsWithLocalStore(d, logger, redisOpts)

				// Periodic sync every 5 seconds
				ticker := time.NewTicker(2 * time.Minute)
				defer ticker.Stop()
				for range ticker.C {
					syncJobsWithLocalStore(d, logger, redisOpts)
				}
			}()
		}),
	)
	dcronInstance.Start()
	defer dcronInstance.Stop()
	// run forever
	tick := time.NewTicker(time.Hour)
	for range tick.C {
	}
}

package main

import (
	"context"
	"flag"

	"github.com/redis/go-redis/v9"
	examplesCommon "github.com/starton-io/go-dispatch/dcron/examples/common"
)

var (
	addr = flag.String("addr", "127.0.0.1:6379", "redis addr")
	pwd  = flag.String("pwd", "123456", "redis password")
	key  = flag.String("key", "key", "stable job key")
)

func Store(cli *redis.Client, jobName string, job *examplesCommon.ExecJob) {
	b, _ := job.Serialize()
	_ = cli.HSet(context.Background(), *key, jobName, b)
}

func main() {
	flag.Parse()
	opts := &redis.Options{
		Addr:     *addr,
		Password: *pwd,
	}
	cli := redis.NewClient(opts)

	Store(cli, "job1", &examplesCommon.ExecJob{
		Cron:    "@every 10s",
		Command: "date",
	})
	Store(cli, "job2", &examplesCommon.ExecJob{
		Cron:    "@every 10s",
		Command: "df -h",
	})
	Store(cli, "job3", &examplesCommon.ExecJob{
		Cron:    "@every 10s",
		Command: "echo job3",
	})
	Store(cli, "job4", &examplesCommon.ExecJob{
		Cron:    "@every 20s",
		Command: "echo job4",
	})
	Store(cli, "job5", &examplesCommon.ExecJob{
		Cron:    "@every 10s",
		Command: "echo job5",
	})
	Store(cli, "job6", &examplesCommon.ExecJob{
		Cron:    "@every 20s",
		Command: "echo job6",
	})
	Store(cli, "job7", &examplesCommon.ExecJob{
		Cron:    "@every 20s",
		Command: "echo job7",
	})
	Store(cli, "job8", &examplesCommon.ExecJob{
		Cron:    "@every 20s",
		Command: "echo job8",
	})
	Store(cli, "job9", &examplesCommon.ExecJob{
		Cron:    "@every 10s",
		Command: "echo job9",
	})
	Store(cli, "job10", &examplesCommon.ExecJob{
		Cron:    "@every 10s",
		Command: "echo job10",
	})
	Store(cli, "job11", &examplesCommon.ExecJob{
		Cron:    "@every 10s",
		Command: "echo job11",
	})
	Store(cli, "job12", &examplesCommon.ExecJob{
		Cron:    "@every 20s",
		Command: "echo job12",
	})
	Store(cli, "job13", &examplesCommon.ExecJob{
		Cron:    "@every 10s",
		Command: "echo job13",
	})
	Store(cli, "job14", &examplesCommon.ExecJob{
		Cron:    "@every 20s",
		Command: "echo job14",
	})
}

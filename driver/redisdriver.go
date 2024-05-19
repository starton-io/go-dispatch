package driver

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	redis "github.com/redis/go-redis/v9"
	dlog "github.com/starton-io/go-dispatch/logger"
)

const (
	redisDefaultTimeout = 5 * time.Second
)

type RedisDriver struct {
	Client          *redis.Client
	GlobalKeyPrefix string

	serviceName string
	nodeID      string
	timeout     time.Duration
	logger      dlog.Logger
	started     bool

	// this context is used to define
	// the lifetime of this driver.
	runtimeCtx    context.Context
	runtimeCancel context.CancelFunc

	sync.Mutex
}

func newRedisDriver(redisClient *redis.Client) *RedisDriver {
	rd := &RedisDriver{
		Client:          redisClient,
		GlobalKeyPrefix: DefaultGlobalKeyPrefix,
		logger:          dlog.DefaultLogger(),
		timeout:         redisDefaultTimeout,
	}
	rd.started = false
	return rd
}

func (rd *RedisDriver) Init(serviceName string, opts ...Option) {
	rd.serviceName = serviceName
	for _, opt := range opts {
		rd.withOption(opt)
	}
	rd.nodeID = GetNodeId(rd.GlobalKeyPrefix, rd.serviceName)
}

func (rd *RedisDriver) NodeID() string {
	return rd.nodeID
}

func (rd *RedisDriver) Start(ctx context.Context) (err error) {
	rd.Lock()
	defer rd.Unlock()
	if rd.started {
		err = errors.New("this driver is started")
		return
	}
	rd.runtimeCtx, rd.runtimeCancel = context.WithCancel(context.TODO())
	rd.started = true
	// register
	err = rd.registerServiceNode()
	if err != nil {
		rd.logger.Errorf("register service error=%v", err)
		return
	}
	// heartbeat timer
	go rd.heartBeat()
	return
}

func (rd *RedisDriver) Stop(ctx context.Context) (err error) {
	rd.Lock()
	defer rd.Unlock()
	rd.runtimeCancel()
	rd.started = false
	return
}

func (rd *RedisDriver) GetNodes(ctx context.Context) (nodes []string, err error) {
	mathStr := fmt.Sprintf("%s*", GetKeyPre(rd.GlobalKeyPrefix, rd.serviceName))
	return rd.scan(ctx, mathStr)
}

// private function
func (rd *RedisDriver) heartBeat() {
	tick := time.NewTicker(rd.timeout / 2)
	for {
		select {
		case <-tick.C:
			{
				if err := rd.registerServiceNode(); err != nil {
					rd.logger.Errorf("register service node error %+v", err)
				}
			}
		case <-rd.runtimeCtx.Done():
			{
				if err := rd.Client.Del(context.Background(), rd.nodeID, rd.nodeID).Err(); err != nil {
					rd.logger.Errorf("unregister service node error %+v", err)
				}
				return
			}
		}
	}
}

func (rd *RedisDriver) registerServiceNode() error {
	exists, err := rd.Client.Exists(context.Background(), rd.nodeID).Result()
	if err != nil {
		return err
	}
	// If the key does not exist, use GetNodeId to generate a new nodeID
	if rd.nodeID == "" || exists == 0 {
		rd.nodeID = GetNodeId(rd.GlobalKeyPrefix, rd.serviceName)
	}
	return rd.Client.SetEx(context.Background(), rd.nodeID, rd.nodeID, rd.timeout).Err()
}

func (rd *RedisDriver) scan(ctx context.Context, matchStr string) ([]string, error) {
	ret := make([]string, 0)
	iter := rd.Client.Scan(ctx, 0, matchStr, -1).Iterator()
	for iter.Next(ctx) {
		err := iter.Err()
		if err != nil {
			return nil, err
		}
		ret = append(ret, iter.Val())
	}
	return ret, nil
}

func (rd *RedisDriver) withOption(opt Option) (err error) {
	switch opt.Type() {
	case OptionTypeTimeout:
		{
			rd.timeout = opt.(TimeoutOption).timeout
		}
	case OptionTypeLogger:
		{
			rd.logger = opt.(LoggerOption).logger

		}
	case OptionTypeGlobalPrefix:
		{
			rd.GlobalKeyPrefix = opt.(GlobalPrefixOption).globalPrefix
		}
	}
	return
}

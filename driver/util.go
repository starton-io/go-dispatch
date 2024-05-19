package driver

import (
	"time"

	"github.com/google/uuid"
)

// GlobalKeyPrefix is a global redis key prefix
const DefaultGlobalKeyPrefix = "distributed-nodepool:"

func GetKeyPre(globalKeyPrefix, serviceName string) string {
	return globalKeyPrefix + serviceName + ":"
}

func GetNodeId(globalKeyPrefix, serviceName string) string {
	return GetKeyPre(globalKeyPrefix, serviceName) + uuid.New().String()
}

func GetStableJobStore(globalKeyPrefix, serviceName string) string {
	return GetKeyPre(globalKeyPrefix, serviceName) + "stable-jobs"
}

func GetStableJobStoreTxKey(globalKeyPrefix, serviceName string) string {
	return GetKeyPre(globalKeyPrefix, serviceName) + "TX:stable-jobs"
}

func TimePre(t time.Time, preDuration time.Duration) int64 {
	return t.Add(-preDuration).Unix()
}

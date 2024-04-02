package nodepool

import (
	"context"
	"errors"
	"time"
)

var (
	ErrNodePoolIsUpgrading = errors.New("nodePool is upgrading")
)

type INodePool interface {
	Start(ctx context.Context) error
	Stop(ctx context.Context) error

	IsEligible(key string) (bool, error)
	DecreaseLoadByKey(key string) error
	ResetLoads()

	GetHost(key string) (string, error)
	GetNodeID() string
	GetLastNodesUpdateTime() time.Time
	GetState() string
}

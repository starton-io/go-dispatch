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
	CheckJobAvailable(key string) (bool, error)
	Stop(ctx context.Context) error

	GetNodeID() string
	AddKey(key string, targetNode string)
	RemoveKey(key string)
	GetLastNodesUpdateTime() time.Time
}

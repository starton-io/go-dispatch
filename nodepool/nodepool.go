package nodepool

import (
	"context"
	"reflect"
	"sort"
	"sync"
	"sync/atomic"
	"time"

	"github.com/starton-io/go-dispatch/consistenthash"
	"github.com/starton-io/go-dispatch/driver"
	dlog "github.com/starton-io/go-dispatch/logger"
)

const (
	NodePoolStateSteady  = "NodePoolStateSteady"
	NodePoolStateUpgrade = "NodePoolStateUpgrade"
)

// NodePool
// For cluster steable.
// NodePool has 2 states:
//  1. Steady
//     If this nodePoolLists is the same as the last update,
//     we will mark this node's state to Steady. In this state,
//     this node can run jobs.
//  2. Upgrade
//     If this nodePoolLists is different to the last update,
//     we will mark this node's state to Upgrade. In this state,
//     this node can not run jobs.
type NodePool struct {
	serviceName string
	nodeID      string

	rwMut sync.RWMutex
	Nodes *consistenthash.Consistent

	Driver         driver.DriverV2
	hashReplicas   int
	hashLoad       float64
	updateDuration time.Duration

	logger   dlog.Logger
	stopChan chan int
	preNodes []string // sorted

	capacityLoad bool

	lastUpdateNodesTime atomic.Value
	state               atomic.Value
}

type Option func(*NodePool)

func WithCapacityLoad() Option {
	return func(np *NodePool) {
		np.capacityLoad = true
	}
}

func NewNodePool(
	serviceName string,
	drv driver.DriverV2,
	updateDuration time.Duration,
	hashReplicas int,
	logger dlog.Logger,
	options ...Option,
) INodePool {
	np := &NodePool{
		serviceName:    serviceName,
		Driver:         drv,
		hashReplicas:   hashReplicas,
		updateDuration: updateDuration,
		hashLoad:       consistenthash.DefaultLoad,
		logger:         dlog.DefaultLogger(),
		stopChan:       make(chan int, 1),
	}
	if logger != nil {
		np.logger = logger
	}
	np.Driver.Init(serviceName,
		driver.NewTimeoutOption(updateDuration),
		driver.NewLoggerOption(np.logger))
	for _, opt := range options {
		opt(np)
	}
	return np
}

func WithHashLoad(load float64) Option {
	return func(np *NodePool) {
		np.hashLoad = load
	}
}

func (np *NodePool) Start(ctx context.Context) (err error) {
	err = np.Driver.Start(ctx)
	if err != nil {
		np.logger.Errorf("start pool error: %v", err)
		return
	}
	np.nodeID = np.Driver.NodeID()
	nowNodes, err := np.Driver.GetNodes(ctx)
	if err != nil {
		np.logger.Errorf("get nodes error: %v", err)
		return
	}
	np.state.Store(NodePoolStateUpgrade)
	np.updateHashRing(nowNodes)
	go np.waitingForHashRing()

	// stuck util the cluster state came to steady.
	for np.GetState() != NodePoolStateSteady {
		<-time.After(np.updateDuration)
	}
	np.logger.Infof("nodepool started for serve, nodeID=%s", np.nodeID)

	return
}

func (np *NodePool) CheckJobAvailable(jobName string) (bool, error) {
	np.rwMut.RLock()
	defer np.rwMut.RUnlock()

	if np.Nodes == nil {
		np.logger.Errorf("nodeID=%s, NodePool.nodes is nil", np.nodeID)
		return false, nil // Consider returning an error indicating that the nodes are nil
	}
	if np.Nodes.IsEmpty() {
		return false, nil
	}
	if np.state.Load().(string) != NodePoolStateSteady {
		return false, ErrNodePoolIsUpgrading
	}
	var targetNode string
	var err error
	if np.capacityLoad {
		targetNode, err = np.Nodes.GetLeast(jobName)
		if err != nil {
			np.logger.Errorf("get least node error: %v", err)
			return false, err
		}
		np.Nodes.Inc(targetNode)
	} else {
		targetNode, err = np.Nodes.Get(jobName)
		if err != nil {
			np.logger.Errorf("get node error: %v", err)
			return false, err
		}
	}
	return np.nodeID == targetNode, nil
}

func (np *NodePool) Stop(ctx context.Context) error {
	np.stopChan <- 1
	np.Driver.Stop(ctx)
	np.preNodes = make([]string, 0)
	return nil
}

func (np *NodePool) GetNodeID() string {
	return np.nodeID
}

func (np *NodePool) GetLastNodesUpdateTime() time.Time {
	return np.lastUpdateNodesTime.Load().(time.Time)
}

func (np *NodePool) GetState() string {
	return np.state.Load().(string)
}

func (np *NodePool) waitingForHashRing() {
	tick := time.NewTicker(np.updateDuration)
	for {
		select {
		case <-tick.C:
			nowNodes, err := np.Driver.GetNodes(context.Background())
			if err != nil {
				np.logger.Errorf("get nodes error %v", err)
				continue
			}
			np.updateHashRing(nowNodes)
		case <-np.stopChan:
			return
		}
	}
}

// Reset load of host to 0
func (np *NodePool) ResetLoads() {
	np.rwMut.RLock()
	defer np.rwMut.RUnlock()
	np.Nodes.ResetLoads()
}

// Remove deprecated function
//func (np *NodePool) initHashRing(nodes []string) {
//	np.rwMut.Lock()
//	defer np.rwMut.Unlock()
//	if np.equalRing(nodes) {
//		np.state.Store(NodePoolStateSteady)
//		np.logger.Infof("nowNodes=%v, preNodes=%v", nodes, np.preNodes)
//		return
//	}
//	np.lastUpdateNodesTime.Store(time.Now())
//	np.state.Store(NodePoolStateUpgrade)
//	np.logger.Infof("update hashRing nodes=%+v", nodes)
//	np.preNodes = make([]string, len(nodes))
//	copy(np.preNodes, nodes)
//	np.nodes = consistenthash.New(consistenthash.WithReplicas(np.hashReplicas))
//	for _, v := range nodes {
//		np.nodes.Add(v)
//	}
//}

// Remove deprecated function
//func (np *NodePool) equalRing(a []string) bool {
//	if len(a) == len(np.preNodes) {
//		la := len(a)
//		sort.Strings(a)
//		for i := 0; i < la; i++ {
//			if a[i] != np.preNodes[i] {
//				return false
//			}
//		}
//		return true
//	}
//	return false
//}

// updateHashRing updates the hash ring with the given list of nodes.
// This function is optimized the update of the hash ring by comparing the incoming list of nodes with the previous list.
// It only adds or removes nodes from the hash ring if there's a change, and it also logs the nodes that were added or removed.
func (np *NodePool) updateHashRing(nodes []string) {
	np.rwMut.Lock()
	defer np.rwMut.Unlock()

	// Ensure the hash ring is initialized
	if np.Nodes == nil {
		np.Nodes = consistenthash.New(consistenthash.WithReplicas(np.hashReplicas), consistenthash.WithDefaultLoad(np.hashLoad))
	}

	// Sort the incoming list of nodes to ensure consistency in comparison
	sort.Strings(nodes)

	// Check if there's any change in the nodes list
	if reflect.DeepEqual(np.preNodes, nodes) {
		// No change in the nodes list, no need to update the hash ring
		np.logger.Infof("nowNodes=%v, preNodes=%v", nodes, np.preNodes)
		np.state.Store(NodePoolStateSteady)
		return
	}
	// Update the state based on whether the node pool is in steady or upgrade state
	np.state.Store(NodePoolStateUpgrade)

	// Store the time of the last update
	np.lastUpdateNodesTime.Store(time.Now())

	// Identify nodes added or removed by comparing np.preNodes and nodes
	addedNodes, removedNodes := diffNodes(np.preNodes, nodes)

	// Remove nodes from the hash ring
	for _, node := range removedNodes {
		np.Nodes.Remove(node)
	}

	// Add new nodes to the hash ring
	for _, node := range addedNodes {
		np.Nodes.Add(node)
	}

	np.logger.Infof("updateHashRing: addedNodes=%v, removedNodes=%v", addedNodes, removedNodes)

	// Update the preNodes list to reflect the current state
	np.preNodes = make([]string, len(nodes))
	copy(np.preNodes, nodes)

}

// diffNodes compares two slices of node identifiers and returns the nodes that have been added or removed.
func diffNodes(oldNodes, newNodes []string) (added, removed []string) {
	nodeCount := make(map[string]int)

	// Increment count for new nodes
	for _, node := range newNodes {
		nodeCount[node]++
	}

	// Decrement count for old nodes
	for _, node := range oldNodes {
		nodeCount[node]--
	}

	// Nodes with count > 0 are added, count < 0 are removed
	for node, count := range nodeCount {
		if count > 0 {
			added = append(added, node)
		} else if count < 0 {
			removed = append(removed, node)
		}
	}

	return added, removed
}

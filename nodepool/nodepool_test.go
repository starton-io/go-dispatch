package nodepool_test

import (
	"context"
	"strconv"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
	"github.com/starton-io/go-dispatch/consistenthash"
	"github.com/starton-io/go-dispatch/driver"
	"github.com/starton-io/go-dispatch/nodepool"
	"github.com/stretchr/testify/suite"
)

type TestINodePoolSuite struct {
	suite.Suite

	rds                 *miniredis.Miniredis
	defaultHashReplicas int
}

func (ts *TestINodePoolSuite) SetupTest() {
	ts.defaultHashReplicas = 50
}

func (ts *TestINodePoolSuite) TearDownTest() {
	if ts.rds != nil {
		ts.rds.Close()
		ts.rds = nil
	}
}

func (ts *TestINodePoolSuite) setUpRedis() {
	ts.rds = miniredis.RunT(ts.T())
}

func (ts *TestINodePoolSuite) stopAllNodePools(nodePools []nodepool.INodePool) {
	for _, nodePool := range nodePools {
		nodePool.Stop(context.Background())
	}
}

func (ts *TestINodePoolSuite) declareRedisDrivers(clients *[]*redis.Client, drivers *[]driver.DriverV2, numberOfNodes int) {
	for i := 0; i < numberOfNodes; i++ {
		*clients = append(*clients, redis.NewClient(&redis.Options{
			Addr: ts.rds.Addr(),
		}))
		*drivers = append(*drivers, driver.NewRedisDriver((*clients)[i]))
	}
}

func (ts *TestINodePoolSuite) declareRedisZSetDrivers(clients *[]*redis.Client, drivers *[]driver.DriverV2, numberOfNodes int) {
	for i := 0; i < numberOfNodes; i++ {
		*clients = append(*clients, redis.NewClient(&redis.Options{
			Addr: ts.rds.Addr(),
		}))
		*drivers = append(*drivers, driver.NewRedisZSetDriver((*clients)[i]))
	}
}

func (ts *TestINodePoolSuite) runCheckJobAvailable(numberOfNodes int, ServiceName string, nodePools *[]nodepool.INodePool, updateDuration time.Duration) {
	for i := 0; i < numberOfNodes; i++ {
		err := (*nodePools)[i].Start(context.Background())
		ts.Require().Nil(err)
	}
	<-time.After(updateDuration * 2)
	ring := consistenthash.New(consistenthash.WithReplicas(ts.defaultHashReplicas))
	for _, v := range *nodePools {
		ring.Add(v.GetNodeID())
	}

	for i := 0; i < 10000; i++ {
		for j := 0; j < numberOfNodes; j++ {
			nodeID, _ := ring.Get(strconv.Itoa(i))
			ts.T().Logf("nodeID=%s, jobID=%d, nodePool=%d", nodeID, i, j)
			ok, err := (*nodePools)[j].CheckJobAvailable(strconv.Itoa(i))
			ts.Require().Nil(err)
			ts.Require().Equal(
				ok,
				nodeID == (*nodePools)[j].GetNodeID(),
			)
		}
	}
}

func (ts *TestINodePoolSuite) TestMultiNodesRedis() {
	var clients []*redis.Client
	var drivers []driver.DriverV2
	var nodePools []nodepool.INodePool

	numberOfNodes := 5
	ServiceName := "TestMultiNodesRedis"
	updateDuration := 2 * time.Second
	ts.setUpRedis()
	ts.declareRedisDrivers(&clients, &drivers, numberOfNodes)

	for i := 0; i < numberOfNodes; i++ {
		nodePools = append(nodePools, nodepool.NewNodePool(ServiceName, drivers[i], updateDuration, ts.defaultHashReplicas, nil))
	}
	ts.runCheckJobAvailable(numberOfNodes, ServiceName, &nodePools, updateDuration)
	ts.stopAllNodePools(nodePools)
}

func (ts *TestINodePoolSuite) TestMultiNodesRedisZSet() {
	var clients []*redis.Client
	var drivers []driver.DriverV2
	var nodePools []nodepool.INodePool

	numberOfNodes := 5
	ServiceName := "TestMultiNodesZSet"
	updateDuration := 2 * time.Second

	ts.setUpRedis()
	ts.declareRedisZSetDrivers(&clients, &drivers, numberOfNodes)

	for i := 0; i < numberOfNodes; i++ {
		nodePools = append(nodePools, nodepool.NewNodePool(ServiceName, drivers[i], updateDuration, ts.defaultHashReplicas, nil))
	}
	ts.runCheckJobAvailable(numberOfNodes, ServiceName, &nodePools, updateDuration)
	ts.stopAllNodePools(nodePools)
}

func TestTestINodePoolSuite(t *testing.T) {
	s := new(TestINodePoolSuite)
	suite.Run(t, s)
}

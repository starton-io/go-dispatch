package consistenthash

import (
	"errors"
	"fmt"
	"math"
	"sort"
	"sync"
	"sync/atomic"

	"github.com/cespare/xxhash/v2"
)

const (
	replicationFactor = 50
	DefaultLoad       = 1.25
)

var ErrNoHosts = errors.New("no hosts added")

type Host struct {
	Name string
	Load int64
}

type Opt func(*Consistent)

type Consistent struct {
	hosts       map[uint64]string
	sortedSet   []uint64
	loadMap     map[string]*Host
	totalLoad   int64
	replicas    int
	defaultLoad float64

	sync.RWMutex
}

func New(opts ...Opt) *Consistent {
	c := &Consistent{
		hosts:     map[uint64]string{},
		sortedSet: []uint64{},
		loadMap:   map[string]*Host{},
		replicas:  replicationFactor,
	}
	for _, opt := range opts {
		opt(c)
	}
	return c
}

func WithReplicas(number int) Opt {
	return func(c *Consistent) {
		c.replicas = number
	}
}

func WithDefaultLoad(load float64) Opt {
	return func(c *Consistent) {
		// If the load is less than 1.00, set it to the 1.00
		if load < 1.00 {
			c.defaultLoad = 1.00
		}
		c.defaultLoad = load
	}
}

func (c *Consistent) Add(host string) {
	c.Lock()
	defer c.Unlock()

	if _, ok := c.loadMap[host]; ok {
		return
	}

	c.loadMap[host] = &Host{Name: host, Load: 0}
	for i := 0; i < c.replicas; i++ {
		h := c.hash(fmt.Sprintf("%s%d", host, i))
		c.hosts[uint64(h)] = host
		c.sortedSet = append(c.sortedSet, uint64(h))

	}
	// sort hashes ascendingly
	sort.Slice(c.sortedSet, func(i int, j int) bool {
		return c.sortedSet[i] < c.sortedSet[j]
	})
}

func (c *Consistent) GetSortedSet() []uint64 {
	copySortedSet := make([]uint64, len(c.sortedSet))
	copy(copySortedSet, c.sortedSet)
	return copySortedSet
}

func (c *Consistent) IsEmpty() bool {
	c.RLock()
	defer c.RUnlock()
	return len(c.hosts) == 0
}

// Reset loads of all hosts to 0
func (c *Consistent) ResetLoads() {
	c.Lock()
	defer c.Unlock()
	for _, host := range c.loadMap {
		host.Load = 0
	}
	c.totalLoad = 0
}

// Returns the host that owns `key`.
//
// As described in https://en.wikipedia.org/wiki/Consistent_hashing
//
// It returns ErrNoHosts if the ring has no hosts in it.
func (c *Consistent) Get(key string) (string, error) {
	c.RLock()
	defer c.RUnlock()

	if len(c.hosts) == 0 {
		return "", ErrNoHosts
	}

	h := c.hash(key)
	idx := c.search(h)
	return c.hosts[c.sortedSet[idx]], nil
}

// It uses Consistent Hashing With Bounded loads
//
// https://research.googleblog.com/2017/04/consistent-hashing-with-bounded-loads.html
//
// to pick the least loaded host that can serve the key
//
// It returns ErrNoHosts if the ring has no hosts in it.
//func (c *Consistent) GetLeastDeprecated(key string) (string, error) {
//	c.RLock()
//	defer c.RUnlock()
//
//	if len(c.hosts) == 0 {
//		return "", ErrNoHosts
//	}
//
//	h := c.hash(key)
//	idx := c.search(h)
//
//	i := idx
//	for {
//		host, ok := c.hosts[c.sortedSet[i]]
//		if !ok {
//			return "", ErrNoHosts
//		}
//		if c.loadOK(host) {
//			return host, nil
//		}
//		i++
//		if i >= len(c.hosts) {
//			i = 0
//		}
//	}
//}

// GetLeastv2 is an alternative implementation of GetLeast that is more efficient
func (c *Consistent) GetLeast(key string) (string, error) {
	c.RLock()
	defer c.RUnlock()

	if len(c.hosts) == 0 {
		return "", ErrNoHosts
	}

	h := c.hash(key)
	idx := c.search(h)

	leastLoadedHost := ""
	leastLoad := int64(math.MaxInt64)
	consideredHosts := make(map[string]struct{}) // Map to track considered hosts

	// Initially assume no node can accept more load; track the least loaded node
	for {
		// Wrap around the ring if necessary
		if idx >= len(c.sortedSet) {
			idx = 0
		}
		// If we've considered all unique hosts, stop the search
		if len(consideredHosts) == len(c.loadMap) {
			break
		}

		host, ok := c.hosts[c.sortedSet[idx]]
		if !ok {
			return "", ErrNoHosts // This should not happen, but just in case
		}

		// Skip if we've already considered this host
		if _, alreadyConsidered := consideredHosts[host]; alreadyConsidered {
			idx++
			continue
		}
		consideredHosts[host] = struct{}{}

		// Check if the current host can accept more load before updating least loaded host
		if c.loadOK(host) {
			return host, nil
		}

		hostInfo, ok := c.loadMap[host]
		if !ok {
			hostInfo = &Host{Name: host, Load: 0}
		}
		hostLoad := hostInfo.Load

		// Update least loaded host if current host has lower load
		if hostLoad < leastLoad {
			leastLoadedHost = host
			leastLoad = hostLoad
		}

		idx++
	}

	// Return the least loaded host if all nodes are over the threshold
	return leastLoadedHost, nil
}

func (c *Consistent) search(key uint64) int {
	idx := sort.Search(len(c.sortedSet), func(i int) bool {
		return c.sortedSet[i] >= key
	})

	if idx >= len(c.sortedSet) {
		idx = 0
	}
	return idx
}

// Sets the load of `host` to the given `load`
func (c *Consistent) UpdateLoad(host string, load int64) {
	c.Lock()
	defer c.Unlock()

	if _, ok := c.loadMap[host]; !ok {
		return
	}
	c.totalLoad -= c.loadMap[host].Load
	c.loadMap[host].Load = load
	c.totalLoad += load
}

// Increments the load of host by 1
//
// should only be used with if you obtained a host with GetLeast
func (c *Consistent) Inc(host string) {
	c.Lock()
	defer c.Unlock()

	if _, ok := c.loadMap[host]; !ok {
		return
	}
	atomic.AddInt64(&c.loadMap[host].Load, 1)
	atomic.AddInt64(&c.totalLoad, 1)
}

// Decrements the load of host by 1
//
// should only be used with if you obtained a host with GetLeast
func (c *Consistent) Done(host string) {
	c.Lock()
	defer c.Unlock()

	if _, ok := c.loadMap[host]; !ok {
		return
	}
	atomic.AddInt64(&c.loadMap[host].Load, -1)
	atomic.AddInt64(&c.totalLoad, -1)
}

// Deletes host from the ring
func (c *Consistent) Remove(host string) bool {
	c.Lock()
	defer c.Unlock()

	for i := 0; i < replicationFactor; i++ {
		h := c.hash(fmt.Sprintf("%s%d", host, i))

		delete(c.hosts, h)
		c.delSlice(h)
	}
	delete(c.loadMap, host)
	return true
}

// Return the list of hosts in the ring
func (c *Consistent) Hosts() (hosts []string) {
	c.RLock()
	defer c.RUnlock()
	for k := range c.loadMap {
		hosts = append(hosts, k)
	}
	return hosts
}

// Returns the loads of all the hosts
func (c *Consistent) GetLoads() map[string]int64 {
	loads := map[string]int64{}

	for k, v := range c.loadMap {
		loads[k] = v.Load
	}
	return loads
}

// Returns the maximum load of the single host
// which is:
// (total_load/number_of_hosts)*1.25
// total_load = is the total number of active requests served by hosts
// for more info:
// https://research.googleblog.com/2017/04/consistent-hashing-with-bounded-loads.html
func (c *Consistent) MaxLoad() int64 {
	if c.totalLoad == 0 {
		c.totalLoad = 1
	}
	var avgLoadPerNode float64
	avgLoadPerNode = float64(c.totalLoad / int64(len(c.loadMap)))
	if avgLoadPerNode == 0 {
		avgLoadPerNode = 1
	}
	avgLoadPerNode = math.Ceil(avgLoadPerNode * 1.25)
	return int64(avgLoadPerNode)
}

func (c *Consistent) loadOK(host string) bool {
	// a safety check if someone performed c.Done more than needed
	if c.totalLoad < 0 {
		c.totalLoad = 0
	}

	var avgLoadPerNode float64
	avgLoadPerNode = float64((c.totalLoad + 1) / int64(len(c.loadMap)))
	if avgLoadPerNode == 0 {
		avgLoadPerNode = 1
	}
	avgLoadPerNode = math.Ceil(avgLoadPerNode * 1.25)

	bhost, ok := c.loadMap[host]
	if !ok {
		//panic(fmt.Sprintf("given host(%s) not in loadsMap", bhost.Name))
		return false
	}

	if float64(bhost.Load)+1 <= avgLoadPerNode {
		return true
	}

	return false
}

func (c *Consistent) delSlice(val uint64) {
	idx := -1
	l := 0
	r := len(c.sortedSet) - 1
	for l <= r {
		m := (l + r) / 2
		if c.sortedSet[m] == val {
			idx = m
			break
		} else if c.sortedSet[m] < val {
			l = m + 1
		} else if c.sortedSet[m] > val {
			r = m - 1
		}
	}
	if idx != -1 {
		c.sortedSet = append(c.sortedSet[:idx], c.sortedSet[idx+1:]...)
	}
}

func (c *Consistent) hash(key string) uint64 {
	out := xxhash.Sum64([]byte(key))
	return out
}

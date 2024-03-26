# Consistent Hashing Package

## Description

The `consistenthash` package implements Consistent Hashing, a key technique in distributed systems for distributing data across a cluster in a way that minimizes reorganization when nodes are added or removed. This is particularly useful for load balancing and caching strategies.

## Installation

To use the `consistenthash` package in your Go project, install it using the `go get` command:

```bash
go get github.com/starton-io/go-dispatch/consistenthash
```

## Usage


Below is as simple example demonstrating how to use the `consistenthash` package to distribute data across a cluster of nodes:

```go
import (
    "fmt"
    "github.com/starton-io/go-dispatch/consistenthash"
)

func main() {
    // Initialize the consistent hash with a specified number of replicas and an optional hash function.
    ch := consistenthash.New(consistenthash.WithReplicas(50)) // Here, 50 is the number of replicas.

    // Add nodes to the hash ring.
    ch.Add("Node1")
    ch.Add("Node2")
    ch.Add("Node3")

    // Retrieve the node responsible for a given key.
    node := ch.Get("myKey")
    fmt.Println("Node for key 'myKey':", node)

    // Retrieve the node responsible for a given key by using the least loaded node
    node, _ := ch.GetLeast("myKey")
    fmt.Println("Node for key 'myKey' by using the least loaded node:", node)

	// Use inc to increase the load of a node
	ch.Inc(node)

    // Example of removing a node from the hash ring.
    ch.Remove("Node2")

    // Retrieve the node for the same key after removing a node.
    nodeAfterRemoval := ch.Get("myKey")
    fmt.Println("Node for key 'myKey' after removal:", nodeAfterRemoval)
}
```

This example covers initializing the hash ring, adding nodes, finding the node for a specific key, and observing the effect of removing a node from the hash ring.

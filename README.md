![Starton Banner](https://github.com/starton-io/.github/blob/master/github-banner.jpg?raw=true)

# Go Dispatch

## Description

The `dispatch` package is designed to manage a pool of nodes within a distributed system, particularly focusing on maintaining the state and availability of these nodes for job execution. It interacts with Redis to register nodes, keep heartbeats, and retrieve the list of active nodes. The package ensures that jobs are only executed on nodes that are in a steady state, avoiding nodes that are upgrading or otherwise unavailable.

![Overview](/resources/dispatch-system.png)

## Key Components

- [**NodePool**](nodepool/): Central component that manages the lifecycle and state of nodes within the distributed system. It ensures that jobs are executed on nodes that are in a steady state and not undergoing upgrades or maintenance.

- [**RedisDriver**](driver/): Responsible for the interaction with Redis. It registers service nodes, maintains heartbeats to keep nodes alive, and retrieves the list of active nodes. This driver plays a crucial role in ensuring that the NodePool has up-to-date information about the state of each node in the system.

- [**ConsistentHash**](consistenthash/): Utilized for evenly distributing jobs across nodes, minimizing reassignment when nodes change, and maintaining load balance.

- [**dcron**](dcron/): package is designed for distributed cron job scheduling within a distributed system. It ensures reliable execution of scheduled tasks across a cluster of nodes by leveraging components like `nodepool` for node management and `RedisDriver` for state synchronization and node communication. Key features include fault tolerance, load balancing, dynamic node management, and support for standard cron syntax for job scheduling. This setup allows `dcron` to efficiently distribute and manage tasks, ensuring high availability and scalability of scheduled jobs across the distributed system.

## Contributing

Feel free to explore, contribute, and shape the future of go-dispatch with us! Your feedback and collaboration are invaluable as we continue to refine and enhance this tool.

To get started, see [CONTRIBUTING.md](./CONTRIBUTING.md).

Please adhere to Starton's [Code of Conduct](./CODE_OF_CONDUCT.md).


## License

go-dispatch is licensed under the [Apache License 2.0](./LICENSE.md).


## Authors

- Starton: [support@starton.com](mailto:support@starton.com)
- Ghislain Cheng
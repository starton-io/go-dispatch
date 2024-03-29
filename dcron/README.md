# Dcron package

The `dcron` package is a distributed cron job scheduling system designed to run scheduled tasks across multiple nodes in a cluster. It leverages Redis as a backend for job storage and synchronization among nodes. Here's a detailed breakdown of its functionality and components:

## Core Features

- **Distributed Execution**: `dcron` allows cron jobs to be executed in a distributed manner across multiple nodes, ensuring high availability and scalability.
- **Job and JobWarpper**: Defines the interface for jobs and a wrapper that includes scheduling information and execution logic.
- **RecentJobPacker**: Manages recently executed jobs, useful for recovering state after cluster changes.
- **Driver**: Interface for backend storage, with a Redis implementation provided for job synchronization across nodes.

## Usage

### Step 1: Define a Job


First, you need to define a job by implementing the [Job interface](job_wrapper.go). A job must have a `Run` method that contains the logic to be executed.

```go
type MyJob struct {
    // Add any fields you need here
}

func (job *MyJob) Run() {
    // Job execution logic goes here
    fmt.Println("Job is running!")
}
```

### Step 2: Initialize `dcron`

Next, create a new `dcron` instance, configure it with a Redis driver for synchronization, and set up any additional options you need.

```go
redisCli := redis.NewClient(&redis.Options{
    Addr: "localhost:6379", // Replace with your Redis server address
})

driver := driver.NewRedisDriver(redisCli)

dcronInstance := dcron.NewDcronWithOption(
    "myServerName",
    driver,
    dcron.WithLogger(logger.DefaultPrintfLogger()),
    dcron.WithHashReplicas(10),
    dcron.WithNodeUpdateDuration(time.Second*10),
)
```

### Step 3: Add a Job

Finally, start the `dcron` instance to begin executing scheduled jobs.

```go
job := &MyJob{}
err := dcronInstance.AddJob("myJob", "*/5 * * * *", job) // Runs every 5 minutes
if err != nil {
    log.Fatalf("Error adding job to dcron: %v", err)
}
```

### Step 4: Start `dcron`

Finally, start the `dcron` instance to begin executing scheduled jobs.

```go
// you can use Start() or  Run() to start the dcron instance.

// unblocking start
dcronInstance.Start()

// blocking start
dcronInstance.Run()
```


### Example
- [examples](examples/)
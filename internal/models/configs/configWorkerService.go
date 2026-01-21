package configs

import "time"

type WorkerServiceConfig struct {
	ExecutionDirectory string
	RunnerPath         string
	ContainerTimeout   time.Duration
	MaxWorkers         int
	QueueSize          int
}

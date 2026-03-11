package client

import (
	"flag"
	"fmt"
	"os"
)

const (
	ExecutionEngineK8s    = "k8s"
	ExecutionEngineDocker = "docker"
)

func ParseExecutionEngineFlag() string {
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: %s [options]\n\n", os.Args[0])
		flag.PrintDefaults()
		os.Exit(1)
	}
	engine := flag.String("engine", ExecutionEngineDocker, "execution engine to use: k8s or docker")
	flag.Parse()

	switch *engine {
	case ExecutionEngineK8s, ExecutionEngineDocker:
		return *engine
	default:
		panic("invalid execution engine: must be k8s or docker")
	}
}

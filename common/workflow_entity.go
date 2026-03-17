package common

import "time"

// WorkflowEntity is a common interface for workflow entities like NextflowContainer and NextflowPod.
// It provides methods to retrieve shared attributes, ensuring compatibility across different workflow entities.
type WorkflowEntity interface {
	// GetStartTime returns the start time of the workflow entity.
	GetStartTime() time.Time

	// GetDieTime returns the end time of the workflow entity.
	GetDieTime() time.Time

	// GetName returns the name of the workflow entity.
	GetName() string

	// GetWorkDir returns the working directory of the workflow entity.
	GetWorkDir() string
}

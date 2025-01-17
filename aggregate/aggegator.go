package aggregate

import "github.com/prometheus/common/model"

type Aggregator interface {

	// Parse outputs from Prometheus so that the csv files can be written.
	// Format for csv Writer:
	// folder<monitoring_target>/file<slurm_job_id>.csv
	CreateMetaDataOutput(m *map[string][]model.Vector) error

	WriteToCSV() error

	Aggreagate() error

	Match() error

	Verify() error
}

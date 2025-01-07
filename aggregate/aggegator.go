package aggregate

import "github.com/prometheus/common/model"

type Aggregator interface {

	// Check the type of the result from Prometheus.
	CheckAndAssertType(rs map[string][]model.Value) (string, error)

	// Parse outputs from Prometheus so that the csv files can be written.
	// Format for csv Writer:
	// folder<monitoring_target>/file<slurm_job_id>.csv
	Parse(bool, map[string][]model.Value) error

	Aggreagate() error

	Write(*map[string][]model.Value) error

	Match() error

	Verify() error
}

package infrastructure

import (
	"context"
	"fmt"
	"os"
	"strconv"
	"strings"

	"meerkat-v0/internal/metrics/domain"
)

// SystemMetricsReaderImpl implements the domain SystemMetricsReader interface
type SystemMetricsReaderImpl struct{}

// NewSystemMetricsReader creates a new system metrics reader implementation
func NewSystemMetricsReader() domain.SystemMetricsReader {
	return &SystemMetricsReaderImpl{}
}

// ReadLoadAvg reads the 1-minute load average from /proc/loadavg
func (r *SystemMetricsReaderImpl) ReadLoadAvg(ctx context.Context) (float64, error) {
	contents, err := os.ReadFile("/proc/loadavg")
	if err != nil {
		return 0, fmt.Errorf("failed to read /proc/loadavg: %w", err)
	}

	values := strings.Split(string(contents), " ")
	if len(values) == 0 {
		return 0, fmt.Errorf("invalid format in /proc/loadavg")
	}

	loadAvg, err := strconv.ParseFloat(values[0], 64)
	if err != nil {
		return 0, fmt.Errorf("failed to parse load average: %w", err)
	}

	return loadAvg, nil
}


package telemetry

import (
	"context"

	"github.com/CovidShield/server/pkg/persistence"

	"github.com/shirou/gopsutil/cpu"
	"github.com/shirou/gopsutil/mem"
	"go.opentelemetry.io/otel/api/global"
	"go.opentelemetry.io/otel/api/metric"
	"go.opentelemetry.io/otel/api/unit"
)

func initSystemStatsObserver(db persistence.Conn) {
	meter := global.Meter("covidshield")

	// Initialize the first CPU measurement so that a percentage will be calculated the next time this method is called
	getCPUPercentage()

	var memTotal metric.Int64ValueObserver
	var memUsedPercent metric.Float64ValueObserver
	var memUsed metric.Int64ValueObserver
	var memAvailable metric.Int64ValueObserver
	var cpuPercent metric.Float64ValueObserver
	var claimedKeysTotalMetric metric.Int64ValueObserver
	var diagnosisKeysTotalMetric metric.Int64ValueObserver
	var unclaimedKeysTotalMetric metric.Int64ValueObserver

	cb := metric.Must(meter).NewBatchObserver(func(_ context.Context, result metric.BatchObserverResult) {
		v, _ := mem.VirtualMemory()
		claimedKeysTotalMetricCount, _ := db.CountClaimedKeys()
		diagnosisKeysTotalMetricCount, _ := db.CountDiagnosisKeys()
		unclaimedKeysTotalMetricCount, _ := db.CountUnclaimedKeys()
		result.Observe(nil,
			memTotal.Observation(int64(v.Total)),
			memUsedPercent.Observation(v.UsedPercent),
			memUsed.Observation(int64(v.Used)),
			memAvailable.Observation(int64(v.Available)),
			cpuPercent.Observation(getCPUPercentage()),
			diagnosisKeysTotalMetric.Observation(diagnosisKeysTotalMetricCount),
			claimedKeysTotalMetric.Observation(claimedKeysTotalMetricCount),
			unclaimedKeysTotalMetric.Observation(unclaimedKeysTotalMetricCount),
		)
	})

	memTotal = cb.NewInt64ValueObserver("covidshield.system.memory.total",
		metric.WithDescription("Total amount of RAM on this system"),
		metric.WithUnit(unit.Bytes),
	)
	memUsedPercent = cb.NewFloat64ValueObserver("covidshield.system.memory.usedpercent",
		metric.WithDescription("RAM available for programs to allocate"),
	)
	memUsed = cb.NewInt64ValueObserver("covidshield.system.memory.used",
		metric.WithDescription("RAM used by programs"),
		metric.WithUnit(unit.Bytes),
	)
	memAvailable = cb.NewInt64ValueObserver("covidshield.system.memory.free",
		metric.WithDescription("Percentage of RAM used by programs"),
		metric.WithUnit(unit.Bytes),
	)
	cpuPercent = cb.NewFloat64ValueObserver("covidshield.system.cpu.percent",
		metric.WithDescription("Percentage of all CPUs combined"),
	)
	claimedKeysTotalMetric = cb.NewInt64ValueObserver("covidshield.app.claimed_keys.total",
		metric.WithDescription("Total number of claimed keys"),
	)
	diagnosisKeysTotalMetric = cb.NewInt64ValueObserver("covidshield.app.diagnosis_keys.total",
		metric.WithDescription("Total number of diagnosis keys"),
	)
	unclaimedKeysTotalMetric = cb.NewInt64ValueObserver("covidshield.app.unclaimed_keys.total",
		metric.WithDescription("Total number of unclaimed keys"),
	)
}

func getCPUPercentage() float64 {
	cpu, _ := cpu.Percent(0, false)
	return cpu[0]
}

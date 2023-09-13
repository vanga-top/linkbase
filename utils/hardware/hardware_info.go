package hardware

import (
	"flag"
	"github.com/linkbase/middleware/log"
	"github.com/shirou/gopsutil/v3/cpu"
	"github.com/shirou/gopsutil/v3/mem"
	"go.uber.org/automaxprocs/maxprocs"
	"go.uber.org/zap"
	syslog "log"
	"runtime"
	"sync"
)

var (
	icOnce sync.Once
	ic     bool
	icErr  error
)

// GetCPUNum returns the count of cpu core.
func GetCPUNum() int {
	cur := runtime.GOMAXPROCS(0)
	if cur <= 0 {
		cur = runtime.NumCPU()
	}
	return cur
}

// Initialize maxprocs
func InitMaxprocs(serverType string, flags *flag.FlagSet) {
	if serverType == "embedded" {
		// Initialize maxprocs while discarding log.
		maxprocs.Set(maxprocs.Logger(nil))
	} else {
		// Initialize maxprocs.
		maxprocs.Set(maxprocs.Logger(syslog.Printf))
	}
}

// GetCPUUsage returns the cpu usage in percentage.
func GetCPUUsage() float64 {
	percents, err := cpu.Percent(0, false)
	if err != nil {
		log.Warn("failed to get cpu usage",
			zap.Error(err))
		return 0
	}

	if len(percents) != 1 {
		log.Warn("something wrong in cpu.Percent, len(percents) must be equal to 1",
			zap.Int("len(percents)", len(percents)))
		return 0
	}

	return percents[0]
}

// GetMemoryCount returns the memory count in bytes.
func GetMemoryCount() uint64 {
	icOnce.Do(func() {
		ic, icErr = inContainer()
	})
	if icErr != nil {
		log.Error(icErr.Error())
		return 0
	}
	// get host memory by `gopsutil`
	stats, err := mem.VirtualMemory()
	if err != nil {
		log.Warn("failed to get memory count",
			zap.Error(err))
		return 0
	}
	// not in container, return host memory
	if !ic {
		return stats.Total
	}

	// get container memory by `cgroups`
	limit, err := getContainerMemLimit()
	if err != nil {
		log.Error(err.Error())
		return 0
	}
	// in container, return min(hostMem, containerMem)
	if limit < stats.Total {
		return limit
	}
	return stats.Total
}

// GetUsedMemoryCount returns the memory usage in bytes.
func GetUsedMemoryCount() uint64 {
	icOnce.Do(func() {
		ic, icErr = inContainer()
	})
	if icErr != nil {
		log.Error(icErr.Error())
		return 0
	}
	if ic {
		// in container, calculate by `cgroups`
		used, err := getContainerMemUsed()
		if err != nil {
			log.Error(err.Error())
			return 0
		}
		return used
	}
	// not in container, calculate by `gopsutil`
	stats, err := mem.VirtualMemory()
	if err != nil {
		log.Warn("failed to get memory usage count",
			zap.Error(err))
		return 0
	}

	return stats.Used
}

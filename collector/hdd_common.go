package collector

/*
#cgo LDFLAGS: -L. -lperfstat
#include <stdio.h>
#include <stdlib.h>
#include <libperfstat.h>

int getDisksInfo(perfstat_disk_total_t *disks_info) {

	int	rc;
	rc = perfstat_disk_total(NULL, disks_info, sizeof(perfstat_disk_total_t), 1);
	if (rc <= 0 ) {
		return rc;
	}
	return 0;
}

int getDiskInfo(perfstat_disk_t *disk_info, u_int64_t disksNumber) {

	int	rc;
	perfstat_id_t id = { "" };
	rc = perfstat_disk(&id, disk_info, sizeof(perfstat_disk_t) * disksNumber, disksNumber);
	if (rc <= 0 ) {
		return rc;
	}
	return 0;
}
*/
import "C"
import (
	"fmt"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/common/log"
	"strings"
	"unsafe"
)

const (
	diskInfoSubsystem = "disk"
)

type diskInfoCollector struct{}

func init() {
	registerCollector("diskinfo", true, NewDiskCollector)
}

func NewDiskCollector() (Collector, error) {
	return &diskInfoCollector{}, nil
}

func (c *diskInfoCollector) Update(ch chan<- prometheus.Metric) error {
	var metricType prometheus.ValueType
	diskInfo, err := c.getInfo()
	if err != nil {
		return fmt.Errorf("couldn't get diskinfo: %s", err)
	}
	for k, v := range diskInfo {
		if strings.HasSuffix(k, "_total") {
			metricType = prometheus.CounterValue
		} else {
			metricType = prometheus.GaugeValue
		}
		ch <- prometheus.MustNewConstMetric(
			prometheus.NewDesc(
				prometheus.BuildFQName(namespace, diskInfoSubsystem, k),
				fmt.Sprintf("Disk information field %s.", k),
				nil, nil,
			),
			metricType, v,
		)
	}
	return nil
}

func (c *diskInfoCollector) getInfo() (map[string]float64, error) {

	var disksInfo C.perfstat_disk_total_t
	var diskInfo C.perfstat_disk_t

	if _, err := C.getDisksInfo(&disksInfo); err != nil {
		return nil, fmt.Errorf("could not collect disk from getDisksInfo: %v", err)
	}
	defer C.free(unsafe.Pointer(&diskInfo))
	if _, err := C.getDiskInfo(&diskInfo, C.uint64_t(disksInfo.number)); err != nil {
		return nil, fmt.Errorf("could not collect disk from getDiskInfo: %v", err)
	}
	defer C.free(unsafe.Pointer(&disksInfo))
	log.Infoln(diskInfo)

	return map[string]float64{
		"number_of_disks":  float64(disksInfo.number),
		"total_disk_size":  float64(disksInfo.size),
		"total_free_space": float64(disksInfo.free),
	}, nil

}

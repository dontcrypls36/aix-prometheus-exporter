package collector

/*
#cgo LDFLAGS: -L. -lperfstat
#include <stdio.h>
#include <stdlib.h>
#include <libperfstat.h>
#include <string.h>

int getDisksInfo(perfstat_disk_total_t *disks_info) {

	int	rc;
	rc = perfstat_disk_total(NULL, disks_info, sizeof(perfstat_disk_total_t), 1);
	if (rc <= 0 ) {
		return rc;
	}
	return 0;
}

int getDiskInfo(char **diskInfo, size_t *disks_len) {
	int	tot, ret, i;
	perfstat_disk_t *di;
	perfstat_id_t first;
	tot = perfstat_disk(NULL, NULL, sizeof(perfstat_disk_t), 0);
	if (tot > 0 ) {
		di = calloc(tot, sizeof(perfstat_disk_t));
		strcpy(first.name, FIRST_DISK);
		*disks_len = tot*7;
		*diskInfo = (char *) malloc(sizeof(char)*(*disks_len));
		ret = perfstat_disk(&first, di, sizeof(perfstat_disk_t), tot);
		for (i = 0; i < ret; i++) {
			int offset = 7 * i;
			diskInfo[offset] =   di[i].description;
			diskInfo[offset+1] = di[i].vgname;
			diskInfo[offset+2] = di[i].adapter;
			diskInfo[offset+3] = di[i].size;
			diskInfo[offset+4] = di[i].free;
			diskInfo[offset+5] = di[i].rblks;
			diskInfo[offset+6] = di[i].wblks;
		}
		return ret;
	}
	return -1;
}
*/
import "C"
import (
	"fmt"
	"github.com/prometheus/client_golang/prometheus"
	"reflect"
	"strings"
	"unsafe"
)

const (
	diskInfoSubsystem = "disk"
	maxDiskLen        = 1024 * 4
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

	if _, err := C.getDisksInfo(&disksInfo); err != nil {
		return nil, fmt.Errorf("could not collect disk from getDisksInfo: %v", err)
	}
	defer C.free(unsafe.Pointer(&disksInfo))
	var (
		diskInfo   *C.char
		diskLength C.size_t
	)
	if _, err := C.getDiskInfo(&diskInfo, &diskLength); err != nil {
		return nil, fmt.Errorf("could not collect disk from getDiskInfo: %v", err)
	}
	defer C.free(unsafe.Pointer(&disksInfo))
	fmt.Println(reflect.TypeOf(diskInfo))
	dput := (*[maxDiskLen]C.char)(unsafe.Pointer(&diskInfo))[:diskLength:diskLength]
	fmt.Println(reflect.TypeOf(dput))

	//cpuTicks := make([]float64, diskLength)
	for _, value := range dput {
		fmt.Println(reflect.TypeOf(value))
	}

	return map[string]float64{
		"number_of_disks": float64(disksInfo.number),
		//"total_disk_size":  float64(disksInfo.size),
		//"total_free_space": float64(disksInfo.free),
	}, nil

}

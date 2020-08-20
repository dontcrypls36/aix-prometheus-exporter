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

int getDiskInfo(char ***outChar, uint64_t **outMetrics, size_t *disks_char_len, size_t *disks_metrics_len) {
	int	tot, ret, i;
	perfstat_disk_t *di;
	perfstat_id_t first;
	tot = perfstat_disk(NULL, NULL, sizeof(perfstat_disk_t), 0);
	if (tot > 0 ) {
		di = calloc(tot, sizeof(perfstat_disk_t));
		strcpy(first.name, FIRST_DISK);
		*disks_char_len = tot * 3;
		*disks_metrics_len = tot * 4;
		char** diskCharInfo = malloc(sizeof(char*)*tot*3);
		*outMetrics = (uint64_t *) malloc(sizeof(uint64_t)*tot*4);
		ret = perfstat_disk(&first, di, sizeof(perfstat_disk_t), tot);
		for (i = 0; i < ret; i++) {
			int offset1 = 3 * i;
			int offset2 = 4 * i;
			diskCharInfo[offset1] =   di[i].description;
			diskCharInfo[offset1+1] = di[i].vgname;
			diskCharInfo[offset1+2] = di[i].adapter;
			(*outMetrics)[offset2] = di[i].size;
			(*outMetrics)[offset2+1] = di[i].free;
			(*outMetrics)[offset2+2] = di[i].rblks;
			(*outMetrics)[offset2+3] = di[i].wblks;
		}
		*outChar = diskCharInfo;
		return ret;
	}
	return -1;
}

char** getDiskInfoMock() {
	char** diskInfo = malloc(sizeof(char*));
	strcpy(diskInfo[0], "FAKE1");
	return diskInfo;
}
*/
import "C"
import (
	"fmt"
	"github.com/prometheus/client_golang/prometheus"
	"unsafe"
)

const (
	diskInfoSubsystem = "disk"
	maxDiskCharLen    = 256 * 3
	maxDiskMetricsLen = 256 * 4
	descrCount        = 3
	metricsCount      = 4
)

type diskInfoCollector struct {
	disk *prometheus.Desc
}

var (
	diskDescMB = prometheus.NewDesc(
		prometheus.BuildFQName(namespace, diskInfoSubsystem, "MB"),
		"Characteristic measured in MB",
		[]string{"description", "vgName", "adapter"}, nil,
	)
	diskDescBytes = prometheus.NewDesc(
		prometheus.BuildFQName(namespace, diskInfoSubsystem, "Bytes"),
		"Characteristic measured in bytes",
		[]string{"description", "vgName", "adapter"}, nil,
	)
)

func init() {
	registerCollector("diskinfo", true, NewDiskCollector)
}

func NewDiskCollector() (Collector, error) {
	return &diskInfoCollector{}, nil
}

func (c *diskInfoCollector) Update(ch chan<- prometheus.Metric) error {
	diskFields := []string{"size", "free", "wblks", "rblks"}
	labelFields := []string{"description", "vgName", "adapter"}
	numbers, descriptions, metrics, err := c.getInfo()
	if err != nil {
		return fmt.Errorf("couldn't get diskinfo: %s", err)
	}
	for i := 0; i < numbers; i++ {
		descrSlice := descriptions[(i * descrCount):((i + 1) * descrCount)]
		metricSlice := metrics[(i * metricsCount):((i + 1) * metricsCount)]
		for i, value := range metricSlice {
			ch <- prometheus.MustNewConstMetric(
				prometheus.NewDesc(
					prometheus.BuildFQName(namespace, diskInfoSubsystem, diskFields[i%metricsCount]),
					fmt.Sprintf("Disk information field %s.", diskFields[i%metricsCount]),
					labelFields, nil,
				),
				prometheus.GaugeValue, value, descrSlice...)
		}

	}
	return nil
}

func (c *diskInfoCollector) getInfo() (int, []string, []float64, error) {
	var (
		diskCharInfo      **C.char
		diskMetricsInfo   *C.uint64_t
		diskCharLength    C.size_t
		diskMetricsLength C.size_t
	)
	if disksNumber, err := C.getDiskInfo(&diskCharInfo, &diskMetricsInfo, &diskCharLength, &diskMetricsLength); err != nil {
		return 0, nil, nil, fmt.Errorf("could not collect disk from getDiskInfo: %v", err)
	} else {
		defer C.free(unsafe.Pointer(&diskCharInfo))
		defer C.free(unsafe.Pointer(&diskMetricsInfo))
		charInfo := (*[maxDiskCharLen]*C.char)(unsafe.Pointer(diskCharInfo))[:diskCharLength:diskCharLength]
		metricsInfo := (*[maxDiskMetricsLen]C.u_longlong_t)(unsafe.Pointer(diskMetricsInfo))[:diskMetricsLength:diskMetricsLength]
		descriptions := make([]string, int(disksNumber)*descrCount)
		metrics := make([]float64, int(disksNumber)*metricsCount)
		for i := 0; i < int(disksNumber); i++ {
			descriptions[i*descrCount] = C.GoString(charInfo[i*descrCount])
			descriptions[i*descrCount+1] = C.GoString(charInfo[i*descrCount+1])
			descriptions[i*descrCount+2] = C.GoString(charInfo[i*descrCount+2])
			metrics[i*metricsCount] = float64(metricsInfo[i*metricsCount])
			metrics[i*metricsCount+1] = float64(metricsInfo[i*metricsCount+1])
			metrics[i*metricsCount+2] = float64(metricsInfo[i*metricsCount+2])
			metrics[i*metricsCount+3] = float64(metricsInfo[i*metricsCount+3])
		}
		return int(disksNumber), descriptions, metrics, nil
	}

}

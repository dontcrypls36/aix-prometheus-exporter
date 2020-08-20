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
	int descrCount = 3;
	int metricsCount = 28;
	tot = perfstat_disk(NULL, NULL, sizeof(perfstat_disk_t), 0);
	if (tot > 0 ) {
		di = calloc(tot, sizeof(perfstat_disk_t));
		strcpy(first.name, FIRST_DISK);
		*disks_char_len = tot * descrCount;
		*disks_metrics_len = tot * metricsCount;
		char** diskCharInfo = malloc(sizeof(char*)*tot*descrCount);
		*outMetrics = (uint64_t *) malloc(sizeof(uint64_t)*tot*metricsCount);
		ret = perfstat_disk(&first, di, sizeof(perfstat_disk_t), tot);
		for (i = 0; i < ret; i++) {
			int offset1 = descrCount * i;
			int offset2 = metricsCount * i;
			diskCharInfo[offset1] =   di[i].description;
			diskCharInfo[offset1+1] = di[i].vgname;
			diskCharInfo[offset1+2] = di[i].adapter;
			(*outMetrics)[offset2] = di[i].size;
			(*outMetrics)[offset2+1] = di[i].free;
			(*outMetrics)[offset2+2] = di[i].rblks;
			(*outMetrics)[offset2+3] = di[i].wblks;
			(*outMetrics)[offset2+4] = di[i].bsize;
			(*outMetrics)[offset2+5] = di[i].xrate;
			(*outMetrics)[offset2+6] = di[i].xfers;
			(*outMetrics)[offset2+7] = di[i].qdepth;
			(*outMetrics)[offset2+8] = di[i].time;
			(*outMetrics)[offset2+9] = di[i].paths_count;
			(*outMetrics)[offset2+10] = di[i].q_full;
			(*outMetrics)[offset2+11] = di[i].rserv;
			(*outMetrics)[offset2+12] = di[i].rtimeout;
			(*outMetrics)[offset2+13] = di[i].rfailed;
			(*outMetrics)[offset2+14] = di[i].min_rserv;
			(*outMetrics)[offset2+15] = di[i].max_rserv;
			(*outMetrics)[offset2+16] = di[i].wserv;
			(*outMetrics)[offset2+17] = di[i].wtimeout;
			(*outMetrics)[offset2+18] = di[i].wfailed;
			(*outMetrics)[offset2+19] = di[i].min_wserv;
			(*outMetrics)[offset2+20] = di[i].max_wserv;
			(*outMetrics)[offset2+21] = di[i].wq_depth;
			(*outMetrics)[offset2+22] = di[i].wq_sampled;
			(*outMetrics)[offset2+23] = di[i].wq_time;
			(*outMetrics)[offset2+24] = di[i].wq_min_time;
			(*outMetrics)[offset2+25] = di[i].wq_max_time;
			(*outMetrics)[offset2+26] = di[i].q_sampled;
			(*outMetrics)[offset2+27] = di[i].version;
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
	descrCount        = 3
	metricsCount      = 28
	maxDiskCharLen    = 256 * descrCount
	maxDiskMetricsLen = 256 * metricsCount
)

type diskInfoCollector struct {
}

func init() {
	registerCollector("diskinfo", true, NewDiskCollector)
}

func NewDiskCollector() (Collector, error) {
	return &diskInfoCollector{}, nil
}

func (c *diskInfoCollector) Update(ch chan<- prometheus.Metric) error {
	diskFields := []string{"size_of_the_disk", "free_portion_of_the_disk", "number_of_blocks_written_to", "number_of_blocks_read_from", "disk_block_size", "xrate_capability", "number_of_transfers_to_from", "instantaneous_service_queue_depth",
		"amount_of_time_disk_is_active", "number_of_paths_to_disk", "service_queue_full_occurrence_count", "read_or_receive_service_time", "number_of_read_request_timeouts", "number_of_failed_read_requests", "min_read_or_receive_service_time",
		"max_read_or_receive_service_time", "write_or_send_service_time", "number_of_write_request_timeouts", "number_of_failed_request_timeout", "min_write_or_send_service_time", "max_write_or_send_service_time", "instantaneous_wait_queue_depth",
		"accumulated_sampled_dk_wq_depth", "accumulated_wait_queueing_time", "min_wait_queueing_time", "max_wait_queueing_time", "accumulated_sampled_dk_q_depth", "version_number"}
	labelFields := []string{"description", "volume_group_name", "disk_adapter_name"}
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
			for j := i * metricsCount; j < int(disksNumber)*metricsCount; j++ {
				metrics[j] = float64(metricsInfo[j])
			}
			//metrics[i*metricsCount] = float64(metricsInfo[i*metricsCount])
			//metrics[i*metricsCount+1] = float64(metricsInfo[i*metricsCount+1])
			//metrics[i*metricsCount+2] = float64(metricsInfo[i*metricsCount+2])
			//metrics[i*metricsCount+3] = float64(metricsInfo[i*metricsCount+3])
		}
		return int(disksNumber), descriptions, metrics, nil
	}

}

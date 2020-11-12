package main


import (

    "fmt"
    // "strconv"
    "encoding/csv"
    "bytes"
    "strings"
    "os/exec"
    "github.com/prometheus/common/log"
    "github.com/prometheus/client_golang/prometheus"
)


/**
//===================================================
//========= Define The Exporter Constants ===========
//===================================================
*/

// Default values
const (
    TITLE = "Nvidia SMI Exporter"
    NAME = "nvidia_smi_exporter"
    LISTEN_ADDRESS = ":9202"
    METRICS_PATH = "/metrics"
    NVIDIA_SMI_PATH = "/usr/bin/nvidia-smi"
    SERVICE_NAME = "nvidia_smi_exporter"
)

/**
//===================================================
//================ METRICS DEF ======================
//===================================================
*/

var (
    // Create a guage to track the GPU utilisation
    utilizationGpu = prometheus.NewGaugeVec(
        prometheus.GaugeOpts{
            Name:   "nvidia_utilization_gpu",
            Help:   "Percent of time over the past sample period during which one or more kernels was executing on the GPU.",
        },
        []string{"gpu"},
    )
)

func init() {
    // Register the summary and the histogram with Prometheus's default registry.
    prometheus.MustRegister(utilizationGpu)

    // Add Go module build info.
    prometheus.MustRegister(prometheus.NewBuildInfoCollector())
}


/**
//===================================================
//================ UPDATE METRICS  ==================
//===================================================
*/

func metricsUpdate() {
    log.Infoln("metricsUpdate()")
    utilizationGpu.With(prometheus.Labels{"gpu":"0"}).Inc()
}

// name, index, temperature.gpu, utilization.gpu,
// utilization.memory, memory.total, memory.free, memory.used

//nvidia-smi --query-gpu=name,index,temperature.gpu,utilization.gpu,utilization.memory,memory.total,memory.free,memory.used --format=csv,noheader,nounits

/**
* main metrics
*/
func metricsHtml() string {
    out, err := exec.Command(
        "nvidia-smi",
        "--query-gpu=name,index,temperature.gpu,utilization.gpu,utilization.memory,memory.total,memory.free,memory.used",
        "--format=csv,noheader,nounits").Output()

    if err != nil {
        return fmt.Sprintf("%s\n", err)
    }

    csvReader := csv.NewReader(bytes.NewReader(out))
    csvReader.TrimLeadingSpace = true
    records, err := csvReader.ReadAll()

    if err != nil {
        return fmt.Sprintf("%s\n", err)
    }

    metricList := []string {
        "temperature.gpu", "utilization.gpu",
        "utilization.memory", "memory.total", "memory.free", "memory.used"}

    result := ""
    for _, row := range records {
        name := fmt.Sprintf("%s[%s]", row[0], row[1])
        for idx, value := range row[2:] {
            result = fmt.Sprintf(
                "%s%s{gpu=\"%s\"} %s\n", result,
                metricList[idx], name, value)
        }
    }

    html := strings.Replace(result, ".", "_", -1)
    return html
}

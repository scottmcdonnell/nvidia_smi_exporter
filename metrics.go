package main


import (

    "fmt"
    "strconv"
    "encoding/csv"
    "encoding/xml"
    "bytes"
    "strings"
    "regexp"
    "os/exec"
    "github.com/prometheus/common/log"
    "github.com/prometheus/client_golang/prometheus"
)

/**
NVidia System Management Exporter
https://developer.nvidia.com/nvidia-system-management-interface
http://developer.download.nvidia.com/compute/DCGM/docs/nvidia-smi-367.38.pdf

@see https://github.com/phstudy/nvidia_smi_exporter
@see https://github.com/zhebrak/nvidia_smi_exporter
*/

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
	// Create a gauge to track the GPU utilisation
    driverInfo = prometheus.NewGaugeVec(
        prometheus.GaugeOpts{
            Name:   "nvidia_driver_info",
            Help:   "Nvidia driver information",
        },
        []string{"version"},
    )

    // Create a gauge to track number of GPUs in the machine
    deviceCount = prometheus.NewGauge(
        prometheus.GaugeOpts{
            Name:   "nvidia_device_count",
            Help:   "Number of GPUs in the machine",
        },
    )

    // Create a gauge to track the GPU utilisation
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
    prometheus.MustRegister(driverInfo)
    prometheus.MustRegister(deviceCount)
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
    metricsXml()
}


func metricsXml() {

	//look at different paths this may be discoverable at
	//windows it is in System32
	nvidiaSmiPath := "nvidia-smi"

    cmd := exec.Command(nvidiaSmiPath, "-q", "-x")

    // Execute system command
    stdout, err := cmd.Output()
    if err != nil {
        log.Errorln(err.Error())
        return
    }

    // Parse XML
    var xmlData NvidiaSmiLog
    xml.Unmarshal(stdout, &xmlData)

    driverInfo.With(prometheus.Labels{"version": xmlData.DriverVersion}).Set(1)
    deviceCount.Set(filterNumber(xmlData.AttachedGPUs))

}

/**
* get metrics based on csv
/ name, index, temperature.gpu, utilization.gpu, utilization.memory, memory.total, memory.free, memory.used
//nvidia-smi --query-gpu=name,index,temperature.gpu,utilization.gpu,utilization.memory,memory.total,memory.free,memory.used --format=csv,noheader,nounits

*/
func metricsCsv() string {
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

/**
//===================================================
//================ METRIC PARSE  ====================
//===================================================
*/

// @see https://github.com/phstudy/nvidia_smi_exporter/blob/master/src/nvidia_smi_exporter.go
type NvidiaSmiLog struct {
    DriverVersion string `xml:"driver_version"`
    AttachedGPUs string `xml:"attached_gpus"`
    GPUs []struct {
        ProductName string `xml:"product_name"`
        ProductBrand string `xml:"product_brand"`
        UUID string `xml:"uuid"`
        FanSpeed string `xml:"fan_speed"`
        PCI struct {
            PCIBus string `xml:"pci_bus"`
        } `xml:"pci"`
        FbMemoryUsage struct {
            Total string `xml:"total"`
            Used string `xml:"used"`
            Free string `xml:"free"`
        } `xml:"fb_memory_usage"`
        Utilization struct {
            GPUUtil string `xml:"gpu_util"`
            MemoryUtil string `xml:"memory_util"`
        } `xml:"utilization"`
        Temperature struct {
            GPUTemp string `xml:"gpu_temp"`
            GPUTempMaxThreshold string `xml:"gpu_temp_max_threshold"`
            GPUTempSlowThreshold string `xml:"gpu_temp_slow_threshold"`
        } `xml:"temperature"`
        PowerReadings struct {
            PowerDraw string `xml:"power_draw"`
            PowerLimit string `xml:"power_limit"`
        } `xml:"power_readings"`
        Clocks struct {
            GraphicsClock string `xml:"graphics_clock"`
            SmClock string `xml:"sm_clock"`
            MemClock string `xml:"mem_clock"`
            VideoClock string `xml:"video_clock"`
        } `xml:"clocks"`
        MaxClocks struct {
            GraphicsClock string `xml:"graphics_clock"`
            SmClock string `xml:"sm_clock"`
            MemClock string `xml:"mem_clock"`
            VideoClock string `xml:"video_clock"`
        } `xml:"max_clocks"`
    } `xml:"gpu"`
}

func formatValue(key string, meta string, value string) string {
    result := key;
    if (meta != "") {
        result += "{" + meta + "}";
    }
    return result + " " + value +"\n"
}

func filterNumber(value string) float64 {
    r := regexp.MustCompile("[^0-9.]")
    v := r.ReplaceAllString(value, "")
    f, err := strconv.ParseFloat(v, 64)

    if err != nil {
        log.Errorln("%s\n", err)
    }
    return f
}


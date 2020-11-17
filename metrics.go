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
    "math"
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
    NVIDIA_SMI_PATH_LINUX = "/usr/bin/nvidia-smi"
    NVIDIA_SMI_PATH_WINDOWS = "nvidia-smi"
    SERVICE_NAME = "nvidia_smi_exporter"

    COMMAND_APP = "nvidia-smi"
    COMMAND_FLAGS = "-q -x"
)
var COMMAND_APP_PATHS = []string {
    "C:\\Program Files\\NVIDIA Corporation\\NVSMI\\nvidia-smi.exe",
    "C:\\Windows\\System32\\nvidia-smi.exe",
    //"C:\\Windows\\System32\\DriverStore\\FileRepository\\nvdm*\\nvidia-smi.exe",
    "/usr/bin/nvidia-smi",
}

/**
//===================================================
//================ METRICS DEF ======================
//===================================================
*/

var (
    // create our metrics  
    exporterInfo = prometheus.NewGaugeVec(
        prometheus.GaugeOpts{
            Name:   "nvidia_smi_exporter_build_info",
            Help:   "Exporter build information",
        },
        []string{"version"},
    )
    collectorSuccess = prometheus.NewGauge(
        prometheus.GaugeOpts{
            Name:   "nvidia_smi_collector_success",
            Help:   "nvidia_smi_exporter: Whether the collector was successful.",
        },
    )
    driverInfo = prometheus.NewGaugeVec(
        prometheus.GaugeOpts{
            Name:   "nvidia_driver_info",
            Help:   "Nvidia driver information",
        },
        []string{"version"},
    )
    deviceCount = prometheus.NewGauge(
        prometheus.GaugeOpts{
            Name:   "nvidia_device_count",
            Help:   "Number of GPUs in the machine",
        },
    )
    gpuInfo = prometheus.NewGaugeVec(
        prometheus.GaugeOpts{
            Name:   "nvidia_info",
            Help:   "GPU device information",
        },
        []string{"gpu", "name", "uuid", "vbios"},
    )
    gpuFanSpeed = prometheus.NewGaugeVec(
        prometheus.GaugeOpts{
            Name:   "nvidia_fanspeed_ratio",
            Help:   "The fan speed value is the percent of maximum speed from 0 to 1 that the device's fan is currently intended to run at",
        },
        []string{"gpu"},
    )
    gpuMemory = prometheus.NewGaugeVec(
        prometheus.GaugeOpts{
            Name:   "nvidia_memory_bytes",
            Help:   "FB Memory Usage - On-board frame buffer memory information in bytes.",
        },
        []string{"gpu", "state"},
    )
    gpuTemperature = prometheus.NewGaugeVec(
        prometheus.GaugeOpts{
            Name:   "nvidia_temperature_celsius",
            Help:   "Percent of time over the past sample period during which one or more kernels was executing on the GPU.",
        },
        []string{"gpu"},
    )
    gpuTemperatureMax = prometheus.NewGaugeVec(
        prometheus.GaugeOpts{
            Name:   "nvidia_temperature_max_celsius",
            Help:   "Maximum temperature in Celsius for the GPU.",
        },
        []string{"gpu"},
    )
    gpuTemperatureSlow = prometheus.NewGaugeVec(
        prometheus.GaugeOpts{
            Name:   "nvidia_temperature_slow_celsius",
            Help:   "Temperature in Celsius where the GPU will start to slow.",
        },
        []string{"gpu"},
    )
    gpuPower = prometheus.NewGaugeVec(
        prometheus.GaugeOpts{
            Name:   "nvidia_power_watts",
            Help:   "The last measured power draw for the entire board, in watts",
        },
        []string{"gpu"},
    )
    gpuPowerLimit = prometheus.NewGaugeVec(
        prometheus.GaugeOpts{
            Name:   "nvidia_power_limit_watts",
            Help:   "The Limit power is set to in watts",
        },
        []string{"gpu"},
    )

    gpuUtilization = prometheus.NewGaugeVec(
        prometheus.GaugeOpts{
            Name:   "nvidia_utilization_ratio",
            Help:   "Utilization rates report how busy each part of the GPU is over the past sample period. each part can be 0-1.",
        },
        []string{"gpu", "part"},
    )
    gpuClock = prometheus.NewGaugeVec(
        prometheus.GaugeOpts{
            Name:   "nvidia_clock_mhz",
            Help:   "Current frequency at which parts of the GPU are running. All readings are in MHz.",
        },
        []string{"gpu", "part"},
    )
    gpuClockMax = prometheus.NewGaugeVec(
        prometheus.GaugeOpts{
            Name:   "nvidia_clock_max_mhz",
            Help:   "Maximum frequency at which parts of the GPU are design to run. Al readings are in MHz.",
        },
        []string{"gpu", "part"},
    )
)

func init() {
    // Register all metrics.
    prometheus.MustRegister(exporterInfo)
    prometheus.MustRegister(collectorSuccess)
    prometheus.MustRegister(driverInfo)
    prometheus.MustRegister(deviceCount)
    prometheus.MustRegister(gpuFanSpeed)
    prometheus.MustRegister(gpuInfo)
    prometheus.MustRegister(gpuMemory)
    prometheus.MustRegister(gpuTemperature)
    prometheus.MustRegister(gpuTemperatureMax)
    prometheus.MustRegister(gpuTemperatureSlow)
    prometheus.MustRegister(gpuPower)
    prometheus.MustRegister(gpuPowerLimit)
    prometheus.MustRegister(gpuUtilization)
    prometheus.MustRegister(gpuClock)
    prometheus.MustRegister(gpuClockMax)

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
    //set the version from the current git label
    exporterInfo.With(prometheus.Labels{"version": version}).Set(1)
    
    //make sure its 0 until the end of this method
    collectorSuccess.Set(0)

    //get the set commandaAppPath
    command := *commandAppPath
    flags := *commandFlags
    f := strings.Split(flags, " ")

    // create our command - unpack the array of flags
    cmd := exec.Command(command, f...)
    // log.Debugf("command:", cmd.String())
    log.Infoln("command:", cmd.String())

    // Execute system command
    stdout, err := cmd.Output()
    if err != nil {
        log.Errorln(err.Error())
        return
    }

    // Parse XML
    var xmlData NvidiaSmiLog
    xml.Unmarshal(stdout, &xmlData)

    // SANITY CHECK results
    if xmlData.DriverVersion =="" {
        log.Errorln("Nvidia DriverVersion not parsed correctly")
        return
    }


    driverInfo.With(prometheus.Labels{"version": xmlData.DriverVersion}).Set(1)
    deviceCount.Set(filterNumber(xmlData.AttachedGPUs))

    // for each GPU
    for i, GPU := range xmlData.GPUs {
        var idx = strconv.Itoa(i)

        gpuInfo.With(prometheus.Labels{"gpu": idx, "name": GPU.ProductName, "uuid": GPU.UUID, "vbios": GPU.VBiosVersion}).Set(1)
        gpuFanSpeed.With(prometheus.Labels{"gpu": idx}).Set(filterNumber(GPU.FanSpeed)/100)

        gpuMemory.With(prometheus.Labels{"gpu": idx, "state": "used"}).Set(megabytesToBytes(filterNumber(GPU.FbMemoryUsage.Used)))
        gpuMemory.With(prometheus.Labels{"gpu": idx, "state": "free"}).Set(megabytesToBytes(filterNumber(GPU.FbMemoryUsage.Free)))

        gpuTemperature.With(prometheus.Labels{"gpu": idx}).Set(filterNumber(GPU.Temperature.GPUTemp))
        gpuTemperatureMax.With(prometheus.Labels{"gpu": idx}).Set(filterNumber(GPU.Temperature.GPUTempMaxThreshold))
        gpuTemperatureSlow.With(prometheus.Labels{"gpu": idx}).Set(filterNumber(GPU.Temperature.GPUTempSlowThreshold))

        gpuPower.With(prometheus.Labels{"gpu": idx}).Set(filterNumber(GPU.PowerReadings.PowerDraw))
        gpuPowerLimit.With(prometheus.Labels{"gpu": idx}).Set(filterNumber(GPU.PowerReadings.PowerLimit))

        gpuUtilization.With(prometheus.Labels{"gpu": idx, "part": "gpu"}).Set(filterNumber(GPU.Utilization.GPUUtil)/100)
        gpuUtilization.With(prometheus.Labels{"gpu": idx, "part": "memory"}).Set(filterNumber(GPU.Utilization.MemoryUtil)/100)
        gpuUtilization.With(prometheus.Labels{"gpu": idx, "part": "encoder"}).Set(filterNumber(GPU.Utilization.EncoderUtil)/100)
        gpuUtilization.With(prometheus.Labels{"gpu": idx, "part": "decoder"}).Set(filterNumber(GPU.Utilization.DecoderUtil)/100)

        gpuClock.With(prometheus.Labels{"gpu": idx, "part": "graphics"}).Set(filterNumber(GPU.Clocks.GraphicsClock))
        gpuClock.With(prometheus.Labels{"gpu": idx, "part": "sm"}).Set(filterNumber(GPU.Clocks.SmClock))
        gpuClock.With(prometheus.Labels{"gpu": idx, "part": "memory"}).Set(filterNumber(GPU.Clocks.MemClock))
        gpuClock.With(prometheus.Labels{"gpu": idx, "part": "video"}).Set(filterNumber(GPU.Clocks.VideoClock))

        gpuClockMax.With(prometheus.Labels{"gpu": idx, "part": "graphics"}).Set(filterNumber(GPU.MaxClocks.GraphicsClock))
        gpuClockMax.With(prometheus.Labels{"gpu": idx, "part": "sm"}).Set(filterNumber(GPU.MaxClocks.SmClock))
        gpuClockMax.With(prometheus.Labels{"gpu": idx, "part": "memory"}).Set(filterNumber(GPU.MaxClocks.MemClock))
        gpuClockMax.With(prometheus.Labels{"gpu": idx, "part": "video"}).Set(filterNumber(GPU.MaxClocks.VideoClock))

    }
    collectorSuccess.Set(1)
}

func megabytesToBytes(mb float64) float64 {
    return math.Round(mb * 1048576)
}

func mHzToHz(mHz int) float64 {
    return float64(mHz) * 1000 * 1000
}


/**
* get metrics based on csv
/ name, index, temperature.gpu, utilization.gpu, utilization.memory, memory.total, memory.free, memory.used
//nvidia-smi --query-gpu=name,index,temperature.gpu,utilization.gpu,utilization.memory,memory.total,memory.free,memory.used --format=csv,noheader,nounits
*/
func metricsCsv() string {

    nvidiaSmiPath := *commandAppPath
 
    out, err := exec.Command(
        nvidiaSmiPath,
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
        VBiosVersion string `xml:"vbios_version"`
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
            EncoderUtil string `xml:"encoder_util"`
            DecoderUtil string `xml:"decoder_util"`
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
        Processes []struct {
            ProcessInfo struct {
                ProcessName string `xml:"process_name"`
                UsedMemory string `xml:"used_memory"`
                Type string `xml:"type"`
                PID string `xml:"pid"`
                GPUInstanceId string `xml:"gpu_instance_id"`
                ComputeInstanceId string `xml:"compute_instance_id"`
            } `xml:"process_info"`
        } `xml:"processes"`
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


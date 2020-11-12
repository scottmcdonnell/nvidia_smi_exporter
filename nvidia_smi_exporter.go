// +build windows

package main

import (
    //"io"
    "bytes"
    "encoding/csv"
    "fmt"
    "net/http"
    "os"
    "os/exec"
    "strings"
    "strconv"

    "github.com/prometheus/common/log"
    "github.com/prometheus/client_golang/prometheus"
    "github.com/prometheus/client_golang/prometheus/promhttp"
    "golang.org/x/sys/windows/svc"
    // "github.com/StackExchange/wmi"
)
/**
NVidia System Management Exporter
https://developer.nvidia.com/nvidia-system-management-interface
http://developer.download.nvidia.com/compute/DCGM/docs/nvidia-smi-367.38.pdf

@see https://github.com/phstudy/nvidia_smi_exporter
@see 
*/

// Default values
const (
    TITLE = "Nvidia SMI Exporter"
    NAME = "nvidia_smi_exporter"
    LISTEN_ADDRESS = ":9202"
    METRICS_PATH = "/metrics"
    VERSION = "0.0.0"
    NVIDIA_SMI_PATH = "/usr/bin/nvidia-smi"
    SERVICE_NAME = "nvidia_smi_exporter"
)

var (
    listenAddress string
    metricsPath string
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


/**
//===================================================
//================ SERVER ===========================
//===================================================
*/
func metrics(w http.ResponseWriter, r *http.Request) {
    metricsUpdate()

    h := promhttp.HandlerFor(
        prometheus.DefaultGatherer, 
        promhttp.HandlerOpts{},
    )
    h.ServeHTTP(w, r)
}
    

// https://github.com/prometheus-community/windows_exporter/blob/master/exporter.go
func timeoutSeconds(r *http.Request) float64 {
    // == TIMEOUT for long running collectors
    const defaultTimeout = 10.0
    var t float64
    if v := r.Header.Get("X-Prometheus-Scrape-Timeout-Seconds"); v != "" {
        var err error
        t, err = strconv.ParseFloat(v, 64)
        if err != nil {
            log.Warnf("Couldn't parse X-Prometheus-Scrape-Timeout-Seconds: %q. Defaulting timeout to %f", v, defaultTimeout)
        }
    }
    if t == 0 {
        t = defaultTimeout
    }
    //t = t - mh.timeoutMargin
    return t
}

/**
* index page shows metrics path
*/
func index(w http.ResponseWriter, r *http.Request) {
    log.Debugf("Serving /index")

    html := fmt.Sprintf(
`<!doctype html>
<html>
    <head>
        <meta charset="utf-8">
        <title>%s</title>
    </head>
    <body>
        <h1>Nvidia SMI Exporter</h1>
        <p><a href="%s">Metrics</a></p>
    </body>
</html>`, TITLE, metricsPath)

    outputHtml(w, html)
}

/**
* health check page for {"status":"ok"}
*/
func healthCheck(w http.ResponseWriter, r *http.Request) {
    log.Debugf("Serving /health")

    w.Header().Set("Content-Type", "application/json")
    outputHtml(w, `{"status":"ok"}`)
}

func outputHtml(w http.ResponseWriter, s string) {
    _, err := fmt.Fprintln(w, s)
    if err != nil {
        log.Debugf("Failed to write to stream: %v", err)
    }
}

/**
//===================================================
//================ SERVICE ==========================
//===================================================
*/
/*
func initWbem() {
    // This initialization prevents a memory leak on WMF 5+. See
    // https://github.com/prometheus-community/windows_exporter/issues/77 and
    // linked issues for details.
    log.Debugf("Initializing SWbemServices")
    s, err := wmi.InitializeSWbemServices(wmi.DefaultClient)
    if err != nil {
        log.Fatal(err)
    }
    wmi.DefaultClient.AllowMissingFields = true
    wmi.DefaultClient.SWbemServicesClient = s
}
*/

type exporterService struct {
    stopCh chan<- bool
}

func (s *exporterService) Execute(args []string, r <-chan svc.ChangeRequest, changes chan<- svc.Status) (ssec bool, errno uint32) {
    const cmdsAccepted = svc.AcceptStop | svc.AcceptShutdown
    changes <- svc.Status{State: svc.StartPending}
    changes <- svc.Status{State: svc.Running, Accepts: cmdsAccepted}
loop:
    for {
        select {
        case c := <-r:
            switch c.Cmd {
            case svc.Interrogate:
                changes <- c.CurrentStatus
            case svc.Stop, svc.Shutdown:
                s.stopCh <- true
                break loop
            default:
                log.Error(fmt.Sprintf("unexpected control request #%d", c))
            }
        }
    }
    changes <- svc.Status{State: svc.StopPending}
    return
}

/*func 

isInteractive, err := svc.IsAnInteractiveSession()
    if err != nil {
        log.Fatal(err)
    }

    stopCh := make(chan bool)
    if !isInteractive {
        go func() {
            err = svc.Run(serviceName, &windowsExporterService{stopCh: stopCh})
            if err != nil {
                log.Errorf("Failed to start service: %v", err)
            }
        }()
    } 
*/

/**
//===================================================
//================ MAIN =============================
//===================================================
*/
func main() {

    listenAddress = os.Getenv("LISTEN_ADDRESS")
    if (len(listenAddress) == 0) {
        listenAddress = LISTEN_ADDRESS
    }
    metricsPath = METRICS_PATH


    isInteractive, err := svc.IsAnInteractiveSession()
    if err != nil {
        log.Fatal(err)
    }

    stopCh := make(chan bool)
    if !isInteractive {
        go func() {
            err = svc.Run(SERVICE_NAME, &exporterService{stopCh: stopCh})
            if err != nil {
                log.Errorf("Failed to start service: %v", err)
            }
        }()
    }


    http.HandleFunc("/", index)
    http.HandleFunc("/health", healthCheck)
    http.HandleFunc(metricsPath, metrics)
      
    
    go func() {
        log.Infoln(TITLE, "listening on", listenAddress)
        log.Fatalf("cannot start %s: %s", NAME, http.ListenAndServe(listenAddress, nil))
    }()
    for {
        if <-stopCh {
            log.Info("Shutting down ", SERVICE_NAME)
            break
        }
    }
}





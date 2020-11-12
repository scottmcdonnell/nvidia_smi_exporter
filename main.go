// +build windows

package main

import (
    //"io"
    // "bytes"
    // "encoding/csv"
    "fmt"
    "net/http"
    "os"
    // "os/exec"
    // "strings"
    "strconv"

    "golang.org/x/sys/windows/svc"

    "github.com/prometheus/common/log"
    "github.com/prometheus/client_golang/prometheus"
    "github.com/prometheus/client_golang/prometheus/promhttp"
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
//================ CONSTANTS  =======================
//===================================================
*/



var (
    version string
    listenAddress string
    metricsPath string
)

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
        <p><i>Version: %s</i></p>
    </body>
</html>`, TITLE, metricsPath, version)

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




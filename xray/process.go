package xray

import (
    "bytes"
    "encoding/json"
    "errors"
    "fmt"
    "io/fs"
    "os"
    "os/exec"
    "path/filepath"
    "runtime"
    "syscall"
    "time"

    "x-ui/config"
    "x-ui/logger"
    "x-ui/util/common"
)

// استفاده از filepath.Join برای سازگاری با تمام سیستم عامل‌ها
func GetBinaryName() string {
    return fmt.Sprintf("xray-%s-%s", runtime.GOOS, runtime.GOARCH)
}

func GetBinaryPath() string {
    return filepath.Join(config.GetBinFolderPath(), GetBinaryName())
}

func GetConfigPath() string {
    return filepath.Join(config.GetBinFolderPath(), "config.json")
}

func GetGeositePath() string {
    return filepath.Join(config.GetBinFolderPath(), "geosite.dat")
}

func GetGeoipPath() string {
    return filepath.Join(config.GetBinFolderPath(), "geoip.dat")
}

func GetIPLimitLogPath() string {
    return filepath.Join(config.GetLogFolder(), "3xipl.log")
}

func GetIPLimitBannedLogPath() string {
    return filepath.Join(config.GetLogFolder(), "3xipl-banned.log")
}

func GetIPLimitBannedPrevLogPath() string {
    return filepath.Join(config.GetLogFolder(), "3xipl-banned.prev.log")
}

func GetAccessPersistentLogPath() string {
    return filepath.Join(config.GetLogFolder(), "3xipl-ap.log")
}

func GetAccessPersistentPrevLogPath() string {
    return filepath.Join(config.GetLogFolder(), "3xipl-ap.prev.log")
}

// --- ارتقا 1: پارس امن JSON ---
func GetAccessLogPath() (string, error) {
    configData, err := os.ReadFile(GetConfigPath())
    if err != nil {
        logger.Warningf("Failed to read configuration file: %s", err)
        return "", err
    }

    jsonConfig := map[string]json.RawMessage{}
    err = json.Unmarshal(configData, &jsonConfig)
    if err != nil {
        logger.Warningf("Failed to parse JSON configuration: %s", err)
        return "", err
    }

    if rawLog, exists := jsonConfig["log"]; exists {
        var logConfig map[string]any
        if err := json.Unmarshal(rawLog, &logConfig); err == nil {
            if accessPath, ok := logConfig["access"].(string); ok {
                return accessPath, nil
            }
        }
    }
    return "", errors.New("access log path not found in configuration")
}

func stopProcess(p *Process) {
    // حذف Finalizer، بهتر است توقف صریحاً مدیریت شود
}

type Process struct {
    *process
}

func NewProcess(xrayConfig *Config) *Process {
    p := &Process{newProcess(xrayConfig)}
    // حذف: runtime.SetFinalizer(p, stopProcess)
    return p
}

type process struct {
    cmd *exec.Cmd

    version string
    apiPort int

    onlineClients []string

    config    *Config
    logWriter *LogWriter
    exitErr   error
    startTime time.Time
}

func newProcess(config *Config) *process {
    return &process{
        version:   "Unknown",
        config:    config,
        logWriter: NewLogWriter(),
        startTime: time.Now(),
    }
}

func (p *process) IsRunning() bool {
    if p.cmd == nil || p.cmd.Process == nil {
        return false
    }
    if p.cmd.ProcessState == nil {
        return true
    }
    return false
}

func (p *process) GetErr() error {
    return p.exitErr
}

func (p *process) GetResult() string {
    if len(p.logWriter.lastLine) == 0 && p.exitErr != nil {
        return p.exitErr.Error()
    }
    return p.logWriter.lastLine
}

func (p *process) GetVersion() string {
    return p.version
}

func (Process) GetAPIPort() int {
    return p.apiPort
}

func (p *Process) GetConfig() *Config {
    return p.config
}

func (p *Process) GetOnlineClients() []string {
    return p.onlineClients
}

func (p *Process) SetOnlineClients(users []string) {
    p.onlineClients = users
}

func (p *process) GetUptime() uint64 {
    return uint64(time.Since(p.startTime).Seconds())
}

func (p *process) refreshAPIPort() {
    for _, inbound := range p.config.InboundConfigs {
        if inbound.Tag == "api" {
            p.apiPort = inbound.Port
            break
        }
    }
}

func (p *process) refreshVersion() {
    cmd := exec.Command(GetBinaryPath(), "-version")
    data, err := cmd.Output()
    if err != nil {
        p.version = "Unknown"
    } else {
        datas := bytes.Split(data, []byte(" "))
        if len(datas) <= 1 {
            p.version = "Unknown"
        } else {
            p.version = string(datas[1])
        }
    }
}

func (p *process) Start() (err error) {
    if p.IsRunning() {
        return errors.New("xray is already running")
    }

    defer func() {
        if err != nil {
            logger.Error("Failure in running xray-core process: ", err)
            p.exitErr = err
        }
    }()

    data, err := json.MarshalIndent(p.config, "", "  ")
    if err != nil {
        return common.NewErrorf("Failed to generate XRAY configuration files: %v", err)
    }

    err = os.MkdirAll(config.GetLogFolder(), 0o755) // ارتقا: 0o755 استانداردتر است
    if err != nil {
        logger.Warningf("Failed to create log folder: %s", err)
    }

    configPath := GetConfigPath()
    err = os.WriteFile(configPath, data, fs.ModePerm)
    if err != nil {
        return common.NewErrorf("Failed to write configuration file: %v", err)
    }

    cmd := exec.Command(GetBinaryPath(), "-c", configPath)
    p.cmd = cmd

    cmd.Stdout = p.logWriter
    cmd.Stderr = p.logWriter

    go func() {
        err := cmd.Run()
        if err != nil {
            logger.Error("Failure in running xray-core:", err)
            p.exitErr = err
        }
    }()

    p.refreshVersion()
    p.refreshAPIPort()

    return nil
}

// --- ارتقا 2: متوقف شدن مطمئن (Graceful Stop with Timeout) ---
func (p *process) Stop() error {
    if !p.IsRunning() {
        return errors.New("xray is not running")
    }

    err := p.cmd.Process.Signal(syscall.SIGTERM)
    if err != nil {
        return err
    }

    // منتظر ماندن برای خروج پروسه با تایم‌اوت ۵ ثانیه
    done := make(chan error, 1)
    go func() {
        done <- p.cmd.Wait()
    }()

    select {
    case <-time.After(5 * time.Second):
        // اگر بعد از ۵ ثانیه متوجه نشد، به زور آن را بکش
        return p.cmd.Process.Kill()
    case err := <-done:
        return err
    }
}

func writeCrachReport(m []byte) error {
    crashReportPath := config.GetBinFolderPath() + "/core_crash_" + time.Now().Format("20060102_150405") + ".log"
    return os.WriteFile(crashReportPath, m, os.ModePerm)
}

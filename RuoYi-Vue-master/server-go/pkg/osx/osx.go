// Package osx 采集服务器运行指标，等价 ruoyi-framework 的
// Server.copyTo()（Server.java:60-80）。
//
// Java 用的是 OSHI（跨平台的系统信息库）+ Runtime 的 JVM 指标；
// Go 侧用 gopsutil 取系统指标，JVM 那部分**没有对应物**，用 Go runtime 的
// 堆指标代替并明确标注 DIFF（详见 Runtime 的注释）。
//
// 字段名叫 runtime 不叫 jvm：这一块装的是 Go 运行时的指标，
// 沿用 Java 的名字会让页面显示成"Java虚拟机信息 / Java名称：Go Runtime"，
// 前后矛盾（前端标签已同步改成 Go 运行时）。
package osx

import (
	"context"
	"fmt"
	"net"
	"os"
	"runtime"
	"time"

	"github.com/shirou/gopsutil/v3/cpu"
	"github.com/shirou/gopsutil/v3/disk"
	"github.com/shirou/gopsutil/v3/host"
	"github.com/shirou/gopsutil/v3/mem"
)

// Server 服务监控总览，等价 Server.java。
//
// JSON 字段名与 src/views/monitor/server/index.vue 的绑定一一对应
// （server.cpu.* / server.mem.* / server.runtime.* / server.sys.* / server.sysFiles）。
//
// DIFF(review): 第三块在 Java 侧叫 jvm，这里改叫 runtime —— 装的是 Go 运行时
// 指标，不是 JVM。前端标签与这个 key 一起改的。
type Server struct {
	Cpu      Cpu       `json:"cpu"`
	Mem      Mem       `json:"mem"`
	Runtime  Runtime   `json:"runtime"`
	Sys      Sys       `json:"sys"`
	SysFiles []SysFile `json:"sysFiles"`
}

// Cpu CPU 指标，等价 Cpu.java。
type Cpu struct {
	CpuNum int     `json:"cpuNum"`
	Total  float64 `json:"total"`
	Sys    float64 `json:"sys"`
	Used   float64 `json:"used"`
	Wait   float64 `json:"wait"`
	Free   float64 `json:"free"`
}

// Mem 内存指标，等价 Mem.java。单位是 GB（Java 的 setMemInfo 全部 /G）。
type Mem struct {
	Total float64 `json:"total"`
	Used  float64 `json:"used"`
	Free  float64 `json:"free"`
	Usage float64 `json:"usage"`
}

// Runtime Go 运行时指标，对应 Java 侧的 Jvm.java。
//
// DIFF(review): Go 没有 JVM。这里用 **Go runtime 的堆指标**代替，语义最接近：
//
//	total <- m.HeapSys：堆向 OS 申请的内存（对齐 JVM 的 totalMemory）
//	used  <- m.HeapAlloc：堆上活跃对象（对齐 JVM 已用）
//	free  <- total - used
//	max   <- 与 total 同值：Go 堆按需增长，**没有 -Xmx 那种固定上限**
//	usage <- used / total
//
// name / version 给的是 Go 运行时而不是 JVM；
// home 是 GOROOT 而不是 JAVA_HOME；startTime 是进程启动时刻。
//
// 类型名与 JSON key 都叫 runtime 而不叫 jvm（见 Server 的注释）。
type Runtime struct {
	Name      string  `json:"name"`
	Version   string  `json:"version"`
	Home      string  `json:"home"`
	StartTime string  `json:"startTime"`
	RunTime   string  `json:"runTime"`
	InputArgs string  `json:"inputArgs"`
	Total     float64 `json:"total"`
	Max       float64 `json:"max"`
	Free      float64 `json:"free"`
	Used      float64 `json:"used"`
	Usage     float64 `json:"usage"`
}

// Sys 系统信息，等价 Sys.java。
type Sys struct {
	ComputerName string `json:"computerName"`
	ComputerIp   string `json:"computerIp"`
	UserDir      string `json:"userDir"`
	OsName       string `json:"osName"`
	OsArch       string `json:"osArch"`
}

// SysFile 磁盘分区，等价 SysFile.java。total/free/used 是**带单位的字符串**。
type SysFile struct {
	DirName     string  `json:"dirName"`
	SysTypeName string  `json:"sysTypeName"`
	TypeName    string  `json:"typeName"`
	Total       string  `json:"total"`
	Free        string  `json:"free"`
	Used        string  `json:"used"`
	Usage       float64 `json:"usage"`
}

// MB / GB 换算基数。Java 的 Server.java 用的是 1024 进制（div(1024*1024*1024)）。
const (
	kb = 1024.0
	mb = kb * 1024
	gb = mb * 1024
)

// 进程启动时刻，用于算运行时长。
var bootTime = time.Now()

// Collect 采集全部指标，等价 Server.copyTo()。
func Collect() (*Server, error) {
	s := &Server{}
	if err := s.setCpuInfo(); err != nil {
		return nil, err
	}
	if err := s.setMemInfo(); err != nil {
		return nil, err
	}
	s.setRuntimeInfo()
	if err := s.setSysInfo(); err != nil {
		return nil, err
	}
	if err := s.setSysFiles(); err != nil {
		return nil, err
	}
	return s, nil
}

// setCpuInfo 等价 Server.setCpuInfo(L127-152)。
//
// Java 先取一次 tick、睡 OSHI_WAIT_SECOND(1000ms)、再取一次，用**两次差值**算
// 这一秒内的占用率。Go 侧同样是两次采样，因此该接口有约 1 秒耗时（与 Java 一致）。
func (s *Server) setCpuInfo() error {
	prev, err := cpu.Times(false)
	if err != nil {
		return err
	}
	time.Sleep(time.Second)
	now, err := cpu.Times(false)
	if err != nil {
		return err
	}
	if len(prev) == 0 || len(now) == 0 {
		return fmt.Errorf("未取到 CPU 采样")
	}
	d := func(get func(cpu.TimesStat) float64) float64 { return get(now[0]) - get(prev[0]) }
	user := d(func(t cpu.TimesStat) float64 { return t.User })
	nice := d(func(t cpu.TimesStat) float64 { return t.Nice })
	sys := d(func(t cpu.TimesStat) float64 { return t.System })
	idle := d(func(t cpu.TimesStat) float64 { return t.Idle })
	iowait := d(func(t cpu.TimesStat) float64 { return t.Iowait })
	irq := d(func(t cpu.TimesStat) float64 { return t.Irq })
	softirq := d(func(t cpu.TimesStat) float64 { return t.Softirq })
	steal := d(func(t cpu.TimesStat) float64 { return t.Steal })

	total := user + nice + sys + idle + iowait + irq + softirq + steal
	s.Cpu.CpuNum = runtime.NumCPU()
	s.Cpu.Total = total // Java 存的是 tick 数（L144），前端不展示该值
	if total > 0 {
		p := func(v float64) float64 { return round2(v / total * 100) }
		s.Cpu.Sys = p(sys)
		s.Cpu.Used = p(user)
		s.Cpu.Wait = p(iowait)
		s.Cpu.Free = p(idle)
	}
	return nil
}

// setMemInfo 等价 Server.setMemInfo(L153-161)：单位是 GB，保留两位。
func (s *Server) setMemInfo() error {
	v, err := mem.VirtualMemory()
	if err != nil {
		return err
	}
	s.Mem.Total = round2(float64(v.Total) / gb)
	s.Mem.Used = round2(float64(v.Used) / gb)
	s.Mem.Free = round2(float64(v.Available) / gb)
	s.Mem.Usage = round2(float64(v.Used) / float64(v.Total) * 100)
	return nil
}

// setRuntimeInfo 等价 Server.setJvmInfo(L176-187)，见 Runtime 的 DIFF 说明。
func (s *Server) setRuntimeInfo() {
	var m runtime.MemStats
	runtime.ReadMemStats(&m)
	total := float64(m.HeapSys) / mb
	used := float64(m.HeapAlloc) / mb
	free := total - used
	if free < 0 {
		free = 0
	}
	usage := 0.0
	if total > 0 {
		usage = used / total * 100
	}
	runTime := time.Since(bootTime)
	s.Runtime = Runtime{
		Name:      "Go Runtime",
		Version:   runtime.Version(),
		Home:      runtime.GOROOT(),
		StartTime: bootTime.Format("2006-01-02 15:04:05"),
		RunTime:   formatDuration(runTime),
		InputArgs: fmt.Sprint(os.Args),
		Total:     round2(total),
		Max:       round2(total), // Go 堆无固定上限，与 total 同值（见 Jvm 注释）
		Free:      round2(free),
		Used:      round2(used),
		Usage:     round2(usage),
	}
}

// setSysInfo 等价 Server.setSysInfo(L163-174)。
func (s *Server) setSysInfo() error {
	hi, err := host.Info()
	if err != nil {
		return err
	}
	dir, err := os.Getwd()
	if err != nil {
		return err
	}
	s.Sys = Sys{
		ComputerName: hi.Hostname,
		ComputerIp:   localIP(),
		UserDir:      dir,
		OsName:       hi.Platform + " " + hi.PlatformVersion,
		OsArch:       runtime.GOARCH,
	}
	return nil
}

// setSysFiles 等价 Server.setSysFiles(L189-210)。
func (s *Server) setSysFiles() error {
	parts, err := disk.PartitionsWithContext(context.Background(), false)
	if err != nil {
		return err
	}
	out := make([]SysFile, 0, len(parts))
	for _, p := range parts {
		u, err := disk.UsageWithContext(context.Background(), p.Mountpoint)
		if err != nil {
			continue // 个别分区（如 /dev、快照卷）取不到就跳过
		}
		used := u.Total - u.Free
		usage := 0.0
		if u.Total > 0 {
			usage = float64(used) / float64(u.Total) * 100
		}
		out = append(out, SysFile{
			DirName:     p.Mountpoint,
			SysTypeName: p.Fstype,
			TypeName:    p.Device,
			Total:       convertFileSize(u.Total),
			Free:        convertFileSize(u.Free),
			Used:        convertFileSize(used),
			Usage:       round2(usage),
		})
	}
	s.SysFiles = out
	return nil
}

// convertFileSize 等价 Server.convertFileSize(L212-228)：转成带单位的字符串。
func convertFileSize(size uint64) string {
	switch {
	case size >= uint64(gb):
		return fmt.Sprintf("%s GB", trimZero(float64(size)/gb))
	case size >= uint64(mb):
		return fmt.Sprintf("%s MB", trimZero(float64(size)/mb))
	case size >= uint64(kb):
		return fmt.Sprintf("%s KB", trimZero(float64(size)/kb))
	}
	return fmt.Sprintf("%d B", size)
}

// formatDuration 把时长格式化成 "x天x小时x分钟"，Java 侧是
// DateUtils.getDatePoor 的输出风格。
func formatDuration(d time.Duration) string {
	sec := int64(d.Seconds())
	days := sec / 86400
	hours := sec % 86400 / 3600
	minutes := sec % 3600 / 60
	return fmt.Sprintf("%d天%d小时%d分钟", days, hours, minutes)
}

func round2(v float64) float64 { return float64(int64(v*100+0.5)) / 100 }

func trimZero(v float64) string {
	s := fmt.Sprintf("%.1f", v)
	for len(s) > 1 && s[len(s)-1] == '0' {
		s = s[:len(s)-1]
	}
	if len(s) > 0 && s[len(s)-1] == '.' {
		s = s[:len(s)-1]
	}
	return s
}

// localIP 取本机对外 IP，取不到回落 127.0.0.1。
func localIP() string {
	addrs, err := net.InterfaceAddrs()
	if err != nil {
		return "127.0.0.1"
	}
	for _, a := range addrs {
		ipnet, ok := a.(*net.IPNet)
		if !ok || ipnet.IP.IsLoopback() {
			continue
		}
		if v := ipnet.IP.To4(); v != nil {
			return v.String()
		}
	}
	return "127.0.0.1"
}

package monitor

import (
	"context"
	"fmt"
	"log"
	"net"
	"os/exec"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/lazyautoops/lazy-auto-ops/plugins/cmdb"
	"gorm.io/gorm"
)

// Collector 监控数据采集器
type Collector struct {
	db         *gorm.DB
	ctx        context.Context
	cancel     context.CancelFunc
	interval   time.Duration
	mu         sync.RWMutex
	metrics    *SystemMetrics
	netMu      sync.Mutex
	netPrevIn  uint64
	netPrevOut uint64
	netPrevAt  time.Time
}

// SystemMetrics 系统指标
type SystemMetrics struct {
	Timestamp time.Time      `json:"timestamp"`
	CPU       CPUMetrics     `json:"cpu"`
	Memory    MemoryMetrics  `json:"memory"`
	Disk      DiskMetrics    `json:"disk"`
	Network   NetworkMetrics `json:"network"`
	Hosts     []HostMetrics  `json:"hosts"`
}

// CPUMetrics CPU指标
type CPUMetrics struct {
	Usage    float64 `json:"usage"`     // CPU使用率 (%)
	Cores    int     `json:"cores"`     // CPU核心数
	LoadAvg1 float64 `json:"load_avg1"` // 1分钟负载
	LoadAvg5 float64 `json:"load_avg5"` // 5分钟负载
}

// MemoryMetrics 内存指标
type MemoryMetrics struct {
	Total     uint64  `json:"total"`      // 总内存 (bytes)
	Used      uint64  `json:"used"`       // 已使用 (bytes)
	Free      uint64  `json:"free"`       // 空闲 (bytes)
	Usage     float64 `json:"usage"`      // 使用率 (%)
	SwapTotal uint64  `json:"swap_total"` // 交换分区总量
	SwapUsed  uint64  `json:"swap_used"`  // 交换分区使用
}

// DiskMetrics 磁盘指标
type DiskMetrics struct {
	Total     uint64  `json:"total"`      // 总容量 (bytes)
	Used      uint64  `json:"used"`       // 已使用 (bytes)
	Free      uint64  `json:"free"`       // 空闲 (bytes)
	Usage     float64 `json:"usage"`      // 使用率 (%)
	ReadRate  uint64  `json:"read_rate"`  // 读取速率 (bytes/s)
	WriteRate uint64  `json:"write_rate"` // 写入速率 (bytes/s)
}

// NetworkMetrics 网络指标
type NetworkMetrics struct {
	InboundRate   uint64 `json:"inbound_rate"`   // 入站速率 (bytes/s)
	OutboundRate  uint64 `json:"outbound_rate"`  // 出站速率 (bytes/s)
	InboundTotal  uint64 `json:"inbound_total"`  // 入站总量 (bytes)
	OutboundTotal uint64 `json:"outbound_total"` // 出站总量 (bytes)
	Connections   int    `json:"connections"`    // 连接数
}

// HostMetrics 主机指标
type HostMetrics struct {
	HostID   string    `json:"host_id"`
	Hostname string    `json:"hostname"`
	IP       string    `json:"ip"`
	Status   string    `json:"status"` // online, offline, warning
	CPU      float64   `json:"cpu"`
	Memory   float64   `json:"memory"`
	Disk     float64   `json:"disk"`
	Uptime   string    `json:"uptime"`
	LastSeen time.Time `json:"last_seen"`
}

// NewCollector 创建采集器
func NewCollector(db *gorm.DB, interval time.Duration) *Collector {
	ctx, cancel := context.WithCancel(context.Background())
	return &Collector{
		db:       db,
		ctx:      ctx,
		cancel:   cancel,
		interval: interval,
		metrics:  &SystemMetrics{},
	}
}

// Start 启动采集器
func (c *Collector) Start() error {
	log.Println("[Monitor Collector] Starting...")

	// 立即采集一次
	c.collect()

	// 启动定期采集
	go c.collectLoop()

	log.Println("[Monitor Collector] Started successfully")
	return nil
}

// Stop 停止采集器
func (c *Collector) Stop() error {
	log.Println("[Monitor Collector] Stopping...")
	c.cancel()
	log.Println("[Monitor Collector] Stopped")
	return nil
}

// GetMetrics 获取当前指标
func (c *Collector) GetMetrics() *SystemMetrics {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.metrics
}

// collectLoop 采集循环
func (c *Collector) collectLoop() {
	ticker := time.NewTicker(c.interval)
	defer ticker.Stop()

	for {
		select {
		case <-c.ctx.Done():
			return
		case <-ticker.C:
			c.collect()
		}
	}
}

// collect 执行采集
func (c *Collector) collect() {
	metrics := &SystemMetrics{
		Timestamp: time.Now(),
	}

	// 并发采集各项指标
	var wg sync.WaitGroup

	wg.Add(1)
	go func() {
		defer wg.Done()
		metrics.CPU = c.collectCPU()
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()
		metrics.Memory = c.collectMemory()
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()
		metrics.Disk = c.collectDisk()
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()
		metrics.Network = c.collectNetwork()
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()
		metrics.Hosts = c.collectHosts()
	}()

	wg.Wait()

	// 更新指标
	c.mu.Lock()
	c.metrics = metrics
	c.mu.Unlock()

	// 保存到数据库（可选）
	c.saveMetrics(metrics)
}

// collectCPU 采集CPU指标
func (c *Collector) collectCPU() CPUMetrics {
	metrics := CPUMetrics{
		Cores: runtime.NumCPU(),
	}

	switch runtime.GOOS {
	case "linux":
		// Linux: 读取 /proc/stat 和 /proc/loadavg
		if usage, err := c.getCPUUsageLinux(); err == nil {
			metrics.Usage = usage
		}
		if load1, load5, err := c.getLoadAvgLinux(); err == nil {
			metrics.LoadAvg1 = load1
			metrics.LoadAvg5 = load5
		}
	case "darwin":
		// macOS: 使用 top 命令
		if usage, err := c.getCPUUsageDarwin(); err == nil {
			metrics.Usage = usage
		}
		if load1, load5, err := c.getLoadAvgDarwin(); err == nil {
			metrics.LoadAvg1 = load1
			metrics.LoadAvg5 = load5
		}
	default:
		// 其他系统返回模拟数据
		metrics.Usage = 45.0
		metrics.LoadAvg1 = 1.5
		metrics.LoadAvg5 = 1.2
	}

	return metrics
}

// collectMemory 采集内存指标
func (c *Collector) collectMemory() MemoryMetrics {
	metrics := MemoryMetrics{}

	switch runtime.GOOS {
	case "linux":
		if mem, err := c.getMemoryLinux(); err == nil {
			metrics = mem
		}
	case "darwin":
		if mem, err := c.getMemoryDarwin(); err == nil {
			metrics = mem
		}
	default:
		// 模拟数据
		metrics.Total = 16 * 1024 * 1024 * 1024 // 16GB
		metrics.Used = 12 * 1024 * 1024 * 1024  // 12GB
		metrics.Free = 4 * 1024 * 1024 * 1024   // 4GB
		metrics.Usage = 75.0
	}

	return metrics
}

// collectDisk 采集磁盘指标
func (c *Collector) collectDisk() DiskMetrics {
	metrics := DiskMetrics{}

	switch runtime.GOOS {
	case "linux", "darwin":
		if disk, err := c.getDiskUsage("/"); err == nil {
			metrics = disk
		}
	default:
		// 模拟数据
		metrics.Total = 500 * 1024 * 1024 * 1024 // 500GB
		metrics.Used = 225 * 1024 * 1024 * 1024  // 225GB
		metrics.Free = 275 * 1024 * 1024 * 1024  // 275GB
		metrics.Usage = 45.0
	}

	return metrics
}

// collectNetwork 采集网络指标
func (c *Collector) collectNetwork() NetworkMetrics {
	metrics := NetworkMetrics{}

	switch runtime.GOOS {
	case "linux":
		if net, err := c.getNetworkLinux(); err == nil {
			metrics = net
		}
	case "darwin":
		if net, err := c.getNetworkDarwin(); err == nil {
			metrics = net
		}
	default:
		// 模拟数据
		metrics.InboundRate = 125 * 1024 * 1024 // 125 MB/s
		metrics.OutboundRate = 80 * 1024 * 1024 // 80 MB/s
		metrics.Connections = 150
	}

	return metrics
}

// collectHosts 采集主机指标
func (c *Collector) collectHosts() []HostMetrics {
	var hosts []cmdb.Host
	if err := c.db.Find(&hosts).Error; err != nil {
		return []HostMetrics{}
	}

	var metrics []HostMetrics
	var wg sync.WaitGroup
	var mu sync.Mutex

	for _, host := range hosts {
		wg.Add(1)
		go func(h cmdb.Host) {
			defer wg.Done()

			// 探测主机是否在线 (TCP Ping)
			start := time.Now()
			status := "offline"
			port := h.Port
			if port == 0 {
				port = 22
			}

			conn, err := net.DialTimeout("tcp", fmt.Sprintf("%s:%d", h.IP, port), 2*time.Second)
			if err == nil {
				conn.Close()
				status = "online"
			}

			// 更新 CMDB 状态
			newStatusInt := 0
			if status == "online" {
				newStatusInt = 1
			}
			if h.Status != newStatusInt {
				c.db.Model(&h).Update("status", newStatusInt)
			}

			mu.Lock()
			metrics = append(metrics, HostMetrics{
				HostID:   h.ID,
				Hostname: h.Name,
				IP:       h.IP,
				Status:   status,
				Uptime:   time.Since(start).String(), // 这里暂时用探测耗时代替uptime
				LastSeen: time.Now(),
			})
			mu.Unlock()
		}(host)
	}

	wg.Wait()
	return metrics
}

// getCPUUsageLinux Linux系统CPU使用率
func (c *Collector) getCPUUsageLinux() (float64, error) {
	cmd := exec.Command("sh", "-c", "top -bn1 | grep 'Cpu(s)' | awk '{print $2}' | cut -d'%' -f1")
	output, err := cmd.Output()
	if err != nil {
		return 0, err
	}

	usage, err := strconv.ParseFloat(strings.TrimSpace(string(output)), 64)
	if err != nil {
		return 0, err
	}

	return usage, nil
}

// getCPUUsageDarwin macOS系统CPU使用率
func (c *Collector) getCPUUsageDarwin() (float64, error) {
	cmd := exec.Command("sh", "-c", "ps -A -o %cpu | awk '{s+=$1} END {print s}'")
	output, err := cmd.Output()
	if err != nil {
		return 0, err
	}

	usage, err := strconv.ParseFloat(strings.TrimSpace(string(output)), 64)
	if err != nil {
		return 0, err
	}

	return usage, nil
}

// getLoadAvgLinux Linux系统负载
func (c *Collector) getLoadAvgLinux() (float64, float64, error) {
	cmd := exec.Command("cat", "/proc/loadavg")
	output, err := cmd.Output()
	if err != nil {
		return 0, 0, err
	}

	fields := strings.Fields(string(output))
	if len(fields) < 2 {
		return 0, 0, fmt.Errorf("invalid loadavg format")
	}

	load1, _ := strconv.ParseFloat(fields[0], 64)
	load5, _ := strconv.ParseFloat(fields[1], 64)

	return load1, load5, nil
}

// getLoadAvgDarwin macOS系统负载
func (c *Collector) getLoadAvgDarwin() (float64, float64, error) {
	cmd := exec.Command("sysctl", "-n", "vm.loadavg")
	output, err := cmd.Output()
	if err != nil {
		return 0, 0, err
	}

	// 输出格式: { 1.5 1.2 1.0 }
	str := strings.Trim(string(output), "{ }\n")
	fields := strings.Fields(str)
	if len(fields) < 2 {
		return 0, 0, fmt.Errorf("invalid loadavg format")
	}

	load1, _ := strconv.ParseFloat(fields[0], 64)
	load5, _ := strconv.ParseFloat(fields[1], 64)

	return load1, load5, nil
}

// getMemoryLinux Linux系统内存
func (c *Collector) getMemoryLinux() (MemoryMetrics, error) {
	cmd := exec.Command("free", "-b")
	output, err := cmd.Output()
	if err != nil {
		return MemoryMetrics{}, err
	}

	lines := strings.Split(string(output), "\n")
	if len(lines) < 2 {
		return MemoryMetrics{}, fmt.Errorf("invalid free output")
	}

	fields := strings.Fields(lines[1])
	if len(fields) < 3 {
		return MemoryMetrics{}, fmt.Errorf("invalid free format")
	}

	total, _ := strconv.ParseUint(fields[1], 10, 64)
	used, _ := strconv.ParseUint(fields[2], 10, 64)
	free := total - used
	usage := float64(used) / float64(total) * 100

	return MemoryMetrics{
		Total: total,
		Used:  used,
		Free:  free,
		Usage: usage,
	}, nil
}

// getMemoryDarwin macOS系统内存
func (c *Collector) getMemoryDarwin() (MemoryMetrics, error) {
	// 获取总内存
	cmd := exec.Command("sysctl", "-n", "hw.memsize")
	output, err := cmd.Output()
	if err != nil {
		return MemoryMetrics{}, err
	}

	total, _ := strconv.ParseUint(strings.TrimSpace(string(output)), 10, 64)

	// 获取已使用内存（简化计算）
	cmd = exec.Command("sh", "-c", "vm_stat | grep 'Pages active' | awk '{print $3}' | tr -d '.'")
	output, err = cmd.Output()
	if err != nil {
		return MemoryMetrics{}, err
	}

	activePages, _ := strconv.ParseUint(strings.TrimSpace(string(output)), 10, 64)
	used := activePages * 4096 // 页面大小通常是4KB
	free := total - used
	usage := float64(used) / float64(total) * 100

	return MemoryMetrics{
		Total: total,
		Used:  used,
		Free:  free,
		Usage: usage,
	}, nil
}

// getDiskUsage 磁盘使用情况
func (c *Collector) getDiskUsage(path string) (DiskMetrics, error) {
	cmd := exec.Command("df", "-B1", path)
	output, err := cmd.Output()
	if err != nil {
		return DiskMetrics{}, err
	}

	lines := strings.Split(string(output), "\n")
	if len(lines) < 2 {
		return DiskMetrics{}, fmt.Errorf("invalid df output")
	}

	fields := strings.Fields(lines[1])
	if len(fields) < 5 {
		return DiskMetrics{}, fmt.Errorf("invalid df format")
	}

	total, _ := strconv.ParseUint(fields[1], 10, 64)
	used, _ := strconv.ParseUint(fields[2], 10, 64)
	free, _ := strconv.ParseUint(fields[3], 10, 64)
	usageStr := strings.TrimSuffix(fields[4], "%")
	usage, _ := strconv.ParseFloat(usageStr, 64)

	return DiskMetrics{
		Total: total,
		Used:  used,
		Free:  free,
		Usage: usage,
	}, nil
}

// getNetworkLinux Linux网络统计
func (c *Collector) getNetworkLinux() (NetworkMetrics, error) {
	cmd := exec.Command("cat", "/proc/net/dev")
	output, err := cmd.Output()
	if err != nil {
		return NetworkMetrics{}, err
	}

	lines := strings.Split(string(output), "\n")
	var totalIn, totalOut uint64

	for _, line := range lines {
		if strings.Contains(line, ":") {
			fields := strings.Fields(strings.Split(line, ":")[1])
			if len(fields) >= 9 {
				in, _ := strconv.ParseUint(fields[0], 10, 64)
				out, _ := strconv.ParseUint(fields[8], 10, 64)
				totalIn += in
				totalOut += out
			}
		}
	}

	// 使用上一次采样点计算吞吐速率，避免固定值导致面板始终不变
	now := time.Now()
	var inboundRate uint64
	var outboundRate uint64
	c.netMu.Lock()
	if !c.netPrevAt.IsZero() {
		elapsed := now.Sub(c.netPrevAt).Seconds()
		if elapsed > 0 {
			if totalIn >= c.netPrevIn {
				inboundRate = uint64(float64(totalIn-c.netPrevIn) / elapsed)
			}
			if totalOut >= c.netPrevOut {
				outboundRate = uint64(float64(totalOut-c.netPrevOut) / elapsed)
			}
		}
	}
	c.netPrevIn = totalIn
	c.netPrevOut = totalOut
	c.netPrevAt = now
	c.netMu.Unlock()

	return NetworkMetrics{
		InboundTotal:  totalIn,
		OutboundTotal: totalOut,
		InboundRate:   inboundRate,
		OutboundRate:  outboundRate,
		Connections:   c.countConnections(),
	}, nil
}

func (c *Collector) countConnections() int {
	cmd := exec.Command("sh", "-c", "netstat -ant | wc -l")
	output, _ := cmd.Output()
	count, _ := strconv.Atoi(strings.TrimSpace(string(output)))
	return count
}

// getNetworkDarwin macOS网络统计
func (c *Collector) getNetworkDarwin() (NetworkMetrics, error) {
	// 简化实现，返回模拟数据
	return NetworkMetrics{
		InboundRate:  125 * 1024 * 1024,
		OutboundRate: 80 * 1024 * 1024,
		Connections:  150,
	}, nil
}

// saveMetrics 保存指标到数据库
func (c *Collector) saveMetrics(metrics *SystemMetrics) {
	// 创建指标记录
	record := MetricRecord{
		Timestamp:   metrics.Timestamp,
		CPUUsage:    metrics.CPU.Usage,
		MemoryUsage: metrics.Memory.Usage,
		DiskUsage:   metrics.Disk.Usage,
		NetworkIn:   metrics.Network.InboundRate,
		NetworkOut:  metrics.Network.OutboundRate,
	}

	// 保存到数据库
	if err := c.db.Create(&record).Error; err != nil {
		log.Printf("[Monitor Collector] Failed to save metrics: %v", err)
	}

	// 清理旧数据（保留最近24小时）
	cutoff := time.Now().Add(-24 * time.Hour)
	c.db.Where("timestamp < ?", cutoff).Delete(&MetricRecord{})
}

package knowledge

import (
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

// ListSources 获取数据源列表
func (h *KnowledgeHandler) ListSources(c *gin.Context) {
	var sources []SyncSource
	if err := h.db.Order("created_at desc").Find(&sources).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "获取同步源列表失败"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "data": sources})
}

// CreateSource 创建数据源
func (h *KnowledgeHandler) CreateSource(c *gin.Context) {
	var source SyncSource
	if err := c.ShouldBindJSON(&source); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "参数解析失败"})
		return
	}

	source.ID = uuid.New().String()
	source.SyncStatus = "idle"
	source.CreatedAt = time.Now()
	source.UpdatedAt = time.Now()

	if err := h.db.Create(&source).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "创建同步源失败"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "data": source})
}

// DeleteSource 删除数据源
func (h *KnowledgeHandler) DeleteSource(c *gin.Context) {
	id := c.Param("id")
	if err := h.db.Where("id = ?", id).Delete(&SyncSource{}).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "删除失败"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "删除成功"})
}

// SyncSource 一键同步触发
func (h *KnowledgeHandler) SyncSource(c *gin.Context) {
	id := c.Param("id")
	var source SyncSource
	if err := h.db.First(&source, "id = ?", id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"code": 404, "message": "数据源不存在"})
		return
	}

	// 1. 设置状态为 syncing
	h.db.Model(&source).Updates(map[string]interface{}{
		"sync_status": "syncing",
		"updated_at":  time.Now(),
	})

	// 2. 异步或同步执行拉取 (此处由于需要快速响应前端，且演示效果直接，我们直接同步执行，给用户立竿见影的刷新感)
	go func(src SyncSource) {
		err := h.executeSync(&src)
		status := "success"
		now := time.Now()
		if err != nil {
			status = "failed"
		}

		h.db.Model(&SyncSource{}).Where("id = ?", src.ID).Updates(map[string]interface{}{
			"sync_status":     status,
			"last_synced_at":  &now,
			"updated_at":      time.Now(),
		})
	}(source)

	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "已在后台启动知识库同步"})
}

// executeSync 核心同步逻辑
func (h *KnowledgeHandler) executeSync(src *SyncSource) error {
	// 如果是模拟环境或者配置为 mock，自动导入离线的高清 Wiki 排障文档
	isMock := src.URL == "mock" || strings.Contains(strings.ToLower(src.URL), "mock") || strings.Contains(strings.ToLower(src.URL), "localhost")

	if isMock {
		time.Sleep(2 * time.Second) // 模拟网络拉取延时

		mockDocs := []Document{
			{
				Title:     fmt.Sprintf("[%s同步] Nginx 出现 502 Bad Gateway 报错排查与紧急处理流程", strings.ToUpper(src.Type)),
				Category:  "runbook",
				Tags:      "nginx,gateway,emergency",
				CreatedBy: "system_sync",
				Content: `# Nginx 502 Bad Gateway 紧急处置流程

### 1. 现象描述
当外部用户请求服务时，浏览器报错 ` + "`502 Bad Gateway`" + `。

### 2. 排查步骤
1. **检查后端 upstream 应用状态**：
   通过 SSH 登录后端主机，执行 ` + "`ps aux | grep java`" + ` 或 ` + "`docker ps`" + `，确认后端微服务端口是否在正常监听。
2. **检查 PHP-FPM / uWSGI 等网关进程**：
   如果是动态站点，检查网关服务是否挂掉：` + "`systemctl status php-fpm`" + `。
3. **查看 Nginx 错误日志**：
   执行 ` + "`tail -n 100 /var/log/nginx/error.log`" + `。若日志中包含 ` + "`connect() failed (111: Connection refused) while connecting to upstream`" + `，代表后端端口确实未开启或防火墙阻断。

### 3. 解决方案
* 重启后端服务容器或裸进程；
* 检查本机及安全组防火墙设置，放行对应 upstream 端口。`,
			},
			{
				Title:     fmt.Sprintf("[%s同步] JVM 内存溢出 OOM (OutOfMemoryError) 故障深度复盘报告", strings.ToUpper(src.Type)),
				Category:  "post-mortem",
				Tags:      "jvm,oom,java",
				CreatedBy: "system_sync",
				Content: `# JVM 核心内存溢出 OOM 故障复盘

### 1. 故障根因
生产环境在凌晨 02:15 收到堆内存溢出告警，进程由于 OOM 被系统 OOM-killer 强制杀死。
经过 Heap Dump 分析，发现由于 SQL 查询未加 Limit 限制，全表扫描加载了 150 万条数据至 JVM 堆内，导致瞬间塞满。

### 2. 改进措施
* 所有的 SQL 分页查询必须在框架层强制追加硬 limit（最大限制 5000 条）；
* 调整 JVM 启动参数，添加 ` + "`-XX:+HeapDumpOnOutOfMemoryError -XX:HeapDumpPath=/app/dumps/`" + ` 选项以便后续排查。`,
			},
			{
				Title:     fmt.Sprintf("[%s同步] Redis 主从切换延迟与缓存雪崩应急应对指南", strings.ToUpper(src.Type)),
				Category:  "guide",
				Tags:      "redis,cache,cluster",
				CreatedBy: "system_sync",
				Content: `# Redis 缓存雪崩与主从切换指南

### 1. 问题防范
为了防止因大批量缓存在同一时间段过期导致缓存雪崩：
* 设置随机过期抖动值：` + "`expire_time = base_expire + random.randint(1, 300)`" + `；
* 核心高频访问接口必须开启二级本地缓存（如 Caffeine）。

### 2. 应急演练步骤
若出现缓存大面积失效，直接通过 Sentinel 或 Redis Cluster 进行从节点紧急升级，并暂时开启限流熔断以保护下游 MySQL 数据库。`,
			},
		}

		// 检查防重，若有相同标题的文档，就执行覆盖更新，否则直接创建
		for _, md := range mockDocs {
			var oldDoc Document
			err := h.db.Where("title = ?", md.Title).First(&oldDoc).Error
			if err == nil {
				h.db.Model(&oldDoc).Updates(map[string]interface{}{
					"content":    md.Content,
					"tags":       md.Tags,
					"category":   md.Category,
					"updated_at": time.Now(),
				})
			} else if errors.Is(err, gorm.ErrRecordNotFound) {
				md.CreatedAt = time.Now()
				md.UpdatedAt = time.Now()
				h.db.Create(&md)
			}
		}
		return nil
	}

	// 在线逻辑（在此仅作标准的 HTTP 调用框架演示，以防在实际接入真实 Confluence 时使用）
	client := &http.Client{Timeout: 10 * time.Second}
	req, err := http.NewRequest("GET", src.URL+"/rest/api/content?limit=5", nil)
	if err != nil {
		return err
	}
	if src.Token != "" {
		req.Header.Set("Authorization", "Bearer "+src.Token)
	}

	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("HTTP request failed with status code: %d", resp.StatusCode)
	}

	return nil
}

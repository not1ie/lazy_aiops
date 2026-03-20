package domain

import (
	"crypto/tls"
	"encoding/json"
	"fmt"
	"math"
	"net"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type DomainHandler struct {
	db *gorm.DB
}

func NewDomainHandler(db *gorm.DB) *DomainHandler {
	return &DomainHandler{db: db}
}

type domainRuntimeResult struct {
	Domain          string                 `json:"domain"`
	DNSResolved     bool                   `json:"dns_resolved"`
	IPs             []string               `json:"ips"`
	HTTPStatusCode  int                    `json:"http_status_code"`
	ResponseTimeMS  int                    `json:"response_time_ms"`
	HealthStatus    string                 `json:"health_status"`
	SSLDaysToExpire int                    `json:"ssl_days_to_expire"`
	SSL             map[string]interface{} `json:"ssl,omitempty"`
	Error           string                 `json:"error,omitempty"`
	CheckedAt       time.Time              `json:"checked_at"`
}

// ListAccounts 云账号列表
func (h *DomainHandler) ListAccounts(c *gin.Context) {
	var accounts []CloudAccount
	if err := h.db.Find(&accounts).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "data": accounts})
}

// CreateAccount 创建云账号
func (h *DomainHandler) CreateAccount(c *gin.Context) {
	var account CloudAccount
	if err := c.ShouldBindJSON(&account); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "参数错误"})
		return
	}
	if err := h.db.Create(&account).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "data": account})
}

// DeleteAccount 删除云账号
func (h *DomainHandler) DeleteAccount(c *gin.Context) {
	id := c.Param("id")
	if err := h.db.Delete(&CloudAccount{}, "id = ?", id).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "删除成功"})
}

// SyncDomains 同步域名
func (h *DomainHandler) SyncDomains(c *gin.Context) {
	id := c.Param("id")
	var account CloudAccount
	if err := h.db.First(&account, "id = ?", id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"code": 404, "message": "账号不存在"})
		return
	}

	// 根据云厂商调用不同的API
	var domains []CloudDomain
	var err error

	switch account.Provider {
	case "aliyun":
		domains, err = h.syncAliyunDomains(&account)
	case "tencent":
		domains, err = h.syncTencentDomains(&account)
	default:
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "暂不支持该云厂商"})
		return
	}

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "同步失败: " + err.Error()})
		return
	}

	// 更新数据库
	for _, d := range domains {
		d.AccountID = account.ID
		d.Provider = account.Provider
		if d.ExpirationAt != nil {
			d.DaysToExpire = int(time.Until(*d.ExpirationAt).Hours() / 24)
		}

		var existing CloudDomain
		if err := h.db.Where("account_id = ? AND domain = ?", account.ID, d.Domain).First(&existing).Error; err == nil {
			d.ID = existing.ID
			h.db.Save(&d)
		} else {
			h.db.Create(&d)
		}
	}

	// 更新账号
	now := time.Now()
	h.db.Model(&account).Updates(map[string]interface{}{
		"last_sync_at": now,
		"domain_count": len(domains),
	})

	c.JSON(http.StatusOK, gin.H{"code": 0, "data": gin.H{"count": len(domains)}, "message": "同步成功"})
}

// ListDomains 域名列表
func (h *DomainHandler) ListDomains(c *gin.Context) {
	var domains []CloudDomain
	query := h.db.Preload("Account")

	if accountID := c.Query("account_id"); accountID != "" {
		query = query.Where("account_id = ?", accountID)
	}
	if keyword := c.Query("keyword"); keyword != "" {
		query = query.Where("domain LIKE ?", "%"+keyword+"%")
	}

	if err := query.Find(&domains).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": err.Error()})
		return
	}
	for i := range domains {
		h.refreshDomainRuntimeFields(&domains[i])
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "data": domains})
}

// ListExpiringDomains 即将过期的域名
func (h *DomainHandler) ListExpiringDomains(c *gin.Context) {
	days := 30 // 默认30天内过期
	if d := c.Query("days"); d != "" {
		fmt.Sscanf(d, "%d", &days)
	}

	var domains []CloudDomain
	now := time.Now()
	cutoff := now.Add(time.Duration(days) * 24 * time.Hour)
	if err := h.db.Preload("Account").Where("expiration_at IS NOT NULL AND expiration_at > ? AND expiration_at <= ?", now, cutoff).Order("expiration_at ASC").Find(&domains).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": err.Error()})
		return
	}
	for i := range domains {
		h.refreshDomainRuntimeFields(&domains[i])
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "data": domains})
}

// CheckDomain 检查域名信息
func (h *DomainHandler) CheckDomain(c *gin.Context) {
	var req struct {
		Domain string `json:"domain" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "参数错误"})
		return
	}
	result, err := h.inspectDomainRuntime(req.Domain)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": err.Error()})
		return
	}
	h.updateDomainRuntimeByDomain(req.Domain, result)
	c.JSON(http.StatusOK, gin.H{"code": 0, "data": result})
}

// CheckAllDomains 批量体检
func (h *DomainHandler) CheckAllDomains(c *gin.Context) {
	var domains []CloudDomain
	if err := h.db.Select("id", "domain").Where("domain <> ''").Find(&domains).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": err.Error()})
		return
	}
	unique := make(map[string]struct{})
	checked := 0
	success := 0
	failed := 0
	for i := range domains {
		domain := strings.TrimSpace(domains[i].Domain)
		if domain == "" {
			continue
		}
		if _, ok := unique[domain]; ok {
			continue
		}
		unique[domain] = struct{}{}
		checked++
		result, err := h.inspectDomainRuntime(domain)
		if err != nil {
			failed++
			continue
		}
		h.updateDomainRuntimeByDomain(domain, result)
		success++
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "data": gin.H{
		"total":   checked,
		"success": success,
		"failed":  failed,
	}})
}

// ListCerts SSL证书列表
func (h *DomainHandler) ListCerts(c *gin.Context) {
	var certs []SSLCertificate
	if err := h.db.Order("CASE WHEN not_after IS NULL THEN 1 ELSE 0 END, not_after ASC").Find(&certs).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": err.Error()})
		return
	}
	for i := range certs {
		h.refreshCertRuntimeFields(&certs[i])
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "data": certs})
}

// CreateCert 添加SSL证书监控
func (h *DomainHandler) CreateCert(c *gin.Context) {
	var req struct {
		Domain string `json:"domain" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "参数错误"})
		return
	}

	// 检查证书
	certInfo, err := h.checkSSL(req.Domain)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "获取证书失败: " + err.Error()})
		return
	}

	cert := SSLCertificate{
		Domain:       req.Domain,
		Issuer:       certInfo["issuer"].(string),
		Subject:      certInfo["subject"].(string),
		SANs:         certInfo["sans"].(string),
		NotBefore:    certInfo["not_before"].(*time.Time),
		NotAfter:     certInfo["not_after"].(*time.Time),
		DaysToExpire: certInfo["days_to_expire"].(int),
		SerialNumber: certInfo["serial_number"].(string),
		Status:       1,
	}

	if cert.DaysToExpire <= 0 {
		cert.Status = 0 // 已过期
	} else if cert.DaysToExpire <= 30 {
		cert.Status = 2 // 即将过期
	}

	now := time.Now()
	cert.LastCheckAt = &now

	if err := h.db.Create(&cert).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"code": 0, "data": cert})
}

// DeleteCert 删除证书监控
func (h *DomainHandler) DeleteCert(c *gin.Context) {
	id := c.Param("id")
	if err := h.db.Delete(&SSLCertificate{}, "id = ?", id).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "删除成功"})
}

// CheckCert 检查证书
func (h *DomainHandler) CheckCert(c *gin.Context) {
	id := c.Param("id")
	var cert SSLCertificate
	if err := h.db.First(&cert, "id = ?", id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"code": 404, "message": "证书不存在"})
		return
	}

	certInfo, err := h.checkSSL(cert.Domain)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "检查失败: " + err.Error()})
		return
	}

	// 更新证书信息
	now := time.Now()
	updates := map[string]interface{}{
		"issuer":         certInfo["issuer"],
		"subject":        certInfo["subject"],
		"sans":           certInfo["sans"],
		"not_before":     certInfo["not_before"],
		"not_after":      certInfo["not_after"],
		"days_to_expire": certInfo["days_to_expire"],
		"serial_number":  certInfo["serial_number"],
		"last_check_at":  now,
		"status":         1,
	}

	daysToExpire := certInfo["days_to_expire"].(int)
	if daysToExpire <= 0 {
		updates["status"] = 0
	} else if daysToExpire <= 30 {
		updates["status"] = 2
	}

	if err := h.db.Model(&cert).Updates(updates).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": err.Error()})
		return
	}

	resp := gin.H{
		"id":             cert.ID,
		"domain":         cert.Domain,
		"issuer":         certInfo["issuer"],
		"subject":        certInfo["subject"],
		"sans":           certInfo["sans"],
		"not_before":     certInfo["not_before"],
		"not_after":      certInfo["not_after"],
		"days_to_expire": daysToExpire,
		"serial_number":  certInfo["serial_number"],
		"status":         updates["status"],
		"last_check_at":  now,
		"checked_at":     now,
	}

	c.JSON(http.StatusOK, gin.H{"code": 0, "data": resp})
}

// CheckAllCerts 批量检查证书
func (h *DomainHandler) CheckAllCerts(c *gin.Context) {
	var certs []SSLCertificate
	if err := h.db.Find(&certs).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": err.Error()})
		return
	}

	success := 0
	failed := 0
	now := time.Now()
	for _, cert := range certs {
		certInfo, err := h.checkSSL(cert.Domain)
		if err != nil {
			failed++
			continue
		}
		updates := map[string]interface{}{
			"issuer":         certInfo["issuer"],
			"subject":        certInfo["subject"],
			"sans":           certInfo["sans"],
			"not_before":     certInfo["not_before"],
			"not_after":      certInfo["not_after"],
			"days_to_expire": certInfo["days_to_expire"],
			"serial_number":  certInfo["serial_number"],
			"last_check_at":  now,
			"status":         1,
		}

		daysToExpire := certInfo["days_to_expire"].(int)
		if daysToExpire <= 0 {
			updates["status"] = 0
		} else if daysToExpire <= 30 {
			updates["status"] = 2
		}

		if err := h.db.Model(&cert).Updates(updates).Error; err != nil {
			failed++
			continue
		}
		success++
	}

	c.JSON(http.StatusOK, gin.H{"code": 0, "data": gin.H{
		"total":   len(certs),
		"success": success,
		"failed":  failed,
	}})
}

// ListExpiringCerts 即将过期的证书
func (h *DomainHandler) ListExpiringCerts(c *gin.Context) {
	days := 30
	if d := c.Query("days"); d != "" {
		fmt.Sscanf(d, "%d", &days)
	}

	var certs []SSLCertificate
	now := time.Now()
	cutoff := now.Add(time.Duration(days) * 24 * time.Hour)
	if err := h.db.Where("not_after IS NOT NULL AND not_after > ? AND not_after <= ?", now, cutoff).Order("not_after ASC").Find(&certs).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": err.Error()})
		return
	}
	for i := range certs {
		h.refreshCertRuntimeFields(&certs[i])
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "data": certs})
}

func (h *DomainHandler) inspectDomainRuntime(domain string) (*domainRuntimeResult, error) {
	domain = strings.TrimSpace(domain)
	if domain == "" {
		return nil, fmt.Errorf("域名不能为空")
	}
	result := &domainRuntimeResult{
		Domain:       domain,
		HealthStatus: "unknown",
		CheckedAt:    time.Now(),
	}

	ips, dnsErr := net.LookupIP(domain)
	if dnsErr == nil && len(ips) > 0 {
		result.DNSResolved = true
		ipList := make([]string, 0, len(ips))
		for _, ip := range ips {
			ipList = append(ipList, ip.String())
		}
		result.IPs = ipList
	}

	statusCode, latencyMS, httpErr := h.probeHTTP(domain)
	if httpErr == nil {
		result.HTTPStatusCode = statusCode
		result.ResponseTimeMS = latencyMS
	}

	certInfo, sslErr := h.checkSSL(domain)
	if sslErr == nil {
		result.SSL = certInfo
		if days, ok := certInfo["days_to_expire"].(int); ok {
			result.SSLDaysToExpire = days
		}
	}

	result.HealthStatus = deriveHealthStatus(result, dnsErr, httpErr, sslErr)
	if dnsErr != nil {
		result.Error = dnsErr.Error()
	} else if httpErr != nil {
		result.Error = httpErr.Error()
	} else if sslErr != nil {
		result.Error = sslErr.Error()
	}
	return result, nil
}

func (h *DomainHandler) probeHTTP(domain string) (int, int, error) {
	client := &http.Client{
		Timeout: 8 * time.Second,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			if len(via) >= 5 {
				return fmt.Errorf("redirect loop")
			}
			return nil
		},
	}
	targets := []string{"https://" + domain, "http://" + domain}
	var lastErr error
	for _, target := range targets {
		req, _ := http.NewRequest(http.MethodGet, target, nil)
		req.Header.Set("User-Agent", "LazyAutoOps-DomainCheck/1.0")
		start := time.Now()
		resp, err := client.Do(req)
		if err != nil {
			lastErr = err
			continue
		}
		_ = resp.Body.Close()
		return resp.StatusCode, int(time.Since(start).Milliseconds()), nil
	}
	if lastErr == nil {
		lastErr = fmt.Errorf("http probe failed")
	}
	return 0, 0, lastErr
}

func deriveHealthStatus(result *domainRuntimeResult, dnsErr, httpErr, sslErr error) string {
	if result == nil {
		return "unknown"
	}
	if dnsErr != nil || !result.DNSResolved {
		return "critical"
	}
	if httpErr != nil || result.HTTPStatusCode >= 500 || result.HTTPStatusCode == 0 {
		return "critical"
	}
	if result.HTTPStatusCode >= 400 {
		return "warning"
	}
	if sslErr != nil {
		return "warning"
	}
	if result.SSLDaysToExpire > 0 && result.SSLDaysToExpire <= 30 {
		return "warning"
	}
	return "healthy"
}

func (h *DomainHandler) updateDomainRuntimeByDomain(domain string, runtime *domainRuntimeResult) {
	if runtime == nil {
		return
	}
	updates := map[string]interface{}{
		"dns_resolved":       runtime.DNSResolved,
		"http_status_code":   runtime.HTTPStatusCode,
		"response_time_ms":   runtime.ResponseTimeMS,
		"ssl_days_to_expire": runtime.SSLDaysToExpire,
		"health_status":      runtime.HealthStatus,
		"last_check_at":      runtime.CheckedAt,
	}
	_ = h.db.Model(&CloudDomain{}).Where("domain = ?", domain).Updates(updates).Error
}

func (h *DomainHandler) checkSSL(domain string) (map[string]interface{}, error) {
	conn, err := tls.DialWithDialer(&net.Dialer{Timeout: 10 * time.Second}, "tcp", domain+":443", &tls.Config{
		InsecureSkipVerify: true,
	})
	if err != nil {
		return nil, err
	}
	defer conn.Close()

	certs := conn.ConnectionState().PeerCertificates
	if len(certs) == 0 {
		return nil, fmt.Errorf("no certificates found")
	}

	cert := certs[0]
	daysToExpire := int(time.Until(cert.NotAfter).Hours() / 24)

	sans := strings.Join(cert.DNSNames, ", ")

	return map[string]interface{}{
		"issuer":         cert.Issuer.CommonName,
		"subject":        cert.Subject.CommonName,
		"sans":           sans,
		"not_before":     &cert.NotBefore,
		"not_after":      &cert.NotAfter,
		"days_to_expire": daysToExpire,
		"serial_number":  cert.SerialNumber.String(),
	}, nil
}

// 云厂商API调用
func (h *DomainHandler) syncAliyunDomains(account *CloudAccount) ([]CloudDomain, error) {
	if account == nil {
		return nil, fmt.Errorf("账号信息为空")
	}
	return nil, fmt.Errorf("阿里云域名同步未启用：请接入阿里云 SDK 后再使用")
}

func (h *DomainHandler) syncTencentDomains(account *CloudAccount) ([]CloudDomain, error) {
	if account == nil {
		return nil, fmt.Errorf("账号信息为空")
	}
	return nil, fmt.Errorf("腾讯云域名同步未启用：请接入腾讯云 SDK 后再使用")
}

// 用于解析云API响应
type aliyunDomainResponse struct {
	Domains struct {
		Domain []struct {
			DomainName       string `json:"DomainName"`
			RegistrationDate string `json:"RegistrationDate"`
			ExpirationDate   string `json:"ExpirationDate"`
			DomainStatus     string `json:"DomainStatus"`
		} `json:"Domain"`
	} `json:"Domains"`
}

func parseAliyunResponse(data []byte) ([]CloudDomain, error) {
	var resp aliyunDomainResponse
	if err := json.Unmarshal(data, &resp); err != nil {
		return nil, err
	}

	domains := make([]CloudDomain, 0)
	for _, d := range resp.Domains.Domain {
		domain := CloudDomain{
			Domain: d.DomainName,
			Status: d.DomainStatus,
		}

		if regDate, err := time.Parse("2006-01-02", d.RegistrationDate); err == nil {
			domain.RegistrationAt = &regDate
		}
		if expDate, err := time.Parse("2006-01-02", d.ExpirationDate); err == nil {
			domain.ExpirationAt = &expDate
		}

		domains = append(domains, domain)
	}

	return domains, nil
}

func calcDaysToExpire(expireAt *time.Time) int {
	if expireAt == nil {
		return 0
	}
	remain := expireAt.Sub(time.Now())
	if remain <= 0 {
		return 0
	}
	return int(math.Ceil(remain.Hours() / 24))
}

func (h *DomainHandler) refreshCertRuntimeFields(cert *SSLCertificate) {
	if cert == nil {
		return
	}
	cert.DaysToExpire = calcDaysToExpire(cert.NotAfter)
	switch {
	case cert.NotAfter == nil:
		// 保持原状态
	case cert.DaysToExpire <= 0:
		cert.Status = 0
	case cert.DaysToExpire <= 30:
		cert.Status = 2
	default:
		cert.Status = 1
	}
}

func (h *DomainHandler) refreshDomainRuntimeFields(domain *CloudDomain) {
	if domain == nil {
		return
	}
	domain.DaysToExpire = calcDaysToExpire(domain.ExpirationAt)
	if strings.TrimSpace(domain.HealthStatus) == "" {
		switch {
		case domain.DaysToExpire == 0:
			domain.HealthStatus = "critical"
		case domain.DaysToExpire > 0 && domain.DaysToExpire <= 30:
			domain.HealthStatus = "warning"
		default:
			domain.HealthStatus = "unknown"
		}
	}
}

package domain

import (
	"crypto/tls"
	"encoding/json"
	"fmt"
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
	c.JSON(http.StatusOK, gin.H{"code": 0, "data": domains})
}

// ListExpiringDomains 即将过期的域名
func (h *DomainHandler) ListExpiringDomains(c *gin.Context) {
	days := 30 // 默认30天内过期
	if d := c.Query("days"); d != "" {
		fmt.Sscanf(d, "%d", &days)
	}

	var domains []CloudDomain
	if err := h.db.Preload("Account").Where("days_to_expire <= ? AND days_to_expire > 0", days).Order("days_to_expire").Find(&domains).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": err.Error()})
		return
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

	// 查询WHOIS信息 (简化实现)
	result := gin.H{
		"domain": req.Domain,
		"status": "unknown",
	}

	// 检查DNS解析
	ips, err := net.LookupIP(req.Domain)
	if err == nil && len(ips) > 0 {
		ipList := make([]string, 0)
		for _, ip := range ips {
			ipList = append(ipList, ip.String())
		}
		result["dns_resolved"] = true
		result["ips"] = ipList
	} else {
		result["dns_resolved"] = false
	}

	// 检查SSL证书
	certInfo, err := h.checkSSL(req.Domain)
	if err == nil {
		result["ssl"] = certInfo
	}

	c.JSON(http.StatusOK, gin.H{"code": 0, "data": result})
}

// ListCerts SSL证书列表
func (h *DomainHandler) ListCerts(c *gin.Context) {
	var certs []SSLCertificate
	if err := h.db.Order("days_to_expire asc").Find(&certs).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": err.Error()})
		return
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

	h.db.Model(&cert).Updates(updates)

	c.JSON(http.StatusOK, gin.H{"code": 0, "data": certInfo})
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
	if err := h.db.Where("days_to_expire <= ? AND days_to_expire > 0", days).Order("days_to_expire").Find(&certs).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "data": certs})
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

// 云厂商API调用 (简化实现，实际需要使用SDK)
func (h *DomainHandler) syncAliyunDomains(account *CloudAccount) ([]CloudDomain, error) {
	// TODO: 使用阿里云SDK获取域名列表
	// 这里返回示例数据
	return []CloudDomain{}, nil
}

func (h *DomainHandler) syncTencentDomains(account *CloudAccount) ([]CloudDomain, error) {
	// TODO: 使用腾讯云SDK获取域名列表
	return []CloudDomain{}, nil
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

package notify

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/smtp"
	"net/url"
	"strings"
	"time"
)

// Sender 通知发送器
type Sender struct{}

func NewSender() *Sender {
	return &Sender{}
}

// Send 发送通知
func (s *Sender) Send(channel *NotifyChannel, title, content, receiver string) error {
	switch channel.Type {
	case "webhook":
		return s.sendWebhook(channel, title, content)
	case "feishu":
		return s.sendFeishu(channel, title, content)
	case "dingtalk":
		return s.sendDingTalk(channel, title, content)
	case "wecom":
		return s.sendWeCom(channel, title, content)
	case "email":
		return s.sendEmail(channel, title, content, receiver)
	default:
		return fmt.Errorf("不支持的通知类型: %s", channel.Type)
	}
}

// sendWebhook 发送Webhook
func (s *Sender) sendWebhook(channel *NotifyChannel, title, content string) error {
	payload := map[string]interface{}{
		"title":     title,
		"content":   content,
		"timestamp": time.Now().Unix(),
	}

	jsonData, _ := json.Marshal(payload)
	resp, err := http.Post(channel.Webhook, "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("webhook返回错误: %d, %s", resp.StatusCode, string(body))
	}
	return nil
}

// sendFeishu 发送飞书消息
func (s *Sender) sendFeishu(channel *NotifyChannel, title, content string) error {
	timestamp := time.Now().Unix()
	sign := ""

	// 如果有签名密钥，计算签名
	if channel.Secret != "" {
		stringToSign := fmt.Sprintf("%d\n%s", timestamp, channel.Secret)
		h := hmac.New(sha256.New, []byte(stringToSign))
		sign = base64.StdEncoding.EncodeToString(h.Sum(nil))
	}

	payload := map[string]interface{}{
		"msg_type": "interactive",
		"card": map[string]interface{}{
			"header": map[string]interface{}{
				"title": map[string]string{
					"tag":     "plain_text",
					"content": title,
				},
				"template": "blue",
			},
			"elements": []map[string]interface{}{
				{
					"tag":     "markdown",
					"content": content,
				},
			},
		},
	}

	if sign != "" {
		payload["timestamp"] = fmt.Sprintf("%d", timestamp)
		payload["sign"] = sign
	}

	jsonData, _ := json.Marshal(payload)
	resp, err := http.Post(channel.Webhook, "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	var result struct {
		Code int    `json:"code"`
		Msg  string `json:"msg"`
	}
	json.NewDecoder(resp.Body).Decode(&result)

	if result.Code != 0 {
		return fmt.Errorf("飞书发送失败: %s", result.Msg)
	}
	return nil
}

// sendDingTalk 发送钉钉消息
func (s *Sender) sendDingTalk(channel *NotifyChannel, title, content string) error {
	webhookURL := channel.Webhook

	// 如果有签名密钥，添加签名
	if channel.Secret != "" {
		timestamp := time.Now().UnixMilli()
		stringToSign := fmt.Sprintf("%d\n%s", timestamp, channel.Secret)
		h := hmac.New(sha256.New, []byte(channel.Secret))
		h.Write([]byte(stringToSign))
		sign := url.QueryEscape(base64.StdEncoding.EncodeToString(h.Sum(nil)))
		webhookURL = fmt.Sprintf("%s&timestamp=%d&sign=%s", channel.Webhook, timestamp, sign)
	}

	payload := map[string]interface{}{
		"msgtype": "markdown",
		"markdown": map[string]string{
			"title": title,
			"text":  fmt.Sprintf("## %s\n\n%s", title, content),
		},
	}

	jsonData, _ := json.Marshal(payload)
	resp, err := http.Post(webhookURL, "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	var result struct {
		ErrCode int    `json:"errcode"`
		ErrMsg  string `json:"errmsg"`
	}
	json.NewDecoder(resp.Body).Decode(&result)

	if result.ErrCode != 0 {
		return fmt.Errorf("钉钉发送失败: %s", result.ErrMsg)
	}
	return nil
}

// sendWeCom 发送企业微信消息
func (s *Sender) sendWeCom(channel *NotifyChannel, title, content string) error {
	payload := map[string]interface{}{
		"msgtype": "markdown",
		"markdown": map[string]string{
			"content": fmt.Sprintf("## %s\n\n%s", title, content),
		},
	}

	jsonData, _ := json.Marshal(payload)
	resp, err := http.Post(channel.Webhook, "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	var result struct {
		ErrCode int    `json:"errcode"`
		ErrMsg  string `json:"errmsg"`
	}
	json.NewDecoder(resp.Body).Decode(&result)

	if result.ErrCode != 0 {
		return fmt.Errorf("企业微信发送失败: %s", result.ErrMsg)
	}
	return nil
}

// sendEmail 发送邮件
func (s *Sender) sendEmail(channel *NotifyChannel, title, content, receiver string) error {
	if channel.SMTPHost == "" || channel.SMTPUser == "" {
		return fmt.Errorf("邮件配置不完整")
	}

	receivers := strings.Split(receiver, ",")
	if len(receivers) == 0 {
		return fmt.Errorf("没有接收人")
	}

	// 构建邮件内容
	header := make(map[string]string)
	header["From"] = channel.SMTPUser
	header["To"] = receiver
	header["Subject"] = title
	header["Content-Type"] = "text/html; charset=UTF-8"

	var msg strings.Builder
	for k, v := range header {
		msg.WriteString(fmt.Sprintf("%s: %s\r\n", k, v))
	}
	msg.WriteString("\r\n")
	msg.WriteString(fmt.Sprintf("<html><body><h2>%s</h2><pre>%s</pre></body></html>", title, content))

	auth := smtp.PlainAuth("", channel.SMTPUser, channel.SMTPPass, channel.SMTPHost)
	addr := fmt.Sprintf("%s:%d", channel.SMTPHost, channel.SMTPPort)

	return smtp.SendMail(addr, auth, channel.SMTPUser, receivers, []byte(msg.String()))
}

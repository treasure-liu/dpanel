package notice

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"strings"
	"time"
)

// WechatWorkConfig 企业微信配置
type WechatWorkConfig struct {
	WebhookURL string // 企业微信机器人webhook地址
	IsEnabled  bool   // 是否启用
}

// GetWechatWorkConfig 从环境变量获取企业微信配置
func GetWechatWorkConfig() WechatWorkConfig {
	webhookURL := os.Getenv("WECHATWORK_WEBHOOK_URL")
	isEnabled := strings.ToLower(os.Getenv("WECHATWORK_ENABLED")) == "true"

	return WechatWorkConfig{
		WebhookURL: webhookURL,
		IsEnabled:  isEnabled,
	}
}

// SendWechatWorkMessage 发送企业微信消息
func SendWechatWorkMessage(title string, content string, level string) error {
	config := GetWechatWorkConfig()
	if !config.IsEnabled || config.WebhookURL == "" {
		return nil
	}

	// 准备消息内容
	messageType := "通知"
	if level == TypeError {
		messageType = "警告"
	} else if level == TypeInfo {
		messageType = "信息"
	} else if level == TypeSuccess {
		messageType = "成功"
	}

	// 根据消息类型选择不同的显示样式
	var msgContent string
	switch level {
	case TypeError:
		msgContent = fmt.Sprintf("<font color=\"warning\">**%s: %s**</font>\n\n%s\n\n发送时间: %s", 
			messageType, 
			title, 
			content, 
			time.Now().Format("2006-01-02 15:04:05"))
	case TypeInfo:
		msgContent = fmt.Sprintf("<font color=\"info\">**%s: %s**</font>\n\n%s\n\n发送时间: %s", 
			messageType, 
			title, 
			content, 
			time.Now().Format("2006-01-02 15:04:05"))
	case TypeSuccess:
		msgContent = fmt.Sprintf("<font color=\"info\">**%s: %s**</font>\n\n%s\n\n发送时间: %s", 
			messageType, 
			title, 
			content, 
			time.Now().Format("2006-01-02 15:04:05"))
	default:
		msgContent = fmt.Sprintf("**%s: %s**\n\n%s\n\n发送时间: %s", 
			messageType, 
			title, 
			content, 
			time.Now().Format("2006-01-02 15:04:05"))
	}

	// 构建请求数据
	requestData := map[string]interface{}{
		"msgtype": "markdown",
		"markdown": map[string]string{
			"content": msgContent,
		},
	}

	jsonData, err := json.Marshal(requestData)
	if err != nil {
		return err
	}

	// 发送请求
	req, err := http.NewRequest(http.MethodPost, config.WebhookURL, bytes.NewReader(jsonData))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("failed to send wechat work message, status code: %d, body: %s", resp.StatusCode, string(body))
	}

	body, _ := io.ReadAll(resp.Body)
	slog.Debug("wechat work response", "body", string(body))
	return nil
} 
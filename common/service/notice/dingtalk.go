package notice

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"
)

// DingtalkConfig 钉钉配置
type DingtalkConfig struct {
	WebhookURL string // 钉钉机器人webhook地址
	Secret     string // 钉钉机器人安全设置的签名
	IsEnabled  bool   // 是否启用
}

// GetDingtalkConfig 从环境变量获取钉钉配置
func GetDingtalkConfig() DingtalkConfig {
	webhookURL := os.Getenv("DINGTALK_WEBHOOK_URL")
	secret := os.Getenv("DINGTALK_SECRET")
	isEnabled := strings.ToLower(os.Getenv("DINGTALK_ENABLED")) == "true"

	return DingtalkConfig{
		WebhookURL: webhookURL,
		Secret:     secret,
		IsEnabled:  isEnabled,
	}
}

// SendDingtalkMessage 发送钉钉消息
func SendDingtalkMessage(title string, content string, level string) error {
	config := GetDingtalkConfig()
	if !config.IsEnabled || config.WebhookURL == "" {
		return nil
	}

	// 准备消息内容
	messageType := "text"
	if level == TypeError {
		messageType = "警告"
	} else if level == TypeInfo {
		messageType = "信息"
	} else if level == TypeSuccess {
		messageType = "成功"
	}

	// 构建markdown消息
	msgContent := fmt.Sprintf("### %s: %s\n\n%s\n\n###### 发送时间: %s", 
		messageType, 
		title, 
		content, 
		time.Now().Format("2006-01-02 15:04:05"))

	// 构建请求数据
	requestData := map[string]interface{}{
		"msgtype": "markdown",
		"markdown": map[string]string{
			"title": title,
			"text":  msgContent,
		},
	}

	jsonData, err := json.Marshal(requestData)
	if err != nil {
		return err
	}

	// 生成签名URL
	webhookURL := config.WebhookURL
	if config.Secret != "" {
		timestamp := strconv.FormatInt(time.Now().UnixMilli(), 10)
		sign := generateDingtalkSign(timestamp, config.Secret)
		webhookURL = fmt.Sprintf("%s&timestamp=%s&sign=%s", config.WebhookURL, timestamp, sign)
	}

	// 发送请求
	req, err := http.NewRequest(http.MethodPost, webhookURL, strings.NewReader(string(jsonData)))
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
		return fmt.Errorf("failed to send dingtalk message, status code: %d, body: %s", resp.StatusCode, string(body))
	}

	body, _ := io.ReadAll(resp.Body)
	slog.Debug("dingtalk response", "body", string(body))
	return nil
}

// generateDingtalkSign 生成钉钉签名
func generateDingtalkSign(timestamp, secret string) string {
	stringToSign := timestamp + "\n" + secret
	h := hmac.New(sha256.New, []byte(secret))
	h.Write([]byte(stringToSign))
	signature := base64.StdEncoding.EncodeToString(h.Sum(nil))
	return url.QueryEscape(signature)
} 
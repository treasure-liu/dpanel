package notice

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"
)

// FeishuConfig 飞书配置
type FeishuConfig struct {
	WebhookURL string // 飞书机器人webhook地址
	Secret     string // 飞书机器人安全设置的签名
	IsEnabled  bool   // 是否启用
}

// GetFeishuConfig 从环境变量获取飞书配置
func GetFeishuConfig() FeishuConfig {
	webhookURL := os.Getenv("FEISHU_WEBHOOK_URL")
	secret := os.Getenv("FEISHU_SECRET")
	isEnabled := strings.ToLower(os.Getenv("FEISHU_ENABLED")) == "true"

	return FeishuConfig{
		WebhookURL: webhookURL,
		Secret:     secret,
		IsEnabled:  isEnabled,
	}
}

// SendFeishuMessage 发送飞书消息
func SendFeishuMessage(title string, content string, level string) error {
	config := GetFeishuConfig()
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

	// 构建飞书卡片消息
	timestamp := time.Now().Unix()
	var color string
	switch level {
	case TypeError:
		color = "red"
	case TypeInfo:
		color = "blue"
	case TypeSuccess:
		color = "green"
	default:
		color = "grey"
	}

	// 构建飞书卡片消息
	requestData := map[string]interface{}{
		"timestamp": timestamp,
		"msg_type":  "interactive",
		"card": map[string]interface{}{
			"config": map[string]interface{}{
				"wide_screen_mode": true,
			},
			"header": map[string]interface{}{
				"title": map[string]interface{}{
					"tag":     "plain_text",
					"content": fmt.Sprintf("%s: %s", messageType, title),
				},
				"template": color,
			},
			"elements": []map[string]interface{}{
				{
					"tag": "div",
					"text": map[string]interface{}{
						"tag":     "lark_md",
						"content": content,
					},
				},
				{
					"tag": "hr",
				},
				{
					"tag": "note",
					"elements": []map[string]interface{}{
						{
							"tag":     "plain_text",
							"content": fmt.Sprintf("发送时间: %s", time.Now().Format("2006-01-02 15:04:05")),
						},
					},
				},
			},
		},
	}

	// 生成签名
	if config.Secret != "" {
		sign := generateFeishuSign(timestamp, config.Secret)
		requestData["sign"] = sign
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
		return fmt.Errorf("failed to send feishu message, status code: %d, body: %s", resp.StatusCode, string(body))
	}

	body, _ := io.ReadAll(resp.Body)
	slog.Debug("feishu response", "body", string(body))
	return nil
}

// generateFeishuSign 生成飞书签名
func generateFeishuSign(timestamp int64, secret string) string {
	// timestamp + key 做sha256, 再进行base64 encode
	stringToSign := fmt.Sprintf("%v", timestamp) + secret
	h := hmac.New(sha256.New, []byte(stringToSign))
	h.Write([]byte{})
	signature := base64.StdEncoding.EncodeToString(h.Sum(nil))
	return signature
} 
package notice

import (
	"bytes"
	"crypto/tls"
	"fmt"
	"html/template"
	"log/slog"
	"net/smtp"
	"os"
	"strings"
	"time"
)

// EmailConfig 邮件配置
type EmailConfig struct {
	Host     string // SMTP服务器地址
	Port     string // SMTP服务器端口
	Username string // SMTP用户名
	Password string // SMTP密码
	From     string // 发件人邮箱
	To       string // 收件人邮箱（多个用逗号分隔）
	IsSSL    bool   // 是否使用SSL
	IsEnabled bool  // 是否启用
}

// GetEmailConfig 从环境变量获取邮件配置
func GetEmailConfig() EmailConfig {
	host := os.Getenv("EMAIL_HOST")
	port := os.Getenv("EMAIL_PORT")
	username := os.Getenv("EMAIL_USERNAME")
	password := os.Getenv("EMAIL_PASSWORD")
	from := os.Getenv("EMAIL_FROM")
	to := os.Getenv("EMAIL_TO")
	isSSL := strings.ToLower(os.Getenv("EMAIL_USE_SSL")) == "true"
	isEnabled := strings.ToLower(os.Getenv("EMAIL_ENABLED")) == "true"

	return EmailConfig{
		Host:     host,
		Port:     port,
		Username: username,
		Password: password,
		From:     from,
		To:       to,
		IsSSL:    isSSL,
		IsEnabled: isEnabled,
	}
}

// 邮件模板
const emailHTMLTemplate = `
<!DOCTYPE html>
<html lang="zh-CN">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>{{.Title}}</title>
    <style>
        body {
            font-family: "Helvetica Neue", Arial, sans-serif;
            line-height: 1.6;
            color: #333;
            max-width: 800px;
            margin: 0 auto;
            padding: 20px;
        }
        .container {
            border: 1px solid #e1e1e1;
            border-radius: 5px;
            overflow: hidden;
            box-shadow: 0 2px 10px rgba(0,0,0,0.05);
        }
        .header {
            padding: 20px;
            background-color: {{.HeaderColor}};
            color: white;
        }
        .header h1 {
            margin: 0;
            font-size: 24px;
        }
        .content {
            padding: 20px;
            background-color: #fff;
        }
        .message {
            white-space: pre-line;
            background-color: #f8f9fa;
            padding: 15px;
            border-radius: 4px;
            border-left: 4px solid {{.BorderColor}};
        }
        .footer {
            text-align: center;
            padding: 15px;
            font-size: 12px;
            color: #6c757d;
            background-color: #f8f9fa;
            border-top: 1px solid #e1e1e1;
        }
        .timestamp {
            color: #6c757d;
            font-size: 14px;
            margin-top: 15px;
        }
        .alert-icon {
            font-size: 24px;
            margin-right: 10px;
            vertical-align: middle;
        }
    </style>
</head>
<body>
    <div class="container">
        <div class="header">
            <h1>{{.Icon}} {{.Title}}</h1>
        </div>
        <div class="content">
            <h2>{{.SubTitle}}</h2>
            <div class="message">{{.Message}}</div>
            <div class="timestamp">发送时间: {{.Time}}</div>
        </div>
        <div class="footer">
            此邮件由 DPanel 系统自动发送，请勿回复
        </div>
    </div>
</body>
</html>
`

// SendEmailMessage 发送邮件消息
func SendEmailMessage(title string, content string, level string) error {
	config := GetEmailConfig()
	if !config.IsEnabled || config.Host == "" || config.To == "" {
		return nil
	}

	// 准备消息内容
	messageType := "通知"
	headerColor := "#3498db" // 默认蓝色
	borderColor := "#3498db"
	icon := "📝"
	
	switch level {
	case TypeError:
		messageType = "警告"
		headerColor = "#e74c3c" // 红色
		borderColor = "#e74c3c"
		icon = "⚠️"
	case TypeInfo:
		messageType = "信息"
		headerColor = "#3498db" // 蓝色
		borderColor = "#3498db"
		icon = "ℹ️"
	case TypeSuccess:
		messageType = "成功"
		headerColor = "#2ecc71" // 绿色
		borderColor = "#2ecc71"
		icon = "✅"
	}

	// 准备邮件数据
	data := struct {
		Title       string
		SubTitle    string
		Message     string
		Time        string
		HeaderColor string
		BorderColor string
		Icon        string
	}{
		Title:       fmt.Sprintf("%s通知", messageType),
		SubTitle:    title,
		Message:     content,
		Time:        time.Now().Format("2006-01-02 15:04:05"),
		HeaderColor: headerColor,
		BorderColor: borderColor,
		Icon:        icon,
	}

	// 解析模板
	tmpl, err := template.New("emailTemplate").Parse(emailHTMLTemplate)
	if err != nil {
		return fmt.Errorf("解析邮件模板失败: %w", err)
	}

	// 渲染模板
	var body bytes.Buffer
	if err := tmpl.Execute(&body, data); err != nil {
		return fmt.Errorf("渲染邮件模板失败: %w", err)
	}

	// 设置邮件头信息
	headers := make(map[string]string)
	headers["From"] = config.From
	headers["To"] = config.To
	headers["Subject"] = fmt.Sprintf("DPanel %s通知: %s", messageType, title)
	headers["MIME-Version"] = "1.0"
	headers["Content-Type"] = "text/html; charset=UTF-8"

	// 构建邮件内容
	message := ""
	for k, v := range headers {
		message += fmt.Sprintf("%s: %s\r\n", k, v)
	}
	message += "\r\n" + body.String()

	// 准备收件人列表
	recipientList := strings.Split(config.To, ",")
	for i, recipient := range recipientList {
		recipientList[i] = strings.TrimSpace(recipient)
	}

	// 发送邮件
	addr := fmt.Sprintf("%s:%s", config.Host, config.Port)
	auth := smtp.PlainAuth("", config.Username, config.Password, config.Host)

	var err2 error
	if config.IsSSL {
		// 使用SSL发送
		tlsConfig := &tls.Config{
			InsecureSkipVerify: true,
			ServerName:         config.Host,
		}
		
		conn, err := tls.Dial("tcp", addr, tlsConfig)
		if err != nil {
			return fmt.Errorf("连接到SMTP服务器失败: %w", err)
		}
		defer conn.Close()
		
		client, err := smtp.NewClient(conn, config.Host)
		if err != nil {
			return fmt.Errorf("创建SMTP客户端失败: %w", err)
		}
		defer client.Close()
		
		if err = client.Auth(auth); err != nil {
			return fmt.Errorf("SMTP认证失败: %w", err)
		}
		
		if err = client.Mail(config.From); err != nil {
			return fmt.Errorf("SMTP设置发件人失败: %w", err)
		}
		
		for _, recipient := range recipientList {
			if err = client.Rcpt(recipient); err != nil {
				return fmt.Errorf("SMTP设置收件人失败: %w", err)
			}
		}
		
		w, err := client.Data()
		if err != nil {
			return fmt.Errorf("SMTP准备发送数据失败: %w", err)
		}
		
		_, err = w.Write([]byte(message))
		if err != nil {
			return fmt.Errorf("SMTP写入数据失败: %w", err)
		}
		
		err = w.Close()
		if err != nil {
			return fmt.Errorf("SMTP关闭连接失败: %w", err)
		}
		
		err2 = client.Quit()
	} else {
		// 使用普通SMTP发送
		err2 = smtp.SendMail(addr, auth, config.From, recipientList, []byte(message))
	}

	if err2 != nil {
		return fmt.Errorf("发送邮件失败: %w", err2)
	}

	slog.Debug("邮件发送成功", "to", config.To, "subject", title)
	return nil
} 
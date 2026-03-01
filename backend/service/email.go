package service

import (
	"crypto/tls"
	"fmt"
	"math/rand"
	"net/smtp"
	"sync"
	"time"

	"finance-system/config"
)

// EmailService 邮件服务
type EmailService struct {
	config *config.EmailConfig
}

// 验证码存储（内存存储，生产环境建议使用 Redis）
var (
	verificationCodes = make(map[string]*VerificationCode)
	codesMutex        sync.RWMutex
)

// VerificationCode 验证码信息
type VerificationCode struct {
	Code      string
	Email     string
	ExpiresAt time.Time
	Used      bool
}

// NewEmailService 创建邮件服务
func NewEmailService(cfg *config.EmailConfig) *EmailService {
	return &EmailService{config: cfg}
}

// GenerateCode 生成6位验证码
func GenerateCode() string {
	rand.Seed(time.Now().UnixNano())
	return fmt.Sprintf("%06d", rand.Intn(1000000))
}

// SendVerificationCode 发送验证码邮件
func (s *EmailService) SendVerificationCode(email string) (string, error) {
	if !s.config.Enabled {
		// 如果邮件功能未启用，返回固定验证码用于测试
		code := "123456"
		s.storeCode(email, code)
		return code, nil
	}

	code := GenerateCode()

	subject := "验证码 - 财务管理系统"
	body := fmt.Sprintf(`
<!DOCTYPE html>
<html>
<head>
    <meta charset="UTF-8">
    <style>
        body { font-family: Arial, sans-serif; line-height: 1.6; color: #333; }
        .container { max-width: 600px; margin: 0 auto; padding: 20px; }
        .code { font-size: 32px; font-weight: bold; color: #4A90E2; letter-spacing: 5px; margin: 20px 0; }
        .footer { margin-top: 30px; font-size: 12px; color: #999; }
    </style>
</head>
<body>
    <div class="container">
        <h2>您好！</h2>
        <p>您正在注册财务管理系统账号，您的验证码是：</p>
        <div class="code">%s</div>
        <p>验证码有效期为 <strong>5 分钟</strong>，请勿泄露给他人。</p>
        <p>如果这不是您的操作，请忽略此邮件。</p>
        <div class="footer">
            <p>此邮件由系统自动发送，请勿回复。</p>
        </div>
    </div>
</body>
</html>
`, code)

	err := s.sendEmail(email, subject, body)
	if err != nil {
		return "", fmt.Errorf("发送邮件失败: %w", err)
	}

	// 存储验证码
	s.storeCode(email, code)

	return code, nil
}

// storeCode 存储验证码
func (s *EmailService) storeCode(email, code string) {
	codesMutex.Lock()
	defer codesMutex.Unlock()

	verificationCodes[email] = &VerificationCode{
		Code:      code,
		Email:     email,
		ExpiresAt: time.Now().Add(5 * time.Minute),
		Used:      false,
	}
}

// VerifyCode 验证验证码
func (s *EmailService) VerifyCode(email, code string) bool {
	codesMutex.Lock()
	defer codesMutex.Unlock()

	vc, exists := verificationCodes[email]
	if !exists {
		return false
	}

	if vc.Used {
		return false
	}

	if time.Now().After(vc.ExpiresAt) {
		delete(verificationCodes, email)
		return false
	}

	if vc.Code != code {
		return false
	}

	// 标记为已使用
	vc.Used = true
	return true
}

// sendEmail 发送邮件
func (s *EmailService) sendEmail(to, subject, body string) error {
	from := s.config.From
	if s.config.FromName != "" {
		from = fmt.Sprintf("%s <%s>", s.config.FromName, s.config.From)
	}

	headers := make(map[string]string)
	headers["From"] = from
	headers["To"] = to
	headers["Subject"] = subject
	headers["MIME-Version"] = "1.0"
	headers["Content-Type"] = "text/html; charset=UTF-8"

	message := ""
	for k, v := range headers {
		message += fmt.Sprintf("%s: %s\r\n", k, v)
	}
	message += "\r\n" + body

	addr := fmt.Sprintf("%s:%d", s.config.Host, s.config.Port)

	// 根据端口选择连接方式
	if s.config.Port == 465 {
		// SSL/TLS 直接加密连接
		return s.sendMailSSL(addr, to, []byte(message))
	}

	// STARTTLS 方式
	auth := smtp.PlainAuth("", s.config.Username, s.config.Password, s.config.Host)
	return smtp.SendMail(addr, auth, s.config.From, []string{to}, []byte(message))
}

// sendMailSSL 通过 SSL 发送邮件（端口 465）
func (s *EmailService) sendMailSSL(addr, to string, msg []byte) error {
	tlsConfig := &tls.Config{
		ServerName: s.config.Host,
	}

	conn, err := tls.Dial("tcp", addr, tlsConfig)
	if err != nil {
		return fmt.Errorf("TLS连接失败: %w", err)
	}
	defer conn.Close()

	client, err := smtp.NewClient(conn, s.config.Host)
	if err != nil {
		return fmt.Errorf("创建SMTP客户端失败: %w", err)
	}
	defer client.Close()

	auth := smtp.PlainAuth("", s.config.Username, s.config.Password, s.config.Host)
	if err := client.Auth(auth); err != nil {
		return fmt.Errorf("SMTP认证失败: %w", err)
	}

	if err := client.Mail(s.config.From); err != nil {
		return fmt.Errorf("设置发件人失败: %w", err)
	}

	if err := client.Rcpt(to); err != nil {
		return fmt.Errorf("设置收件人失败: %w", err)
	}

	w, err := client.Data()
	if err != nil {
		return fmt.Errorf("获取数据写入器失败: %w", err)
	}

	_, err = w.Write(msg)
	if err != nil {
		return fmt.Errorf("写入邮件内容失败: %w", err)
	}

	err = w.Close()
	if err != nil {
		return fmt.Errorf("关闭数据写入器失败: %w", err)
	}

	return client.Quit()
}

// CleanExpiredCodes 清理过期验证码（可定期调用）
func CleanExpiredCodes() {
	codesMutex.Lock()
	defer codesMutex.Unlock()

	now := time.Now()
	for email, vc := range verificationCodes {
		if now.After(vc.ExpiresAt) || vc.Used {
			delete(verificationCodes, email)
		}
	}
}

package service

import (
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"mime/multipart"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// UploadService 上传服务
type UploadService struct {
	uploadDir string
	baseURL   string
}

// NewUploadService 创建上传服务
func NewUploadService(uploadDir, baseURL string) *UploadService {
	// 确保上传目录存在
	os.MkdirAll(uploadDir, 0755)
	return &UploadService{
		uploadDir: uploadDir,
		baseURL:   baseURL,
	}
}

// UploadImage 上传图片文件
func (s *UploadService) UploadImage(file multipart.File, header *multipart.FileHeader) (string, error) {
	// 验证文件类型
	ext := strings.ToLower(filepath.Ext(header.Filename))
	allowedExts := map[string]bool{
		".jpg":  true,
		".jpeg": true,
		".png":  true,
		".gif":  true,
		".webp": true,
	}

	if !allowedExts[ext] {
		return "", errors.New("不支持的图片格式，仅支持 jpg, jpeg, png, gif, webp")
	}

	// 验证文件大小（最大 5MB）
	if header.Size > 5*1024*1024 {
		return "", errors.New("图片大小不能超过 5MB")
	}

	// 生成文件名
	filename := fmt.Sprintf("%d%s", time.Now().UnixNano(), ext)

	// 按日期创建子目录
	dateDir := time.Now().Format("2006/01/02")
	fullDir := filepath.Join(s.uploadDir, dateDir)
	if err := os.MkdirAll(fullDir, 0755); err != nil {
		return "", err
	}

	// 保存文件
	filePath := filepath.Join(fullDir, filename)
	dst, err := os.Create(filePath)
	if err != nil {
		return "", err
	}
	defer dst.Close()

	if _, err := io.Copy(dst, file); err != nil {
		return "", err
	}

	// 返回相对路径
	relativePath := filepath.Join(dateDir, filename)
	return strings.ReplaceAll(relativePath, "\\", "/"), nil
}

// UploadBase64Image 上传 Base64 编码的图片
func (s *UploadService) UploadBase64Image(base64Data string) (string, error) {
	// 解析 Base64 数据
	// 支持格式: data:image/png;base64,xxxxx 或直接 base64 字符串
	var imageData []byte
	var ext string

	if strings.HasPrefix(base64Data, "data:image/") {
		// 解析 data URL
		parts := strings.SplitN(base64Data, ",", 2)
		if len(parts) != 2 {
			return "", errors.New("无效的 Base64 图片数据")
		}

		// 获取图片类型
		mimeType := strings.TrimPrefix(strings.Split(parts[0], ";")[0], "data:")
		switch mimeType {
		case "image/jpeg", "image/jpg":
			ext = ".jpg"
		case "image/png":
			ext = ".png"
		case "image/gif":
			ext = ".gif"
		case "image/webp":
			ext = ".webp"
		default:
			return "", errors.New("不支持的图片格式")
		}

		var err error
		imageData, err = base64.StdEncoding.DecodeString(parts[1])
		if err != nil {
			return "", errors.New("Base64 解码失败")
		}
	} else {
		// 尝试直接解码
		var err error
		imageData, err = base64.StdEncoding.DecodeString(base64Data)
		if err != nil {
			return "", errors.New("Base64 解码失败")
		}
		// 默认使用 jpg 扩展名
		ext = ".jpg"
	}

	// 验证文件大小（最大 5MB）
	if len(imageData) > 5*1024*1024 {
		return "", errors.New("图片大小不能超过 5MB")
	}

	// 生成文件名
	filename := fmt.Sprintf("%d%s", time.Now().UnixNano(), ext)

	// 按日期创建子目录
	dateDir := time.Now().Format("2006/01/02")
	fullDir := filepath.Join(s.uploadDir, dateDir)
	if err := os.MkdirAll(fullDir, 0755); err != nil {
		return "", err
	}

	// 保存文件
	filePath := filepath.Join(fullDir, filename)
	if err := os.WriteFile(filePath, imageData, 0644); err != nil {
		return "", err
	}

	// 返回相对路径
	relativePath := filepath.Join(dateDir, filename)
	return strings.ReplaceAll(relativePath, "\\", "/"), nil
}

// DeleteImage 删除图片
func (s *UploadService) DeleteImage(imagePath string) error {
	if imagePath == "" {
		return nil
	}

	fullPath := filepath.Join(s.uploadDir, imagePath)
	if _, err := os.Stat(fullPath); os.IsNotExist(err) {
		return nil // 文件不存在，视为删除成功
	}

	return os.Remove(fullPath)
}

// GetImagePath 获取图片完整路径
func (s *UploadService) GetImagePath(relativePath string) string {
	if relativePath == "" {
		return ""
	}
	if s.baseURL != "" {
		return s.baseURL + "/uploads/" + relativePath
	}
	return "/uploads/" + relativePath
}

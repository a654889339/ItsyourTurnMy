package service

import (
	"bytes"
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/tencentyun/cos-go-sdk-v5"
)

// UploadService 上传服务
type UploadService struct {
	uploadDir   string
	baseURL     string
	cosClient   *cos.Client
	cosBucket   string
	cosRegion   string
	cosEnabled  bool
}

// COSConfig 腾讯云 COS 配置
type COSConfig struct {
	SecretID  string
	SecretKey string
	Bucket    string
	Region    string
}

// NewUploadService 创建上传服务
func NewUploadService(uploadDir, baseURL string) *UploadService {
	// 确保上传目录存在
	os.MkdirAll(uploadDir, 0755)

	svc := &UploadService{
		uploadDir:  uploadDir,
		baseURL:    baseURL,
		cosEnabled: false,
	}

	// 尝试从环境变量初始化 COS
	secretID := os.Getenv("COS_SECRET_ID")
	secretKey := os.Getenv("COS_SECRET_KEY")
	bucket := os.Getenv("COS_BUCKET")
	region := os.Getenv("COS_REGION")

	if secretID != "" && secretKey != "" && bucket != "" && region != "" {
		err := svc.InitCOS(COSConfig{
			SecretID:  secretID,
			SecretKey: secretKey,
			Bucket:    bucket,
			Region:    region,
		})
		if err != nil {
			fmt.Printf("COS 初始化失败: %v，将使用本地存储\n", err)
		} else {
			fmt.Printf("COS 已启用，存储桶: %s，地域: %s\n", bucket, region)
		}
	}

	return svc
}

// InitCOS 初始化腾讯云 COS 客户端
func (s *UploadService) InitCOS(config COSConfig) error {
	// 构建 COS URL
	bucketURL, err := url.Parse(fmt.Sprintf("https://%s.cos.%s.myqcloud.com", config.Bucket, config.Region))
	if err != nil {
		return err
	}

	// 创建 COS 客户端
	s.cosClient = cos.NewClient(&cos.BaseURL{BucketURL: bucketURL}, &http.Client{
		Transport: &cos.AuthorizationTransport{
			SecretID:  config.SecretID,
			SecretKey: config.SecretKey,
		},
	})

	s.cosBucket = config.Bucket
	s.cosRegion = config.Region
	s.cosEnabled = true

	return nil
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

	// 读取文件内容
	fileData, err := io.ReadAll(file)
	if err != nil {
		return "", err
	}

	// 生成文件名
	filename := fmt.Sprintf("%d%s", time.Now().UnixNano(), ext)
	dateDir := time.Now().Format("2006/01/02")
	objectKey := fmt.Sprintf("images/%s/%s", dateDir, filename)

	// 如果启用了 COS，上传到 COS
	if s.cosEnabled {
		return s.uploadToCOS(objectKey, fileData)
	}

	// 否则保存到本地
	return s.saveToLocal(objectKey, fileData)
}

// UploadBase64Image 上传 Base64 编码的图片
func (s *UploadService) UploadBase64Image(base64Data string) (string, error) {
	// 解析 Base64 数据
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
		ext = ".jpg"
	}

	// 验证文件大小（最大 5MB）
	if len(imageData) > 5*1024*1024 {
		return "", errors.New("图片大小不能超过 5MB")
	}

	// 生成文件名
	filename := fmt.Sprintf("%d%s", time.Now().UnixNano(), ext)
	dateDir := time.Now().Format("2006/01/02")
	objectKey := fmt.Sprintf("images/%s/%s", dateDir, filename)

	// 如果启用了 COS，上传到 COS
	if s.cosEnabled {
		return s.uploadToCOS(objectKey, imageData)
	}

	// 否则保存到本地
	return s.saveToLocal(objectKey, imageData)
}

// uploadToCOS 上传到腾讯云 COS
func (s *UploadService) uploadToCOS(objectKey string, data []byte) (string, error) {
	ctx := context.Background()

	// 上传文件
	_, err := s.cosClient.Object.Put(ctx, objectKey, bytes.NewReader(data), nil)
	if err != nil {
		return "", fmt.Errorf("上传到 COS 失败: %v", err)
	}

	// 返回完整的 COS URL
	cosURL := fmt.Sprintf("https://%s.cos.%s.myqcloud.com/%s", s.cosBucket, s.cosRegion, objectKey)
	return cosURL, nil
}

// saveToLocal 保存到本地
func (s *UploadService) saveToLocal(objectKey string, data []byte) (string, error) {
	// 按日期创建子目录
	fullDir := filepath.Join(s.uploadDir, filepath.Dir(objectKey))
	if err := os.MkdirAll(fullDir, 0755); err != nil {
		return "", err
	}

	// 保存文件
	filePath := filepath.Join(s.uploadDir, objectKey)
	if err := os.WriteFile(filePath, data, 0644); err != nil {
		return "", err
	}

	// 返回相对路径
	return "/" + strings.ReplaceAll(objectKey, "\\", "/"), nil
}

// DeleteImage 删除图片
func (s *UploadService) DeleteImage(imagePath string) error {
	if imagePath == "" {
		return nil
	}

	// 如果是 COS URL，从 COS 删除
	if strings.Contains(imagePath, ".cos.") && strings.Contains(imagePath, ".myqcloud.com") {
		if s.cosEnabled {
			// 提取 object key
			u, err := url.Parse(imagePath)
			if err != nil {
				return err
			}
			objectKey := strings.TrimPrefix(u.Path, "/")
			_, err = s.cosClient.Object.Delete(context.Background(), objectKey)
			return err
		}
		return nil
	}

	// 否则从本地删除
	fullPath := filepath.Join(s.uploadDir, imagePath)
	if _, err := os.Stat(fullPath); os.IsNotExist(err) {
		return nil
	}

	return os.Remove(fullPath)
}

// GetImagePath 获取图片完整路径
func (s *UploadService) GetImagePath(relativePath string) string {
	if relativePath == "" {
		return ""
	}

	// 如果已经是完整的 URL，直接返回
	if strings.HasPrefix(relativePath, "http://") || strings.HasPrefix(relativePath, "https://") {
		return relativePath
	}

	if s.baseURL != "" {
		return s.baseURL + "/uploads/" + relativePath
	}
	return "/uploads/" + relativePath
}

// IsCOSEnabled 检查 COS 是否启用
func (s *UploadService) IsCOSEnabled() bool {
	return s.cosEnabled
}

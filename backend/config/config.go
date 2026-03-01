package config

import (
	"fmt"
	"os"
	"regexp"
	"strconv"
	"strings"

	"gopkg.in/yaml.v3"
)

// Config 应用配置
type Config struct {
	Server       ServerConfig       `yaml:"server"`
	Database     DatabaseConfig     `yaml:"database"`
	JWT          JWTConfig          `yaml:"jwt"`
	Log          LogConfig          `yaml:"log"`
	CORS         CORSConfig         `yaml:"cors"`
	TencentCloud TencentCloudConfig `yaml:"tencent_cloud"`
}

// ServerConfig 服务器配置
type ServerConfig struct {
	HTTPPort     int    `yaml:"http_port"`
	GRPCPort     int    `yaml:"grpc_port"`
	Mode         string `yaml:"mode"`
	ReadTimeout  int    `yaml:"read_timeout"`
	WriteTimeout int    `yaml:"write_timeout"`
}

// DatabaseConfig 数据库配置
type DatabaseConfig struct {
	Driver     string      `yaml:"driver"`
	SQLitePath string      `yaml:"sqlite_path"`
	MySQL      MySQLConfig `yaml:"mysql"`
}

// MySQLConfig MySQL配置
type MySQLConfig struct {
	Host         string `yaml:"host"`
	Port         int    `yaml:"port"`
	Username     string `yaml:"username"`
	Password     string `yaml:"password"`
	Database     string `yaml:"database"`
	Charset      string `yaml:"charset"`
	MaxIdleConns int    `yaml:"max_idle_conns"`
	MaxOpenConns int    `yaml:"max_open_conns"`
}

// JWTConfig JWT配置
type JWTConfig struct {
	Secret      string `yaml:"secret"`
	ExpireHours int    `yaml:"expire_hours"`
	Issuer      string `yaml:"issuer"`
}

// LogConfig 日志配置
type LogConfig struct {
	Level      string `yaml:"level"`
	Format     string `yaml:"format"`
	Output     string `yaml:"output"`
	FilePath   string `yaml:"file_path"`
	MaxSize    int    `yaml:"max_size"`
	MaxAge     int    `yaml:"max_age"`
	MaxBackups int    `yaml:"max_backups"`
}

// CORSConfig CORS配置
type CORSConfig struct {
	AllowedOrigins   []string `yaml:"allowed_origins"`
	AllowedMethods   []string `yaml:"allowed_methods"`
	AllowedHeaders   []string `yaml:"allowed_headers"`
	AllowCredentials bool     `yaml:"allow_credentials"`
}

// TencentCloudConfig 腾讯云配置
type TencentCloudConfig struct {
	COS COSConfig `yaml:"cos"`
	CLS CLSConfig `yaml:"cls"`
}

// COSConfig 腾讯云对象存储配置
type COSConfig struct {
	Enabled   bool   `yaml:"enabled"`
	SecretID  string `yaml:"secret_id"`
	SecretKey string `yaml:"secret_key"`
	Region    string `yaml:"region"`
	Bucket    string `yaml:"bucket"`
}

// CLSConfig 腾讯云日志服务配置
type CLSConfig struct {
	Enabled   bool   `yaml:"enabled"`
	SecretID  string `yaml:"secret_id"`
	SecretKey string `yaml:"secret_key"`
	Region    string `yaml:"region"`
	TopicID   string `yaml:"topic_id"`
}

// 全局配置实例
var Cfg *Config

// Load 加载配置文件
func Load(path string) (*Config, error) {
	// 读取配置文件
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("读取配置文件失败: %w", err)
	}

	// 替换环境变量
	content := expandEnvVars(string(data))

	// 解析 YAML
	var cfg Config
	if err := yaml.Unmarshal([]byte(content), &cfg); err != nil {
		return nil, fmt.Errorf("解析配置文件失败: %w", err)
	}

	// 设置默认值
	setDefaults(&cfg)

	Cfg = &cfg
	return &cfg, nil
}

// expandEnvVars 替换配置中的环境变量
// 支持格式: ${ENV_VAR} 或 ${ENV_VAR:default_value}
func expandEnvVars(content string) string {
	re := regexp.MustCompile(`\$\{([^}:]+)(?::([^}]*))?\}`)
	return re.ReplaceAllStringFunc(content, func(match string) string {
		groups := re.FindStringSubmatch(match)
		envVar := groups[1]
		defaultVal := ""
		if len(groups) > 2 {
			defaultVal = groups[2]
		}

		if val := os.Getenv(envVar); val != "" {
			return val
		}
		return defaultVal
	})
}

// setDefaults 设置默认值
func setDefaults(cfg *Config) {
	if cfg.Server.HTTPPort == 0 {
		cfg.Server.HTTPPort = 8080
	}
	if cfg.Server.GRPCPort == 0 {
		cfg.Server.GRPCPort = 9090
	}
	if cfg.Server.Mode == "" {
		cfg.Server.Mode = "release"
	}
	if cfg.Server.ReadTimeout == 0 {
		cfg.Server.ReadTimeout = 30
	}
	if cfg.Server.WriteTimeout == 0 {
		cfg.Server.WriteTimeout = 30
	}

	if cfg.Database.Driver == "" {
		cfg.Database.Driver = "sqlite"
	}
	if cfg.Database.SQLitePath == "" {
		cfg.Database.SQLitePath = "./data/finance.db"
	}

	if cfg.JWT.ExpireHours == 0 {
		cfg.JWT.ExpireHours = 24
	}
	if cfg.JWT.Issuer == "" {
		cfg.JWT.Issuer = "finance-system"
	}

	if cfg.Log.Level == "" {
		cfg.Log.Level = "info"
	}
	if cfg.Log.Format == "" {
		cfg.Log.Format = "json"
	}
	if cfg.Log.Output == "" {
		cfg.Log.Output = "stdout"
	}
}

// GetMySQLDSN 获取MySQL连接字符串
func (c *DatabaseConfig) GetMySQLDSN() string {
	return fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?charset=%s&parseTime=true&loc=Local",
		c.MySQL.Username,
		c.MySQL.Password,
		c.MySQL.Host,
		c.MySQL.Port,
		c.MySQL.Database,
		c.MySQL.Charset,
	)
}

// GetAddr 获取HTTP服务地址
func (c *ServerConfig) GetAddr() string {
	return fmt.Sprintf(":%d", c.HTTPPort)
}

// GetGRPCAddr 获取gRPC服务地址
func (c *ServerConfig) GetGRPCAddr() string {
	return fmt.Sprintf(":%d", c.GRPCPort)
}

// LoadFromEnv 从环境变量加载配置（用于容器化部署）
func LoadFromEnv() *Config {
	cfg := &Config{}

	// Server
	cfg.Server.HTTPPort = getEnvInt("HTTP_PORT", 8080)
	cfg.Server.GRPCPort = getEnvInt("GRPC_PORT", 9090)
	cfg.Server.Mode = getEnvString("SERVER_MODE", "release")
	cfg.Server.ReadTimeout = getEnvInt("READ_TIMEOUT", 30)
	cfg.Server.WriteTimeout = getEnvInt("WRITE_TIMEOUT", 30)

	// Database
	cfg.Database.Driver = getEnvString("DB_DRIVER", "sqlite")
	cfg.Database.SQLitePath = getEnvString("SQLITE_PATH", "./data/finance.db")
	cfg.Database.MySQL.Host = getEnvString("MYSQL_HOST", "localhost")
	cfg.Database.MySQL.Port = getEnvInt("MYSQL_PORT", 3306)
	cfg.Database.MySQL.Username = getEnvString("MYSQL_USER", "root")
	cfg.Database.MySQL.Password = getEnvString("MYSQL_PASSWORD", "")
	cfg.Database.MySQL.Database = getEnvString("MYSQL_DATABASE", "finance")
	cfg.Database.MySQL.Charset = getEnvString("MYSQL_CHARSET", "utf8mb4")
	cfg.Database.MySQL.MaxIdleConns = getEnvInt("MYSQL_MAX_IDLE_CONNS", 10)
	cfg.Database.MySQL.MaxOpenConns = getEnvInt("MYSQL_MAX_OPEN_CONNS", 100)

	// JWT
	cfg.JWT.Secret = getEnvString("JWT_SECRET", "your-secret-key-change-in-production")
	cfg.JWT.ExpireHours = getEnvInt("JWT_EXPIRE_HOURS", 24)
	cfg.JWT.Issuer = getEnvString("JWT_ISSUER", "finance-system")

	// Log
	cfg.Log.Level = getEnvString("LOG_LEVEL", "info")
	cfg.Log.Format = getEnvString("LOG_FORMAT", "json")
	cfg.Log.Output = getEnvString("LOG_OUTPUT", "stdout")

	// CORS
	cfg.CORS.AllowedOrigins = strings.Split(getEnvString("CORS_ORIGINS", "*"), ",")
	cfg.CORS.AllowCredentials = getEnvBool("CORS_CREDENTIALS", true)

	Cfg = cfg
	return cfg
}

func getEnvString(key, defaultVal string) string {
	if val := os.Getenv(key); val != "" {
		return val
	}
	return defaultVal
}

func getEnvInt(key string, defaultVal int) int {
	if val := os.Getenv(key); val != "" {
		if i, err := strconv.Atoi(val); err == nil {
			return i
		}
	}
	return defaultVal
}

func getEnvBool(key string, defaultVal bool) bool {
	if val := os.Getenv(key); val != "" {
		if b, err := strconv.ParseBool(val); err == nil {
			return b
		}
	}
	return defaultVal
}

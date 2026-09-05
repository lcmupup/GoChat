package config

import (
	"os"

	"gopkg.in/yaml.v3"
)

type Config struct {
	Server   ServerConfig   `yaml:"server"`
	MySQL    MySQLConfig    `yaml:"mysql"`
	Redis    RedisConfig    `yaml:"redis"`
	RabbitMQ RabbitMQConfig `yaml:"rabbitmq"`
	JWT      JWTConfig      `yaml:"jwt"`
	File     FileConfig     `yaml:"file"`
}

type ServerConfig struct {
	Port      int    `yaml:"port"`
	PprofPort int    `yaml:"pprof_port"`
	WsPath    string `yaml:"ws_path"` // WebSocket 端点的 URL 路径
	UploadDir string `yaml:"upload_dir"`
}

type MySQLConfig struct {
	Host     string `yaml:"host"`
	Port     int    `yaml:"port"`
	User     string `yaml:"user"`
	Password string `yaml:"password"`
	DBName   string `yaml:"db_name"`
}

type RedisConfig struct {
	Addr     string `yaml:"addr"`
	Password string `yaml:"password"`
	DB       int    `yaml:"db"`
}

type RabbitMQConfig struct {
	URL string `yaml:"url"`
}

type JWTConfig struct {
	Secret         string `yaml:"secret"`
	AccessExpHours int    `yaml:"access_exp_hours"`
	RefreshExpDays int    `yaml:"refresh_exp_days"`
}

type FileConfig struct {
	MaxSizeMB   int      `yaml:"max_size_mb"`
	AllowedExts []string `yaml:"allowed_exts"`
	UploadDir   string   `yaml:"upload_dir"`
}

func LoadConfig(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	data = []byte(os.ExpandEnv(string(data)))
	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, err
	}
	return &cfg, nil
}

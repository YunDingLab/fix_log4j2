package config

import (
	"fmt"
	"io"
	"os"

	"github.com/YunDingLab/fix_log4j2/internal/logs"
	"github.com/natefinch/lumberjack"
	"go.uber.org/zap/zapcore"
	"gopkg.in/yaml.v3"
)

var std *Config

// Config 。
type Config struct {
	MainConf MainConfig        `yaml:"main"`
	LogConf  LogConfig         `yaml:"logger"`
	Clue     VulnerabilityClue `yaml:"clue"`
}

// MainConfig 住配置
type MainConfig struct {
	TmpDir     string `yaml:"tmp_dir"`
	KubeConfig string `yaml:"kubeConfig"`
}

// LogConfig 。
type LogConfig struct {
	Level             zapcore.Level `yaml:"level"`
	lumberjack.Logger `yaml:",inline"`
}

type VulnerabilityClue struct {
	Images []string `yaml:"images"`
}

// LoadConfig .
func LoadConfig(filepath string) (*Config, error) {
	cfg := &Config{}
	f, err := os.Open(filepath)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	err = yaml.NewDecoder(f).Decode(cfg)
	if err != nil {
		return nil, err
	}

	if std == nil {
		std = cfg
	}

	return cfg, nil
}

// Conf .
func Conf() *Config {
	if std == nil {
		panic(fmt.Errorf("config is nil"))
	}
	return std
}

// LoadYamlLocalFile .
func LoadYamlLocalFile(file string, cfg interface{}) error {
	f, err := os.Open(file)
	if err != nil {
		logs.Errorf("[config] laod %s failed, %s", file, err)
		return err
	}

	err = yaml.NewDecoder(f).Decode(cfg)
	if err != nil {
		logs.Errorf("[config] decode %s failed, %s", file, err)
		return err
	}

	return nil
}

// LoadYamlReader .
func LoadYamlReader(r io.Reader, cfg interface{}) error {
	err := yaml.NewDecoder(r).Decode(cfg)
	if err != nil {
		logs.Errorf("[config] decode %T failed, %s", r, err)
		return err
	}

	return nil
}

package config

import (
	"flag"
	"gopkg.in/yaml.v3"
	"os"
	"runtime"
	"sync"
	"time"
)

var cfg *Config

type Config struct {
	App         string         `yaml:"app"`
	Server      string         `yaml:"server"`
	BathDir     string         `yaml:"bathDir"`
	LogConf     *LogConf       `yaml:"logger"`
	ServantList []*ServantConf `yaml:"servantList"`
	ClientConf  *ClientConf    `yaml:"client"`
}

type ServantConf struct {
	Name          string        `yaml:"name"`
	IP            string        `yaml:"ip"`
	Port          string        `yaml:"port"`
	ReadBufSize   int32         `yaml:"readBufSize"`
	WriteBufSize  int32         `yaml:"writeBufSize"`
	MaxRequestNum int32         `yaml:"maxRequestNum"`
	ReqTimeout    time.Duration `yaml:"reqTimeout"`
}

type ClientConf struct {
	MaxConnNum  int    `yaml:"maxConnNum"`
	EncoderName string `yaml:"encoderName"`
}

type LogConf struct {
	LogPath string `yaml:"logPath"`
	LogName string `yaml:"logName"`
}

var (
	configPath = flag.String("config", "config.yaml", "--config=config.yaml")
	initOnce   sync.Once
)

func initConfig() {
	flag.Parse()
	file, err := os.Open(*configPath)
	if err != nil {
		panic(err)
	}
	defer file.Close()

	decoder := yaml.NewDecoder(file)

	conf := &Config{}
	err = decoder.Decode(conf)
	if err != nil {
		panic(err)
	}
	cfg = conf

	for _, servant := range cfg.ServantList {
		checkServantConf(servant)
	}
	if cfg.ClientConf == nil {
		cfg.ClientConf = &ClientConf{}
	}
	checkClientConf(cfg.ClientConf)
}

func GetConfig() *Config {
	initOnce.Do(initConfig)
	if cfg == nil {
		panic("config not init")
	}
	return cfg
}

func GetServantConf(name string) *ServantConf {
	cfg = GetConfig()
	for _, servantConf := range cfg.ServantList {
		if servantConf.Name == name {
			return servantConf
		}
	}
	panic(name + " config not found")
}

func GetClientConf() *ClientConf {
	cfg = GetConfig()
	return cfg.ClientConf
}

func checkServantConf(cfg *ServantConf) {
	if cfg.MaxRequestNum <= 0 {
		cfg.MaxRequestNum = 10000
	}
	cfg.ReqTimeout = parserTimeDuration(cfg.ReqTimeout, time.Millisecond, 0)
}

func checkClientConf(cfg *ClientConf) {
	if cfg.MaxConnNum <= 0 {
		cfg.MaxConnNum = runtime.NumCPU()
	}
	if cfg.EncoderName == "" {
		cfg.EncoderName = "json"
	}
}

func parserTimeDuration(num time.Duration, pec time.Duration, defValue time.Duration) time.Duration {
	if num == 0 {
		return defValue
	}
	return num * pec
}

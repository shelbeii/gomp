package gomp

import (
	"os"
	"sync"
	"sync/atomic"

	"gopkg.in/yaml.v3"
)

// GompConfig 框架配置（导出类型，供外部包使用 SetConfig）
type GompConfig struct {
	EnableSQLPrint    bool `yaml:"enableSqlPrint"`
	AllowGlobalUpdate bool `yaml:"allowGlobalUpdate"`
	AllowGlobalDelete bool `yaml:"allowGlobalDelete"`
	// SaveBatchSize 批量插入每批大小，默认 100，有效范围 [1, 5000]
	SaveBatchSize int `yaml:"saveBatchSize"`
	// PageMaxSize 分页查询每页最大条数，默认 1000，有效范围 [1, 10000]
	PageMaxSize int `yaml:"pageMaxSize"`
}

type configWrapper struct {
	Gomp GompConfig `yaml:"gomp"`
}

// 使用 Go 1.19+ atomic.Pointer 替代 unsafe.Pointer，类型安全且无 GC 隐患
var (
	configPtr atomic.Pointer[configWrapper]
	configMu  sync.Mutex // 仅写时加锁，防止并发写互相覆盖
)

func init() {
	defaultCfg := &configWrapper{
		Gomp: GompConfig{
			SaveBatchSize: 100,
			PageMaxSize:   1000,
		},
	}
	configPtr.Store(defaultCfg)
}

// getConfig 并发安全地获取当前配置（无锁读）
func getConfig() *configWrapper {
	return configPtr.Load()
}

// normalizeCfg 校正配置边界值
func normalizeCfg(cfg *GompConfig) {
	if cfg.SaveBatchSize < 1 {
		cfg.SaveBatchSize = 100
	} else if cfg.SaveBatchSize > 5000 {
		cfg.SaveBatchSize = 5000
	}
	if cfg.PageMaxSize < 1 {
		cfg.PageMaxSize = 1000
	} else if cfg.PageMaxSize > 10000 {
		cfg.PageMaxSize = 10000
	}
}

// InitConfig 从 YAML 文件初始化配置（并发安全）
func InitConfig(filePath string) error {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return err
	}
	newCfg := &configWrapper{}
	if err = yaml.Unmarshal(data, newCfg); err != nil {
		return err
	}
	normalizeCfg(&newCfg.Gomp)
	configMu.Lock()
	configPtr.Store(newCfg)
	configMu.Unlock()
	return nil
}

// SetConfig 通过代码直接设置配置（并发安全，适用于不使用 yaml 文件的场景）
func SetConfig(cfg GompConfig) {
	normalizeCfg(&cfg)
	configMu.Lock()
	configPtr.Store(&configWrapper{Gomp: cfg})
	configMu.Unlock()
}

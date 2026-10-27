package repository

import (
	"errors"

	"github.com/YGeni/ruoyi-go/server-go/config"
	"github.com/YGeni/ruoyi-go/server-go/internal/model/entity"
	"github.com/YGeni/ruoyi-go/server-go/pkg/db"
	"github.com/YGeni/ruoyi-go/server-go/pkg/redisx"

	"gorm.io/gorm"
)

// SelectConfigValueByKey 读取 sys_config 的配置值，等价于 SysConfigServiceImpl 的
// selectConfigByKey。缓存键为 sys_config:<configKey>（与 Java 一致）。
//
// DIFF(review): Java 在启动时用 @PostConstruct 把全部配置灌进 Redis，并只缓存
// status=0 的记录（SysConfigServiceImpl.java:35-77）；Go 侧改为按需加载并缓存，
// 首次读库后写入相同 key，后续与 Java 端命中同一份缓存。
func SelectConfigValueByKey(key string) (string, bool, error) {
	if v, ok, err := redisx.Get(config.SysConfigKey + key); err == nil && ok {
		return v, true, nil
	} else if err != nil {
		return "", false, err
	}
	var c entity.SysConfig
	err := db.DB.Where("config_key = ?", key).First(&c).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return "", false, nil
	}
	if err != nil {
		return "", false, err
	}
	if err := redisx.Set(config.SysConfigKey+key, c.ConfigValue, 0); err != nil {
		return "", false, err
	}
	return c.ConfigValue, true, nil
}

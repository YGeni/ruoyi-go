package repository

import (
	"errors"

	"github.com/YGeni/ruoyi-go/server-go/config"
	"github.com/YGeni/ruoyi-go/server-go/internal/model/entity"
	"github.com/YGeni/ruoyi-go/server-go/internal/model/query"
	"github.com/YGeni/ruoyi-go/server-go/pkg/db"
	"github.com/YGeni/ruoyi-go/server-go/pkg/page"
	"github.com/YGeni/ruoyi-go/server-go/pkg/redisx"

	"gorm.io/gorm"
)

// SelectConfigList 分页查询参数配置，等价 SysConfigMapper.xml:33-50。
//
// Java：
//
//	select ... from sys_config
//	where 1=1 [and config_name like ?] [and config_type=?] [and config_key like ?]
//
// 参数**没有** @DataScope 注解（SysConfigServiceImpl.selectConfigList 无数据权限）。
func SelectConfigList(q *query.ConfigList, p *page.PageDomain) ([]entity.SysConfig, int64, error) {
	tx := db.DB.Table("sys_config")
	if q.ConfigName != "" {
		tx = tx.Where("config_name LIKE ?", "%"+q.ConfigName+"%")
	}
	if q.ConfigType != "" {
		tx = tx.Where("config_type = ?", q.ConfigType)
	}
	if q.ConfigKey != "" {
		tx = tx.Where("config_key LIKE ?", "%"+q.ConfigKey+"%")
	}
	var rows []entity.SysConfig
	total, err := page.Paginate(tx, p, "", "", &rows)
	if err != nil {
		return nil, 0, err
	}
	return rows, total, nil
}

// SelectConfigByID 按主键查参数，等价 SysConfigMapper.xml selectConfigById。
func SelectConfigByID(configID int64) (*entity.SysConfig, error) {
	var x entity.SysConfig
	err := db.DB.Where("config_id = ?", configID).First(&x).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &x, nil
}

// CheckConfigKeyUnique 参数键名是否可用，
// 等价 SysConfigServiceImpl.checkConfigKeyUnique（Spring 的 selectConfig 用等值查询）。
func CheckConfigKeyUnique(key string, excludeConfigID int64) (bool, error) {
	tx := db.DB.Table("sys_config").Where("config_key = ?", key)
	if excludeConfigID != 0 {
		tx = tx.Where("config_id != ?", excludeConfigID)
	}
	var n int64
	if err := tx.Count(&n).Error; err != nil {
		return false, err
	}
	return n == 0, nil
}

// InsertConfig 新增参数，等价 SysConfigMapper.xml insertConfig。
func InsertConfig(x *entity.SysConfig) error {
	return db.DB.Create(x).Error
}

// UpdateConfig 修改参数并清缓存，等价 SysConfigMapper.xml updateConfig。
//
// DIFF(review): Java 侧 updateConfig 之后由 SysConfigServiceImpl.clearConfigCache
// 删除该 key 的缓存；Go 侧在此一并处理，避免调用方忘记清缓存导致读到旧值。
func UpdateConfig(x *entity.SysConfig) error {
	sets := map[string]any{}
	if x.ConfigName != "" {
		sets["config_name"] = x.ConfigName
	}
	if x.ConfigKey != "" {
		sets["config_key"] = x.ConfigKey
	}
	if x.ConfigValue != "" {
		sets["config_value"] = x.ConfigValue
	}
	if x.ConfigType != "" {
		sets["config_type"] = x.ConfigType
	}
	if x.Remark != "" {
		sets["remark"] = x.Remark
	}
	if x.UpdateBy != "" {
		sets["update_by"] = x.UpdateBy
	}
	if len(sets) == 0 {
		return nil
	}
	if err := db.DB.Table("sys_config").Where("config_id = ?", x.ConfigID).Updates(sets).Error; err != nil {
		return err
	}
	return DeleteConfigCache([]string{x.ConfigKey})
}

// DeleteConfigByIDs **物理删除**参数，等价 SysConfigMapper.xml deleteConfigByIds。
func DeleteConfigByIDs(ids []int64) error {
	if len(ids) == 0 {
		return nil
	}
	var keys []string
	if err := db.DB.Table("sys_config").Where("config_id IN ?", ids).Pluck("config_key", &keys).Error; err != nil {
		return err
	}
	if err := db.DB.Table("sys_config").Where("config_id IN ?", ids).Delete(nil).Error; err != nil {
		return err
	}
	return DeleteConfigCache(keys)
}

// DeleteConfigCache 清理参数缓存，等价 SysConfigServiceImpl.clearConfigCache。
func DeleteConfigCache(keys []string) error {
	for _, k := range keys {
		if k == "" {
			continue
		}
		if err := redisx.Del(config.SysConfigKey + k); err != nil {
			return err
		}
	}
	return nil
}

// RefreshConfigCache 清空全部参数缓存并重新加载，
// 等价 SysConfigServiceImpl.loadingConfigCache + clearConfigCache。
//
// DELETE /system/config/refreshCache 走这里：先删掉所有 sys_config:* 键，
// 后续按需读库回填（Go 侧是懒加载，不需要像 Java 那样启动时全量灌入）。
func RefreshConfigCache() error {
	keys, err := redisx.Keys(config.SysConfigKey + "*")
	if err != nil {
		return err
	}
	for _, k := range keys {
		if err := redisx.Del(k); err != nil {
			return err
		}
	}
	return nil
}

// SelectConfigValueByKey 读取 sys_config 的配置值，等价于 SysConfigServiceImpl 的
// selectConfigByKey。缓存键为 sys_config:<configKey>（与 Java 一致）。
//
// DIFF(review): Java 在启动时用 @PostConstruct 把全部配置灌进 Redis，并只缓存
// status=0 的记录；Go 侧改为按需加载并缓存
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

package service

import (
	"errors"

	"github.com/YGeni/ruoyi-go/server-go/internal/model/entity"
	"github.com/YGeni/ruoyi-go/server-go/internal/model/query"
	"github.com/YGeni/ruoyi-go/server-go/internal/repository"
	"github.com/YGeni/ruoyi-go/server-go/pkg/page"
)

// SelectConfigList 参数列表，等价 SysConfigServiceImpl.selectConfigList。
//
// 参数**没有** @DataScope 注解（不在 DataScope 五个落点内）。
func SelectConfigList(q *query.ConfigList, p *page.PageDomain) ([]entity.SysConfig, int64, error) {
	return repository.SelectConfigList(q, p)
}

// SelectConfigByID 按主键查参数。
func SelectConfigByID(configID int64) (*entity.SysConfig, error) {
	return repository.SelectConfigByID(configID)
}

// SelectConfigValueByKey 按 key 取参数值（/system/config/configKey/{configKey}），
// 等价 SysConfigServiceImpl.selectConfigByKey。
func SelectConfigValueByKey(key string) (string, bool, error) {
	return repository.SelectConfigValueByKey(key)
}

// InsertConfig 新增参数，等价 SysConfigServiceImpl.insertConfig(L92-106)。
func InsertConfig(x *entity.SysConfig) error {
	ok, err := repository.CheckConfigKeyUnique(x.ConfigKey, 0)
	if err != nil {
		return err
	}
	if !ok {
		return errors.New("新增参数'" + x.ConfigName + "'失败，参数键名已存在")
	}
	return repository.InsertConfig(x)
}

// UpdateConfig 修改参数，等价 SysConfigServiceImpl.updateConfig(L113-127)。
func UpdateConfig(x *entity.SysConfig) error {
	ok, err := repository.CheckConfigKeyUnique(x.ConfigKey, x.ConfigID)
	if err != nil {
		return err
	}
	if !ok {
		return errors.New("修改参数'" + x.ConfigName + "'失败，参数键名已存在")
	}
	return repository.UpdateConfig(x)
}

// DeleteConfigByIDs 删除参数，等价 SysConfigServiceImpl.deleteConfigByIds(L134-141)。
//
// 前置校验：config_type = 'Y'（系统内置）不允许删除。
func DeleteConfigByIDs(ids []int64) error {
	for _, id := range ids {
		c, err := repository.SelectConfigByID(id)
		if err != nil {
			return err
		}
		if c != nil && c.ConfigType == "Y" {
			return errors.New("内置参数不能删除")
		}
	}
	return repository.DeleteConfigByIDs(ids)
}

// RefreshConfigCache 刷新参数缓存（DELETE /system/config/refreshCache），
// 等价 SysConfigServiceImpl.resetConfigCache(L148-154)。
func RefreshConfigCache() error {
	return repository.RefreshConfigCache()
}

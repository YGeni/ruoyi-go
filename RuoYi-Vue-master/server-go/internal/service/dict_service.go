package service

import (
	"errors"

	"github.com/YGeni/ruoyi-go/server-go/internal/model/entity"
	"github.com/YGeni/ruoyi-go/server-go/internal/model/query"
	"github.com/YGeni/ruoyi-go/server-go/internal/repository"
	"github.com/YGeni/ruoyi-go/server-go/pkg/page"
)

// SelectDictTypeList 字典类型列表，等价 SysDictTypeServiceImpl.selectDictTypeList。
//
// 字典**没有** @DataScope 注解（不在 DataScope 五个落点内）。
func SelectDictTypeList(q *query.DictTypeList, p *page.PageDomain) ([]entity.SysDictType, int64, error) {
	return repository.SelectDictTypeList(q, p)
}

// SelectDictTypeAll 全部字典类型（optionselect 用），
// 等价 SysDictTypeServiceImpl.selectDictTypeAll。
func SelectDictTypeAll() ([]entity.SysDictType, error) {
	return repository.SelectDictTypeAll()
}

// SelectDictTypeByID 按主键查字典类型。
func SelectDictTypeByID(dictID int64) (*entity.SysDictType, error) {
	return repository.SelectDictTypeByID(dictID)
}

// InsertDictType 新增字典类型，等价 SysDictTypeServiceImpl.insertDictType(L73-86)。
func InsertDictType(x *entity.SysDictType) error {
	ok, err := repository.CheckDictTypeUnique(x.DictType, 0)
	if err != nil {
		return err
	}
	if !ok {
		return errors.New("新增字典'" + x.DictName + "'失败，字典类型已存在")
	}
	return repository.InsertDictType(x)
}

// UpdateDictType 修改字典类型，等价 SysDictTypeServiceImpl.updateDictType(L93-113)。
//
// Java 在更新成功后会 clearDictCache，Go 侧一并处理。
func UpdateDictType(x *entity.SysDictType) error {
	ok, err := repository.CheckDictTypeUnique(x.DictType, x.DictID)
	if err != nil {
		return err
	}
	if !ok {
		return errors.New("修改字典'" + x.DictName + "'失败，字典类型已存在")
	}
	if err := repository.UpdateDictType(x); err != nil {
		return err
	}
	return repository.DeleteDictCache([]string{x.DictType})
}

// DeleteDictTypeByIDs 删除字典类型，等价 SysDictTypeServiceImpl.deleteDictTypeByIds(L120-127)。
func DeleteDictTypeByIDs(ids []int64) error {
	keys := make([]string, 0, len(ids))
	for _, id := range ids {
		t, err := repository.SelectDictTypeByID(id)
		if err != nil {
			return err
		}
		if t != nil && t.DictType != "" {
			keys = append(keys, t.DictType)
		}
	}
	if err := repository.DeleteDictTypeByIDs(ids); err != nil {
		return err
	}
	return repository.DeleteDictCache(keys)
}

// RefreshDictCache 刷新字典缓存（DELETE /system/dict/type/refreshCache），
// 等价 SysDictTypeServiceImpl.resetDictCache + loadingDictCache。
func RefreshDictCache() error {
	all, err := repository.SelectDictTypeAll()
	if err != nil {
		return err
	}
	keys := make([]string, 0, len(all))
	for i := range all {
		keys = append(keys, all[i].DictType)
	}
	return repository.DeleteDictCache(keys)
}

// SelectDictDataList 字典数据列表，等价 SysDictDataServiceImpl.selectDictDataList。
func SelectDictDataList(q *query.DictDataList, p *page.PageDomain) ([]entity.SysDictData, int64, error) {
	return repository.SelectDictDataList(q, p)
}

// SelectDictDataByType 按字典类型查数据（/system/dict/data/type/{dictType}），
// 等价 SysDictDataServiceImpl.selectDictDataByType。
func SelectDictDataByType(dictType string) ([]entity.SysDictData, error) {
	return repository.SelectDictDataByType(dictType)
}

// SelectDictDataByCode 按主键查字典数据。
func SelectDictDataByCode(dictCode int64) (*entity.SysDictData, error) {
	return repository.SelectDictDataByCode(dictCode)
}

// InsertDictData 新增字典数据，等价 SysDictDataServiceImpl.insertDictData。
func InsertDictData(x *entity.SysDictData) error {
	if err := repository.InsertDictData(x); err != nil {
		return err
	}
	return repository.DeleteDictCache([]string{x.DictType})
}

// UpdateDictData 修改字典数据，等价 SysDictDataServiceImpl.updateDictData。
func UpdateDictData(x *entity.SysDictData) error {
	if err := repository.UpdateDictData(x); err != nil {
		return err
	}
	return repository.DeleteDictCache([]string{x.DictType})
}

// DeleteDictDataByCodes 删除字典数据，等价 SysDictDataServiceImpl.deleteDictDataByIds。
func DeleteDictDataByCodes(codes []int64) error {
	keys := make([]string, 0, len(codes))
	for _, c := range codes {
		x, err := repository.SelectDictDataByCode(c)
		if err != nil {
			return err
		}
		if x != nil && x.DictType != "" {
			keys = append(keys, x.DictType)
		}
	}
	if err := repository.DeleteDictDataByCodes(codes); err != nil {
		return err
	}
	return repository.DeleteDictCache(keys)
}

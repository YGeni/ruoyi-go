package repository

import (
	"errors"
	"time"

	"github.com/YGeni/ruoyi-go/server-go/config"
	"github.com/YGeni/ruoyi-go/server-go/internal/model/entity"
	"github.com/YGeni/ruoyi-go/server-go/internal/model/query"
	"github.com/YGeni/ruoyi-go/server-go/pkg/db"
	"github.com/YGeni/ruoyi-go/server-go/pkg/page"
	"github.com/YGeni/ruoyi-go/server-go/pkg/redisx"

	"gorm.io/gorm"
)

// ---------------------------------------------------------------- 字典类型

// dictTypeListQuery 构造 selectDictTypeList（SysDictTypeMapper.xml）。
//
// Java：
//
//	select dict_id, dict_name, dict_type, status, create_by, create_time, remark
//	from sys_dict_type
//	where 1=1 [and dict_name like ?] [and dict_type like ?] [and status=?]
//
// 字典**没有** @DataScope 注解（SysDictTypeServiceImpl.selectDictTypeList 无数据权限）。
func dictTypeListQuery(q *query.DictTypeList) *gorm.DB {
	tx := db.DB.Table("sys_dict_type")
	if q.DictName != "" {
		tx = tx.Where("dict_name LIKE ?", "%"+q.DictName+"%")
	}
	if q.DictType != "" {
		tx = tx.Where("dict_type LIKE ?", "%"+q.DictType+"%")
	}
	if q.Status != "" {
		tx = tx.Where("status = ?", q.Status)
	}
	return tx
}

// SelectDictTypeList 分页查询字典类型，等价 SysDictTypeMapper.xml selectDictTypeList。
func SelectDictTypeList(q *query.DictTypeList, p *page.PageDomain) ([]entity.SysDictType, int64, error) {
	var rows []entity.SysDictType
	total, err := page.Paginate(dictTypeListQuery(q), p, "", "", &rows)
	if err != nil {
		return nil, 0, err
	}
	return rows, total, nil
}

// SelectDictTypeAll 全部字典类型（optionselect 用），
// 等价 SysDictTypeMapper.xml selectDictTypeAll。
func SelectDictTypeAll() ([]entity.SysDictType, error) {
	var rows []entity.SysDictType
	err := db.DB.Find(&rows).Error
	return rows, err
}

// SelectDictTypeByID 按主键查字典类型，等价 SysDictTypeMapper.xml selectDictTypeById。
func SelectDictTypeByID(dictID int64) (*entity.SysDictType, error) {
	var x entity.SysDictType
	err := db.DB.Where("dict_id = ?", dictID).First(&x).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &x, nil
}

// CheckDictTypeUnique 字典类型是否可用，等价 SysDictTypeServiceImpl.checkDictTypeUnique。
func CheckDictTypeUnique(dictType string, excludeDictID int64) (bool, error) {
	tx := db.DB.Table("sys_dict_type").Where("dict_type = ?", dictType)
	if excludeDictID != 0 {
		tx = tx.Where("dict_id != ?", excludeDictID)
	}
	var n int64
	if err := tx.Count(&n).Error; err != nil {
		return false, err
	}
	return n == 0, nil
}

// InsertDictType 新增字典类型，等价 SysDictTypeMapper.xml insertDictType。
func InsertDictType(x *entity.SysDictType) error {
	return db.DB.Create(x).Error
}

// UpdateDictType 修改字典类型，等价 SysDictTypeMapper.xml updateDictType 的 <set>。
func UpdateDictType(x *entity.SysDictType) error {
	sets := map[string]any{"update_time": time.Now()}
	if x.DictName != "" {
		sets["dict_name"] = x.DictName
	}
	if x.DictType != "" {
		sets["dict_type"] = x.DictType
	}
	if x.Status != "" {
		sets["status"] = x.Status
	}
	if x.Remark != "" {
		sets["remark"] = x.Remark
	}
	if x.UpdateBy != "" {
		sets["update_by"] = x.UpdateBy
	}
	return db.DB.Table("sys_dict_type").Where("dict_id = ?", x.DictID).Updates(sets).Error
}

// DeleteDictTypeByIDs **物理删除**字典类型，等价 SysDictTypeMapper.xml deleteDictTypeByIds。
func DeleteDictTypeByIDs(ids []int64) error {
	if len(ids) == 0 {
		return nil
	}
	return db.DB.Table("sys_dict_type").Where("dict_id IN ?", ids).Delete(nil).Error
}

// DeleteDictCache 清理字典缓存，等价 SysDictTypeServiceImpl.resetDictCache。
//
// Java 侧按 sys_dict:<dictType> 逐 key 删除（DictUtils.clearDictCache）。
func DeleteDictCache(dictTypes []string) error {
	for _, t := range dictTypes {
		if err := redisx.Del(config.SysDictKey + t); err != nil {
			return err
		}
	}
	return nil
}

// ---------------------------------------------------------------- 字典数据

// SelectDictDataList 分页查询字典数据，等价 SysDictDataMapper.xml selectDictDataList。
//
// 排序固定 dict_sort asc（Java 写死在 SQL 末尾）。
func SelectDictDataList(q *query.DictDataList, p *page.PageDomain) ([]entity.SysDictData, int64, error) {
	tx := db.DB.Table("sys_dict_data").Order("dict_sort asc")
	if q.DictType != "" {
		tx = tx.Where("dict_type = ?", q.DictType)
	}
	if q.DictLabel != "" {
		tx = tx.Where("dict_label LIKE ?", "%"+q.DictLabel+"%")
	}
	if q.Status != "" {
		tx = tx.Where("status = ?", q.Status)
	}
	var rows []entity.SysDictData
	total, err := page.Paginate(tx, p, "", "", &rows)
	if err != nil {
		return nil, 0, err
	}
	return rows, total, nil
}

// SelectDictDataByType 按字典类型查数据（/system/dict/data/type/{dictType}），
// 等价 SysDictDataMapper.xml selectDictDataByType。
//
// DIFF(review): Java 侧该接口结果会被 DictUtils 缓存（sys_dict:<dictType>）；
// Go 侧同样先读缓存，未命中再查库并回填，键名与 Java 一致。
func SelectDictDataByType(dictType string) ([]entity.SysDictData, error) {
	if dictType == "" {
		return nil, nil
	}
	key := config.SysDictKey + dictType
	var rows []entity.SysDictData
	if ok, err := redisx.GetObject(key, &rows); err == nil && ok {
		return rows, nil
	}
	if err := db.DB.Where("dict_type = ?", dictType).Where("status = ?", "0").
		Order("dict_sort asc").Find(&rows).Error; err != nil {
		return nil, err
	}
	if rows == nil {
		rows = []entity.SysDictData{}
	}
	_ = redisx.SetObject(key, rows, 0)
	return rows, nil
}

// SelectDictDataByCode 按主键查字典数据，等价 SysDictDataMapper.xml selectDictDataById。
func SelectDictDataByCode(dictCode int64) (*entity.SysDictData, error) {
	var x entity.SysDictData
	err := db.DB.Where("dict_code = ?", dictCode).First(&x).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &x, nil
}

// InsertDictData 新增字典数据，等价 SysDictDataMapper.xml insertDictData。
func InsertDictData(x *entity.SysDictData) error {
	return db.DB.Create(x).Error
}

// UpdateDictData 修改字典数据，等价 SysDictDataMapper.xml updateDictData 的 <set>。
func UpdateDictData(x *entity.SysDictData) error {
	sets := map[string]any{"update_time": time.Now()}
	if x.DictSort != 0 {
		sets["dict_sort"] = x.DictSort
	}
	if x.DictLabel != "" {
		sets["dict_label"] = x.DictLabel
	}
	if x.DictValue != "" {
		sets["dict_value"] = x.DictValue
	}
	if x.DictType != "" {
		sets["dict_type"] = x.DictType
	}
	if x.CssClass != "" {
		sets["css_class"] = x.CssClass
	}
	if x.ListClass != "" {
		sets["list_class"] = x.ListClass
	}
	if x.IsDefault != "" {
		sets["is_default"] = x.IsDefault
	}
	if x.Status != "" {
		sets["status"] = x.Status
	}
	if x.Remark != "" {
		sets["remark"] = x.Remark
	}
	if x.UpdateBy != "" {
		sets["update_by"] = x.UpdateBy
	}
	return db.DB.Table("sys_dict_data").Where("dict_code = ?", x.DictCode).Updates(sets).Error
}

// DeleteDictDataByCodes **物理删除**字典数据，
// 等价 SysDictDataMapper.xml deleteDictDataByIds。
func DeleteDictDataByCodes(codes []int64) error {
	if len(codes) == 0 {
		return nil
	}
	return db.DB.Table("sys_dict_data").Where("dict_code IN ?", codes).Delete(nil).Error
}

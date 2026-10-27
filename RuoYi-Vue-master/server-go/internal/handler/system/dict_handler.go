package system

import (
	"net/http"
	"strconv"

	"github.com/YGeni/ruoyi-go/server-go/internal/model/entity"
	"github.com/YGeni/ruoyi-go/server-go/internal/model/query"
	"github.com/YGeni/ruoyi-go/server-go/internal/service"
	"github.com/YGeni/ruoyi-go/server-go/pkg/page"
	"github.com/YGeni/ruoyi-go/server-go/pkg/result"

	"github.com/gin-gonic/gin"
)

// DictHandler 字典管理，对应 SysDictTypeController.java + SysDictDataController.java。
//
// 字典**不在 DataScope 五个落点内**（两个 ServiceImpl 的 selectXxxList 均无注解）。
type DictHandler struct{}

// ---------------------------------------------------------------- 字典类型

// TypeList GET /system/dict/type/list
func (DictHandler) TypeList(c *gin.Context) {
	if _, ok := loginUser(c); !ok {
		return
	}
	var q query.DictTypeList
	if err := c.ShouldBindQuery(&q); err != nil {
		fail(c, err)
		return
	}
	rows, total, err := service.SelectDictTypeList(&q, page.FromRequest(c))
	if err != nil {
		fail(c, err)
		return
	}
	if rows == nil {
		rows = []entity.SysDictType{}
	}
	c.JSON(http.StatusOK, result.DataTable(rows, total))
}

// TypeOptionSelect GET /system/dict/type/optionselect
func (DictHandler) TypeOptionSelect(c *gin.Context) {
	if _, ok := loginUser(c); !ok {
		return
	}
	rows, err := service.SelectDictTypeAll()
	if err != nil {
		fail(c, err)
		return
	}
	if rows == nil {
		rows = []entity.SysDictType{}
	}
	result.OK(c, result.SuccessData(rows))
}

// TypeGetInfo GET /system/dict/type/:dictId
func (DictHandler) TypeGetInfo(c *gin.Context) {
	if _, ok := loginUser(c); !ok {
		return
	}
	id, err := strconv.ParseInt(c.Param("dictId"), 10, 64)
	if err != nil {
		fail(c, err)
		return
	}
	x, err := service.SelectDictTypeByID(id)
	if err != nil {
		fail(c, err)
		return
	}
	result.OK(c, result.SuccessData(x))
}

// TypeAdd POST /system/dict/type
func (DictHandler) TypeAdd(c *gin.Context) {
	if _, ok := loginUser(c); !ok {
		return
	}
	var x entity.SysDictType
	if err := c.ShouldBindJSON(&x); err != nil {
		fail(c, err)
		return
	}
	if err := service.InsertDictType(&x); err != nil {
		result.Fail(c, result.Warn(err.Error()))
		return
	}
	result.OK(c, result.Success())
}

// TypeEdit PUT /system/dict/type
func (DictHandler) TypeEdit(c *gin.Context) {
	if _, ok := loginUser(c); !ok {
		return
	}
	var x entity.SysDictType
	if err := c.ShouldBindJSON(&x); err != nil {
		fail(c, err)
		return
	}
	if err := service.UpdateDictType(&x); err != nil {
		result.Fail(c, result.Warn(err.Error()))
		return
	}
	result.OK(c, result.Success())
}

// TypeRemove DELETE /system/dict/type/:dictIds
func (DictHandler) TypeRemove(c *gin.Context) {
	if _, ok := loginUser(c); !ok {
		return
	}
	ids := parseIDs(c.Param("dictIds"))
	if len(ids) == 0 {
		result.Fail(c, result.Error("未选择要删除的字典类型"))
		return
	}
	if err := service.DeleteDictTypeByIDs(ids); err != nil {
		result.Fail(c, result.Warn(err.Error()))
		return
	}
	result.OK(c, result.Success())
}

// TypeRefreshCache DELETE /system/dict/type/refreshCache
//
// 等价 SysDictTypeController.java 的同名接口。
// 必须在 /system/dict/type/:dictIds **之前**注册。
func (DictHandler) TypeRefreshCache(c *gin.Context) {
	if _, ok := loginUser(c); !ok {
		return
	}
	if err := service.RefreshDictCache(); err != nil {
		fail(c, err)
		return
	}
	result.OK(c, result.Success())
}

// ---------------------------------------------------------------- 字典数据

// DataList GET /system/dict/data/list
func (DictHandler) DataList(c *gin.Context) {
	if _, ok := loginUser(c); !ok {
		return
	}
	var q query.DictDataList
	if err := c.ShouldBindQuery(&q); err != nil {
		fail(c, err)
		return
	}
	rows, total, err := service.SelectDictDataList(&q, page.FromRequest(c))
	if err != nil {
		fail(c, err)
		return
	}
	if rows == nil {
		rows = []entity.SysDictData{}
	}
	c.JSON(http.StatusOK, result.DataTable(rows, total))
}

// DataGetInfo GET /system/dict/data/:dictCode
func (DictHandler) DataGetInfo(c *gin.Context) {
	if _, ok := loginUser(c); !ok {
		return
	}
	code, err := strconv.ParseInt(c.Param("dictCode"), 10, 64)
	if err != nil {
		fail(c, err)
		return
	}
	x, err := service.SelectDictDataByCode(code)
	if err != nil {
		fail(c, err)
		return
	}
	result.OK(c, result.SuccessData(x))
}

// DataByType GET /system/dict/data/type/:dictType
//
// 等价 SysDictDataController.java:66-73。
// 必须在 /system/dict/data/:dictCode **之前**注册。
func (DictHandler) DataByType(c *gin.Context) {
	if _, ok := loginUser(c); !ok {
		return
	}
	rows, err := service.SelectDictDataByType(c.Param("dictType"))
	if err != nil {
		fail(c, err)
		return
	}
	if rows == nil {
		rows = []entity.SysDictData{}
	}
	result.OK(c, result.SuccessData(rows))
}

// DataAdd POST /system/dict/data
func (DictHandler) DataAdd(c *gin.Context) {
	if _, ok := loginUser(c); !ok {
		return
	}
	var x entity.SysDictData
	if err := c.ShouldBindJSON(&x); err != nil {
		fail(c, err)
		return
	}
	if err := service.InsertDictData(&x); err != nil {
		result.Fail(c, result.Warn(err.Error()))
		return
	}
	result.OK(c, result.Success())
}

// DataEdit PUT /system/dict/data
func (DictHandler) DataEdit(c *gin.Context) {
	if _, ok := loginUser(c); !ok {
		return
	}
	var x entity.SysDictData
	if err := c.ShouldBindJSON(&x); err != nil {
		fail(c, err)
		return
	}
	if err := service.UpdateDictData(&x); err != nil {
		result.Fail(c, result.Warn(err.Error()))
		return
	}
	result.OK(c, result.Success())
}

// DataRemove DELETE /system/dict/data/:dictCodes
func (DictHandler) DataRemove(c *gin.Context) {
	if _, ok := loginUser(c); !ok {
		return
	}
	codes := parseIDs(c.Param("dictCodes"))
	if len(codes) == 0 {
		result.Fail(c, result.Error("未选择要删除的字典数据"))
		return
	}
	if err := service.DeleteDictDataByCodes(codes); err != nil {
		result.Fail(c, result.Warn(err.Error()))
		return
	}
	result.OK(c, result.Success())
}

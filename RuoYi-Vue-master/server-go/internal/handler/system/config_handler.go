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

// ConfigHandler 参数配置，对应。
//
// 参数**不在 DataScope 五个落点内**（SysConfigServiceImpl.selectConfigList 无注解）。
type ConfigHandler struct{}

// List GET /system/config/list
func (ConfigHandler) List(c *gin.Context) {
	if _, ok := loginUser(c); !ok {
		return
	}
	var q query.ConfigList
	if err := c.ShouldBindQuery(&q); err != nil {
		fail(c, err)
		return
	}
	rows, total, err := service.SelectConfigList(&q, page.FromRequest(c))
	if err != nil {
		fail(c, err)
		return
	}
	if rows == nil {
		rows = []entity.SysConfig{}
	}
	c.JSON(http.StatusOK, result.DataTable(rows, total))
}

// GetInfo GET /system/config/:configId
func (ConfigHandler) GetInfo(c *gin.Context) {
	if _, ok := loginUser(c); !ok {
		return
	}
	id, err := strconv.ParseInt(c.Param("configId"), 10, 64)
	if err != nil {
		fail(c, err)
		return
	}
	x, err := service.SelectConfigByID(id)
	if err != nil {
		fail(c, err)
		return
	}
	result.OK(c, result.SuccessData(x))
}

// GetByKey GET /system/config/configKey/:configKey
//
// 等价：返回 sys_config 的 config_value 字符串。
//
// 注意路径冲突：/system/config/configKey/:configKey 必须注册在
// /system/config/:configId **之前**，否则 gin 会把 "configKey" 当成 configId。
func (ConfigHandler) GetByKey(c *gin.Context) {
	if _, ok := loginUser(c); !ok {
		return
	}
	key := c.Param("configKey")
	v, _, err := service.SelectConfigValueByKey(key)
	if err != nil {
		fail(c, err)
		return
	}
	result.OK(c, result.SuccessData(v))
}

// Add POST /system/config
func (ConfigHandler) Add(c *gin.Context) {
	if _, ok := loginUser(c); !ok {
		return
	}
	var x entity.SysConfig
	if err := c.ShouldBindJSON(&x); err != nil {
		fail(c, err)
		return
	}
	if err := service.InsertConfig(&x); err != nil {
		result.Fail(c, result.Warn(err.Error()))
		return
	}
	result.OK(c, result.Success())
}

// Edit PUT /system/config
func (ConfigHandler) Edit(c *gin.Context) {
	if _, ok := loginUser(c); !ok {
		return
	}
	var x entity.SysConfig
	if err := c.ShouldBindJSON(&x); err != nil {
		fail(c, err)
		return
	}
	if err := service.UpdateConfig(&x); err != nil {
		result.Fail(c, result.Warn(err.Error()))
		return
	}
	result.OK(c, result.Success())
}

// Remove DELETE /system/config/:configIds
func (ConfigHandler) Remove(c *gin.Context) {
	if _, ok := loginUser(c); !ok {
		return
	}
	ids := parseIDs(c.Param("configIds"))
	if len(ids) == 0 {
		result.Fail(c, result.Error("未选择要删除的参数"))
		return
	}
	if err := service.DeleteConfigByIDs(ids); err != nil {
		result.Fail(c, result.Warn(err.Error()))
		return
	}
	result.OK(c, result.Success())
}

// RefreshCache DELETE /system/config/refreshCache
//
// 同样要注意注册顺序：必须在 /system/config/:configIds **之前**。
func (ConfigHandler) RefreshCache(c *gin.Context) {
	if _, ok := loginUser(c); !ok {
		return
	}
	if err := service.RefreshConfigCache(); err != nil {
		fail(c, err)
		return
	}
	result.OK(c, result.Success())
}

// ConfigExport POST /system/config/export
//
// 等价：ExcelUtil 导出，不分页。
//
// 参数**不在 query 里**：前端 download（request.js:127-133）用 tansParams
// 把条件塞进 form-urlencoded 请求体，所以用 ShouldBind 而非 ShouldBindQuery。
func (ConfigHandler) ConfigExport(c *gin.Context) {
	if _, ok := loginUser(c); !ok {
		return
	}
	var q query.ConfigList
	if err := c.ShouldBind(&q); err != nil {
		fail(c, err)
		return
	}
	rows, _, err := service.SelectConfigList(&q, page.NoLimit())
	if err != nil {
		fail(c, err)
		return
	}
	headers, data := service.ConfigExportRows(rows)
	writeXlsx(c, "参数配置", headers, data)
}

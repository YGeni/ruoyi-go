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

// ConfigHandler 参数配置，对应 SysConfigController.java。
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
// 等价 SysConfigController.java:75-82：返回 sys_config 的 config_value 字符串。
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
// 等价 SysConfigController.java:104-111。
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

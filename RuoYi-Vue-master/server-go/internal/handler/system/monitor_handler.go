package system

import (
	"net/http"

	"github.com/YGeni/ruoyi-go/server-go/internal/model/entity"
	"github.com/YGeni/ruoyi-go/server-go/internal/service"
	"github.com/YGeni/ruoyi-go/server-go/pkg/result"

	"github.com/gin-gonic/gin"
)

// MonitorHandler 系统监控 - 在线用户，对应 SysUserOnlineController.java。
//
// Java 里该控制器在 com.ruoyi.web.controller.monitor 包下，
// Go 侧暂挂在 system 包（复用 system/common.go 的 loginUser 取值辅助）。
type MonitorHandler struct{}

// OnlineList GET /monitor/online/list
//
// 等价 SysUserOnlineController.java:43-70。
//
// 返回体是 **getDataTable**（{code,msg,rows,total}），不是 {data}：
// 前端 src/views/monitor/online/index.vue:80-81 读的是
//
//	onlineList.value = response.rows
//	total.value      = response.total
//
// 该接口**没有 startPage()**，Java 的 total 恒等于列表长度。
func (MonitorHandler) OnlineList(c *gin.Context) {
	if _, ok := loginUser(c); !ok {
		return
	}
	rows, err := service.SelectOnlineList(c.Query("ipaddr"), c.Query("userName"))
	if err != nil {
		fail(c, err)
		return
	}
	if rows == nil {
		rows = []entity.SysUserOnline{}
	}
	c.JSON(http.StatusOK, result.DataTable(rows, int64(len(rows))))
}

// ForceLogout DELETE /monitor/online/:tokenId
//
// 等价 SysUserOnlineController.java:78-82：删掉 login_tokens:<tokenId>。
//
// :tokenId 是 UUID 字符串（不是数字），强退后该 token 下一次请求会 401。
func (MonitorHandler) ForceLogout(c *gin.Context) {
	if _, ok := loginUser(c); !ok {
		return
	}
	if err := service.ForceLogout(c.Param("tokenId")); err != nil {
		fail(c, err)
		return
	}
	result.OK(c, result.Success())
}

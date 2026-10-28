package system

import (
	"net/http"

	"github.com/YGeni/ruoyi-go/server-go/internal/model/entity"
	"github.com/YGeni/ruoyi-go/server-go/internal/model/query"
	"github.com/YGeni/ruoyi-go/server-go/internal/service"
	"github.com/YGeni/ruoyi-go/server-go/pkg/excelx"
	"github.com/YGeni/ruoyi-go/server-go/pkg/page"
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

// ---------- 登录日志：对应 SysLogininforController.java ----------

// LogininforList GET /monitor/logininfor/list
//
// 等价 SysLogininforController.java:39-45：startPage() + getDataTable。
// 前端 logininfor/index.vue:159-160 读 response.rows / response.total。
func (MonitorHandler) LogininforList(c *gin.Context) {
	if _, ok := loginUser(c); !ok {
		return
	}
	var q query.LogininforList
	if err := c.ShouldBindQuery(&q); err != nil {
		fail(c, err)
		return
	}
	q.FillDateRange(c)
	rows, total, err := service.SelectLogininforList(&q, page.FromRequest(c))
	if err != nil {
		fail(c, err)
		return
	}
	if rows == nil {
		rows = []entity.SysLogininfor{}
	}
	c.JSON(http.StatusOK, result.DataTable(rows, total))
}

// LogininforExport POST /monitor/logininfor/export
//
// 等价 SysLogininforController.java:49-55：ExcelUtil 导出，不分页。
//
// 参数**不在 query 里**：前端 download（request.js:127-133）用 tansParams
// 把条件塞进了 form-urlencoded 请求体，因此用 ShouldBind 而非 ShouldBindQuery。
func (MonitorHandler) LogininforExport(c *gin.Context) {
	if _, ok := loginUser(c); !ok {
		return
	}
	var q query.LogininforList
	if err := c.ShouldBind(&q); err != nil {
		fail(c, err)
		return
	}
	q.FillDateRange(c)
	rows, _, err := service.SelectLogininforList(&q, page.NoLimit())
	if err != nil {
		fail(c, err)
		return
	}
	headers, data := service.LogininforExportRows(rows)
	writeXlsx(c, "登录日志", headers, data)
}

// LogininforRemove DELETE /monitor/logininfor/:infoIds
func (MonitorHandler) LogininforRemove(c *gin.Context) {
	if _, ok := loginUser(c); !ok {
		return
	}
	n, err := service.DeleteLogininforByIDs(parseIDs(c.Param("infoIds")))
	if err != nil {
		fail(c, err)
		return
	}
	result.OK(c, result.ToAjaxRows(int(n)))
}

// LogininforClean DELETE /monitor/logininfor/clean
//
// 等价 SysLogininforController.java:68-72：truncate sys_logininfor。
func (MonitorHandler) LogininforClean(c *gin.Context) {
	if _, ok := loginUser(c); !ok {
		return
	}
	if err := service.CleanLogininfor(); err != nil {
		fail(c, err)
		return
	}
	result.OK(c, result.Success())
}

// LogininforUnlock GET /monitor/logininfor/unlock/:userName
//
// 等价 SysLogininforController.java:76-81：只清 Redis 的登录失败计数，
// 不改 sys_user.status。
func (MonitorHandler) LogininforUnlock(c *gin.Context) {
	if _, ok := loginUser(c); !ok {
		return
	}
	if err := service.UnlockLogininfor(c.Param("userName")); err != nil {
		fail(c, err)
		return
	}
	result.OK(c, result.Success())
}

// ---------- 操作日志：对应 SysOperlogController.java ----------

// OperLogList GET /monitor/operlog/list
//
// 等价 SysOperlogController.java:35-41：startPage() + getDataTable。
// 前端 operlog/index.vue:194-195 读 response.rows / response.total。
func (MonitorHandler) OperLogList(c *gin.Context) {
	if _, ok := loginUser(c); !ok {
		return
	}
	var q query.OperLogList
	if err := c.ShouldBindQuery(&q); err != nil {
		fail(c, err)
		return
	}
	q.FillDateRange(c)
	rows, total, err := service.SelectOperLogList(&q, page.FromRequest(c))
	if err != nil {
		fail(c, err)
		return
	}
	if rows == nil {
		rows = []entity.SysOperLog{}
	}
	c.JSON(http.StatusOK, result.DataTable(rows, total))
}

// OperLogExport POST /monitor/operlog/export
func (MonitorHandler) OperLogExport(c *gin.Context) {
	if _, ok := loginUser(c); !ok {
		return
	}
	var q query.OperLogList
	if err := c.ShouldBind(&q); err != nil {
		fail(c, err)
		return
	}
	q.FillDateRange(c)
	rows, _, err := service.SelectOperLogList(&q, page.NoLimit())
	if err != nil {
		fail(c, err)
		return
	}
	headers, data := service.OperLogExportRows(rows)
	writeXlsx(c, "操作日志", headers, data)
}

// OperLogRemove DELETE /monitor/operlog/:operIds
func (MonitorHandler) OperLogRemove(c *gin.Context) {
	if _, ok := loginUser(c); !ok {
		return
	}
	n, err := service.DeleteOperLogByIDs(parseIDs(c.Param("operIds")))
	if err != nil {
		fail(c, err)
		return
	}
	result.OK(c, result.ToAjaxRows(int(n)))
}

// OperLogClean DELETE /monitor/operlog/clean
func (MonitorHandler) OperLogClean(c *gin.Context) {
	if _, ok := loginUser(c); !ok {
		return
	}
	if err := service.CleanOperLog(); err != nil {
		fail(c, err)
		return
	}
	result.OK(c, result.Success())
}

// writeXlsx 导出 xlsx。
//
// 响应头对不上就白导：前端 download 用 responseType: 'blob' 接收，
// blobValidate（ruoyi.js:226-228）的判定是 type !== 'application/json'，
// 所以 Content-Type 必须是 xlsx；文件名由调用方（页面）自己拼，
// 这里只给一个 Content-Disposition 兜底。
func writeXlsx(c *gin.Context, sheet string, headers []string, rows [][]any) {
	c.Header("Content-Type", excelx.ContentType)
	c.Header("Content-Disposition", "attachment; filename=export.xlsx")
	if err := excelx.Write(c.Writer, sheet, headers, rows); err != nil {
		fail(c, err)
		return
	}
}

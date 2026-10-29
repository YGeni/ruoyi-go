// Package result 复刻 Java 侧 AjaxResult / TableDataInfo 的返回体契约。
//
// 契约来源：
// -ruoyi-
//
//	AjaxResult extends HashMap，字段为 code / msg / data，data 为 null 时不输出。
//
// -ruoyi-
//   - 前端 RuoYi-Vue3 src/utils/request.js:76-109 只认这些 code。
//
// 因此 AjaxResult 建模为 map（与 Java 的 HashMap 一致），可像 Java 的 put() 一样
// 追加 token / user / roles / permissions 等动态字段。
package result

import (
	"net/http"

	"github.com/YGeni/ruoyi-go/server-go/config"

	"github.com/gin-gonic/gin"
)

// AjaxResult 等价于 Java 的 AjaxResult（HashMap）。
type AjaxResult map[string]any

const (
	keyCode = "code"
	keyMsg  = "msg"
	keyData = "data"
)

// Success 等价于 AjaxResult.success()：code=200, msg="操作成功"。
func Success() AjaxResult {
	return AjaxResult{keyCode: config.StatusSuccess, keyMsg: config.MsgOperateSuccess}
}

// SuccessMsg 等价于 AjaxResult.success(String msg)。
func SuccessMsg(msg string) AjaxResult {
	return AjaxResult{keyCode: config.StatusSuccess, keyMsg: msg}
}

// SuccessData 等价于 AjaxResult.success(Object data)：额外带上 data 字段。
func SuccessData(data any) AjaxResult {
	r := Success()
	r[keyData] = data
	return r
}

// ToAjaxRows 等价于 BaseController.toAjax(int rows)
//
//	return rows > 0 ? AjaxResult.success() : AjaxResult.error();
//
// 用于新增/修改/删除后按影响行数回包。
func ToAjaxRows(rows int) AjaxResult {
	if rows > 0 {
		return Success()
	}
	return Error(config.MsgOperateFailed)
}

// ToAjax 等价于 Java controller 里"捕获 ServiceException 返回 error(msg)、
// 否则返回 success()"的通用写法（如 SysUserController.add 的 toAjax(userService.insertUser(user))）。
//
// Go 侧 service 统一以 error 表达业务失败，因此这里只需判空。
func ToAjax(err error) AjaxResult {
	if err != nil {
		return Error(err.Error())
	}
	return Success()
}

// Warn 等价于 AjaxResult.warn()：code=601（前端 request.js 走 warning 分支）。
func Warn(msg string) AjaxResult {
	return AjaxResult{keyCode: config.StatusWarn, keyMsg: msg}
}

// Error 等价于 AjaxResult.error()：code=500。
func Error(msg string) AjaxResult {
	return AjaxResult{keyCode: config.StatusError, keyMsg: msg}
}

// ErrorCode 等价于 AjaxResult.error(int code, String msg)。
func ErrorCode(code int, msg string) AjaxResult {
	return AjaxResult{keyCode: code, keyMsg: msg}
}

// Put 等价于 Java 的 AjaxResult.put(k, v)，返回自身以支持链式调用。
func (r AjaxResult) Put(k string, v any) AjaxResult {
	r[k] = v
	return r
}

// TableDataInfo 等价于 Java 的 TableDataInfo：{total, rows, code, msg}。
type TableDataInfo struct {
	Total int64  `json:"total"`
	Rows  any    `json:"rows"`
	Code  int    `json:"code"`
	Msg   string `json:"msg"`
}

// DataTable 等价于 BaseController.getDataTable()：code=200, msg="查询成功"。
func DataTable(rows any, total int64) TableDataInfo {
	return TableDataInfo{Total: total, Rows: rows, Code: config.StatusSuccess, Msg: config.MsgQuerySuccess}
}

// OK 以 200 HTTP 状态码输出 AjaxResult。
func OK(c *gin.Context, r AjaxResult) {
	c.JSON(http.StatusOK, r)
}

// Fail 输出业务错误体（HTTP 仍为 200，由 code 表达失败，与 Java 一致）。
func Fail(c *gin.Context, r AjaxResult) {
	c.JSON(http.StatusOK, r)
}

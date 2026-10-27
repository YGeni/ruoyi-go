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

// NoticeHandler 通知公告，对应 SysNoticeController.java（含 3.9 新增的五个端点）。
//
// 公告**不在 DataScope 五个落点内**（SysNoticeServiceImpl.selectNoticeList 无注解）。
type NoticeHandler struct{}

// List GET /system/notice/list
func (NoticeHandler) List(c *gin.Context) {
	if _, ok := loginUser(c); !ok {
		return
	}
	var q query.NoticeList
	if err := c.ShouldBindQuery(&q); err != nil {
		fail(c, err)
		return
	}
	rows, total, err := service.SelectNoticeList(&q, page.FromRequest(c))
	if err != nil {
		fail(c, err)
		return
	}
	if rows == nil {
		rows = []entity.SysNotice{}
	}
	c.JSON(http.StatusOK, result.DataTable(rows, total))
}

// GetInfo GET /system/notice/:noticeId
//
// 等价 SysNoticeController.java:55-60：**无 @PreAuthorize**，仅需登录态。
func (NoticeHandler) GetInfo(c *gin.Context) {
	if _, ok := loginUser(c); !ok {
		return
	}
	id, err := strconv.ParseInt(c.Param("noticeId"), 10, 64)
	if err != nil {
		fail(c, err)
		return
	}
	x, err := service.SelectNoticeByID(id)
	if err != nil {
		fail(c, err)
		return
	}
	result.OK(c, result.SuccessData(x))
}

// Add POST /system/notice
func (NoticeHandler) Add(c *gin.Context) {
	lu, ok := loginUser(c)
	if !ok {
		return
	}
	var x entity.SysNotice
	if err := c.ShouldBindJSON(&x); err != nil {
		fail(c, err)
		return
	}
	x.CreateBy = usernameOf(lu)
	if err := service.InsertNotice(&x); err != nil {
		result.Fail(c, result.Warn(err.Error()))
		return
	}
	result.OK(c, result.Success())
}

// Edit PUT /system/notice
func (NoticeHandler) Edit(c *gin.Context) {
	lu, ok := loginUser(c)
	if !ok {
		return
	}
	var x entity.SysNotice
	if err := c.ShouldBindJSON(&x); err != nil {
		fail(c, err)
		return
	}
	x.UpdateBy = usernameOf(lu)
	if err := service.UpdateNotice(&x); err != nil {
		result.Fail(c, result.Warn(err.Error()))
		return
	}
	result.OK(c, result.Success())
}

// Remove DELETE /system/notice/:noticeIds
func (NoticeHandler) Remove(c *gin.Context) {
	if _, ok := loginUser(c); !ok {
		return
	}
	ids := parseIDs(c.Param("noticeIds"))
	if len(ids) == 0 {
		result.Fail(c, result.Error("未选择要删除的公告"))
		return
	}
	if err := service.DeleteNoticeByIDs(ids); err != nil {
		result.Fail(c, result.Warn(err.Error()))
		return
	}
	result.OK(c, result.Success())
}

// ---------------------------------------------------------------- 3.9 新增端点

// ListTop GET /system/notice/listTop
//
// 等价 SysNoticeController.java:89-97：返回最多 5 条正常公告（带当前用户已读标记），
// 并额外带上 unreadCount。
//
// 返回体：AjaxResult{code, msg, data: <公告列表>, unreadCount: <未读数>}。
// Java 在 controller 里 result.put("unreadCount", unreadCount)，
// Go 侧用 result.Put 组合，字段顺序不影响 JSON 语义。
//
// 必须在 /system/notice/:noticeId **之前**注册。
func (NoticeHandler) ListTop(c *gin.Context) {
	lu, ok := loginUser(c)
	if !ok {
		return
	}
	rows, unread, err := service.SelectNoticeListTop(lu.UserID)
	if err != nil {
		fail(c, err)
		return
	}
	if rows == nil {
		rows = []entity.SysNoticeWithRead{}
	}
	result.OK(c, result.SuccessData(rows).Put("unreadCount", unread))
}

// MarkRead POST /system/notice/markRead
//
// 等价 SysNoticeController.java:102-108：noticeId 来自 query 或 form。
func (NoticeHandler) MarkRead(c *gin.Context) {
	lu, ok := loginUser(c)
	if !ok {
		return
	}
	noticeID, _ := strconv.ParseInt(firstNonEmpty(c.Query("noticeId"), c.PostForm("noticeId")), 10, 64)
	if err := service.MarkNoticeRead(noticeID, lu.UserID); err != nil {
		fail(c, err)
		return
	}
	result.OK(c, result.Success())
}

// MarkReadAll POST /system/notice/markReadAll
//
// 等价 SysNoticeController.java:113-120：ids 是逗号分隔的字符串
// （Java 用 Convert.toLongArray(ids) 解析，见 add 说明）。
func (NoticeHandler) MarkReadAll(c *gin.Context) {
	lu, ok := loginUser(c)
	if !ok {
		return
	}
	ids := parseIDs(firstNonEmpty(c.Query("ids"), c.PostForm("ids")))
	if err := service.MarkNoticeReadAll(lu.UserID, ids); err != nil {
		fail(c, err)
		return
	}
	result.OK(c, result.Success())
}

// ReadUsers GET /system/notice/readUsers/list
//
// 等价 SysNoticeController.java:126-133：分页返回已读该公告的用户。
// 必须在 /system/notice/:noticeId **之前**注册。
func (NoticeHandler) ReadUsers(c *gin.Context) {
	if _, ok := loginUser(c); !ok {
		return
	}
	var q query.ReadUserList
	if err := c.ShouldBindQuery(&q); err != nil {
		fail(c, err)
		return
	}
	rows, total, err := service.SelectNoticeReadUsers(&q, page.FromRequest(c))
	if err != nil {
		fail(c, err)
		return
	}
	if rows == nil {
		rows = []entity.NoticeReadUser{}
	}
	c.JSON(http.StatusOK, result.DataTable(rows, total))
}

// firstNonEmpty 取第一个非空值：兼容前端用 query 或 form 提交参数。
func firstNonEmpty(vals ...string) string {
	for _, v := range vals {
		if v != "" {
			return v
		}
	}
	return ""
}

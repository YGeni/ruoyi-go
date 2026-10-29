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

// PostHandler 岗位管理，对应。
//
// 岗位**不在 DataScope 五个落点内**（SysPostServiceImpl.selectPostList 无注解），
// 因此列表接口不施加数据范围过滤。
type PostHandler struct{}

// List GET /system/post/list
func (PostHandler) List(c *gin.Context) {
	if _, ok := loginUser(c); !ok {
		return
	}
	var q query.PostList
	if err := c.ShouldBindQuery(&q); err != nil {
		fail(c, err)
		return
	}
	rows, total, err := service.SelectPostList(&q, page.FromRequest(c))
	if err != nil {
		fail(c, err)
		return
	}
	if rows == nil {
		rows = []entity.SysPost{}
	}
	c.JSON(http.StatusOK, result.DataTable(rows, total))
}

// OptionSelect GET /system/post/optionselect
//
// 等价的同名接口：返回全部岗位，不带分页。
func (PostHandler) OptionSelect(c *gin.Context) {
	if _, ok := loginUser(c); !ok {
		return
	}
	rows, err := service.SelectPostAll()
	if err != nil {
		fail(c, err)
		return
	}
	if rows == nil {
		rows = []entity.SysPost{}
	}
	result.OK(c, result.SuccessData(rows))
}

// GetInfo GET /system/post/:postId
func (PostHandler) GetInfo(c *gin.Context) {
	if _, ok := loginUser(c); !ok {
		return
	}
	id, err := strconv.ParseInt(c.Param("postId"), 10, 64)
	if err != nil {
		fail(c, err)
		return
	}
	x, err := service.SelectPostByID(id)
	if err != nil {
		fail(c, err)
		return
	}
	result.OK(c, result.SuccessData(x))
}

// Add POST /system/post
func (PostHandler) Add(c *gin.Context) {
	if _, ok := loginUser(c); !ok {
		return
	}
	var x entity.SysPost
	if err := c.ShouldBindJSON(&x); err != nil {
		fail(c, err)
		return
	}
	if err := service.InsertPost(&x); err != nil {
		result.Fail(c, result.Warn(err.Error()))
		return
	}
	result.OK(c, result.Success())
}

// Edit PUT /system/post
func (PostHandler) Edit(c *gin.Context) {
	if _, ok := loginUser(c); !ok {
		return
	}
	var x entity.SysPost
	if err := c.ShouldBindJSON(&x); err != nil {
		fail(c, err)
		return
	}
	if err := service.UpdatePost(&x); err != nil {
		result.Fail(c, result.Warn(err.Error()))
		return
	}
	result.OK(c, result.Success())
}

// Remove DELETE /system/post/:postIds
func (PostHandler) Remove(c *gin.Context) {
	if _, ok := loginUser(c); !ok {
		return
	}
	ids := parseIDs(c.Param("postIds"))
	if len(ids) == 0 {
		result.Fail(c, result.Error("未选择要删除的岗位"))
		return
	}
	if err := service.DeletePostByIDs(ids); err != nil {
		result.Fail(c, result.Warn(err.Error()))
		return
	}
	result.OK(c, result.Success())
}

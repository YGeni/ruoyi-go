package system

import (
	"errors"
	"net/http"
	"strings"

	"github.com/YGeni/ruoyi-go/server-go/internal/model/entity"
	"github.com/YGeni/ruoyi-go/server-go/internal/model/query"
	"github.com/YGeni/ruoyi-go/server-go/internal/service"
	"github.com/YGeni/ruoyi-go/server-go/pkg/page"
	"github.com/YGeni/ruoyi-go/server-go/pkg/result"

	"github.com/gin-gonic/gin"
)

// GenHandler 代码生成，对应 ruoyi-generator 的 GenController.java。
type GenHandler struct{}

// errGenTableNotFound 业务表不存在。
var errGenTableNotFound = errors.New("业务表不存在")

// GenList GET /tool/gen/list
//
// 等价 GenController.java:58-65：startPage() + getDataTable。
// 前端 src/views/tool/gen/index.vue:203-204 读 response.rows / response.total。
func (GenHandler) GenList(c *gin.Context) {
	if _, ok := loginUser(c); !ok {
		return
	}
	var q query.GenTableList
	if err := c.ShouldBindQuery(&q); err != nil {
		fail(c, err)
		return
	}
	q.FillDateRange(c)
	rows, total, err := service.SelectGenTableList(&q, page.FromRequest(c))
	if err != nil {
		fail(c, err)
		return
	}
	if rows == nil {
		rows = []entity.GenTable{}
	}
	c.JSON(http.StatusOK, result.DataTable(rows, total))
}

// GenDbList GET /tool/gen/db/list
//
// 等价 GenController.java:87-94：列出 information_schema 里**尚未导入**的表。
func (GenHandler) GenDbList(c *gin.Context) {
	if _, ok := loginUser(c); !ok {
		return
	}
	var q query.GenTableList
	if err := c.ShouldBindQuery(&q); err != nil {
		fail(c, err)
		return
	}
	rows, total, err := service.SelectDbTableList(&q, page.FromRequest(c))
	if err != nil {
		fail(c, err)
		return
	}
	if rows == nil {
		rows = []service.DbTable{}
	}
	c.JSON(http.StatusOK, result.DataTable(rows, total))
}

// GenGet GET /tool/gen/:tableId
//
// 等价 GenController.java:70-82：返回 {info, rows} 两个键
// （前端 editTable.vue 读的就是这两个）。
func (GenHandler) GenGet(c *gin.Context) {
	if _, ok := loginUser(c); !ok {
		return
	}
	id, ok := pathID(c, "tableId")
	if !ok {
		return
	}
	info, err := service.GenTableDetail(id)
	if err != nil {
		fail(c, err)
		return
	}
	result.OK(c, result.SuccessData(info))
}

// GenEdit PUT /tool/gen
func (GenHandler) GenEdit(c *gin.Context) {
	if _, ok := loginUser(c); !ok {
		return
	}
	var t entity.GenTable
	if err := c.ShouldBindJSON(&t); err != nil {
		fail(c, err)
		return
	}
	if err := service.UpdateGenTable(&t); err != nil {
		fail(c, err)
		return
	}
	result.OK(c, result.Success())
}

// GenImport POST /tool/gen/importTable
//
// 等价 GenController.java:113-123：参数是 tables（逗号分隔的表名）与 tplWebType。
func (GenHandler) GenImport(c *gin.Context) {
	lu, ok := loginUser(c)
	if !ok {
		return
	}
	tables := strings.Split(c.PostForm("tables"), ",")
	for i := range tables {
		tables[i] = strings.TrimSpace(tables[i])
	}
	op := ""
	if lu != nil {
		op = lu.Username()
	}
	if err := service.ImportTable(tables, op); err != nil {
		fail(c, err)
		return
	}
	result.OK(c, result.Success())
}

// GenCreateTable POST /tool/gen/createTable
//
// 等价 GenController.java:128-159：直接执行前端提交的建表 SQL，
// 失败统一回 error("创建表结构异常")。
func (GenHandler) GenCreateTable(c *gin.Context) {
	if _, ok := loginUser(c); !ok {
		return
	}
	if err := service.CreateTable(c.PostForm("sql")); err != nil {
		fail(c, err)
		return
	}
	result.OK(c, result.Success())
}

// GenSynchDb GET /tool/gen/synchDb/:tableName
func (GenHandler) GenSynchDb(c *gin.Context) {
	lu, ok := loginUser(c)
	if !ok {
		return
	}
	op := ""
	if lu != nil {
		op = lu.Username()
	}
	if err := service.SynchDb(c.Param("tableName"), op); err != nil {
		fail(c, err)
		return
	}
	result.OK(c, result.Success())
}

// GenRemove DELETE /tool/gen/:tableIds
func (GenHandler) GenRemove(c *gin.Context) {
	if _, ok := loginUser(c); !ok {
		return
	}
	if err := service.DeleteGenTableByIDs(parseIDs(c.Param("tableIds"))); err != nil {
		fail(c, err)
		return
	}
	result.OK(c, result.Success())
}

// GenPreview GET /tool/gen/preview/:tableId
//
// 等价 GenController.java:190-198：success({输出路径: 代码内容})。
// 前端 src/views/tool/gen/index.vue:263 直接把 response.data 挂到 preview 上按路径展示。
func (GenHandler) GenPreview(c *gin.Context) {
	if _, ok := loginUser(c); !ok {
		return
	}
	id, ok := pathID(c, "tableId")
	if !ok {
		return
	}
	t, err := service.SelectGenTableByID(id)
	if err != nil {
		fail(c, err)
		return
	}
	if t == nil {
		fail(c, errGenTableNotFound)
		return
	}
	files, err := service.RenderAll(t)
	if err != nil {
		fail(c, err)
		return
	}
	result.OK(c, result.SuccessData(files))
}

// GenCode GET /tool/gen/genCode/:tableName
//
// 等价 GenController.java:215-230：genType='0' 时打成 zip 下载，
// genType='1' 时写到 genPath 指定的自定义路径。
//
// 响应头必须对：前端用 blob 接收，Content-Type 必须是 zip
// （见 monitor_handler.go 的 writeXlsx 注释）。
func (GenHandler) GenCode(c *gin.Context) {
	if _, ok := loginUser(c); !ok {
		return
	}
	name := c.Param("tableName")
	table, err := service.GenTableByName(name)
	if err != nil {
		fail(c, err)
		return
	}
	if table == nil {
		fail(c, errGenTableNotFound)
		return
	}
	if table.GenType == "1" {
		// 自定义路径：落盘，不下载
		if err := service.GenToPath(table); err != nil {
			fail(c, err)
			return
		}
		result.OK(c, result.Success())
		return
	}
	data, err := service.GenZip(table)
	if err != nil {
		fail(c, err)
		return
	}
	c.Header("Content-Type", "application/zip")
	c.Header("Content-Disposition", "attachment; filename=ruoyi.zip")
	c.Data(http.StatusOK, "application/zip", data)
}

// GenColumnList GET /tool/gen/column/:tableId
//
// 等价 GenController.java:99-107：返回 {rows, total}，total 恒等于列表长度
// （与 /monitor/online/list 同理，无 startPage()）。
//
// 前端没用到这个端点（src/api/tool/gen.js 里没有），Java 侧有注解就必须登记，
// 因此一并实现。
func (GenHandler) GenColumnList(c *gin.Context) {
	if _, ok := loginUser(c); !ok {
		return
	}
	id, ok := pathID(c, "tableId")
	if !ok {
		return
	}
	rows, err := service.GenColumns(id)
	if err != nil {
		fail(c, err)
		return
	}
	c.JSON(http.StatusOK, result.DataTable(rows, int64(len(rows))))
}

// GenDownload GET /tool/gen/download/:tableName
//
// 等价 GenController.java:201-208：**始终**下发 zip（不像 genCode 会看 genType）。
// 前端也没用到（走的是 batchGenCode），同样是因为 Java 有注解才登记。
func (GenHandler) GenDownload(c *gin.Context) {
	if _, ok := loginUser(c); !ok {
		return
	}
	t, err := service.GenTableByName(c.Param("tableName"))
	if err != nil {
		fail(c, err)
		return
	}
	if t == nil {
		fail(c, errGenTableNotFound)
		return
	}
	data, err := service.GenZip(t)
	if err != nil {
		fail(c, err)
		return
	}
	writeZip(c, "ruoyi", data)
}

// GenBatchGenCode GET /tool/gen/batchGenCode
//
// 等价 GenController.java:241-247：参数 tables 是逗号分隔的表名。
// 前端 src/views/tool/gen/index.vue:228 正是用
// `proxy.$download.zip("/tool/gen/batchGenCode?tables=" + tbNames, zipName)`
// 调它的——**页面上的"生成代码"按钮走的是这里**，不是 genCode。
func (GenHandler) GenBatchGenCode(c *gin.Context) {
	if _, ok := loginUser(c); !ok {
		return
	}
	names := strings.Split(c.Query("tables"), ",")
	data, err := service.GenZipBatch(names)
	if err != nil {
		fail(c, err)
		return
	}
	writeZip(c, "ruoyi", data)
}

// writeZip 下发 zip。与 writeXlsx 同理：Content-Type 必须是 zip，
// 否则前端 blobValidate 会把它当成错误 JSON。
func writeZip(c *gin.Context, name string, data []byte) {
	c.Header("Content-Type", "application/zip")
	c.Header("Content-Disposition", "attachment; filename="+name+".zip")
	c.Data(http.StatusOK, "application/zip", data)
}

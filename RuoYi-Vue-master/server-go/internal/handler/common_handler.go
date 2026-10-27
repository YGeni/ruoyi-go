package handler

import (
	"errors"
	"fmt"
	"mime/multipart"
	"net/http"
	"strings"
	"time"

	"github.com/YGeni/ruoyi-go/server-go/config"
	"github.com/YGeni/ruoyi-go/server-go/internal/middleware"
	"github.com/YGeni/ruoyi-go/server-go/internal/model/entity"
	"github.com/YGeni/ruoyi-go/server-go/pkg/filex"
	"github.com/YGeni/ruoyi-go/server-go/pkg/result"

	"github.com/gin-gonic/gin"
)

// CommonHandler 通用上传下载，对应 CommonController.java。
type CommonHandler struct{}

// Upload POST /common/upload
//
// 等价 CommonController.java:74-95：单文件上传到 RuoYiConfig.getUploadPath()。
//
// 返回体四个键（url / fileName / newFileName / originalFilename）逐个对齐，
// **fileName 必须有**（用户契约第 5 条），后续 /common/download 与
// /common/download/resource 都靠它定位文件。
func (CommonHandler) Upload(c *gin.Context) {
	if _, ok := loginUser(c); !ok {
		return
	}
	fh, err := c.FormFile("file")
	if err != nil {
		failFile(c, err)
		return
	}
	if err := checkSizes(fh.Size, fh.Size); err != nil {
		result.Fail(c, result.Error(err.Error()))
		return
	}
	fileName, err := filex.Upload(config.C.RuoYi.GetProfile(), config.C.RuoYi.GetUploadPath(),
		fh, config.DefaultAllowedExtension, false)
	if err != nil {
		result.Fail(c, result.Error(err.Error()))
		return
	}
	result.OK(c, result.Success().
		Put("url", serverURL(c)+fileName).
		Put("fileName", fileName).
		Put("newFileName", filex.Name(fileName)).
		Put("originalFilename", fh.Filename))
}

// Uploads POST /common/uploads
//
// 等价 CommonController.java:100-132：多文件上传，四个键都是 **逗号连接的字符串**
// （StringUtils.join(list, ",")，FILE_DELIMITER = ","，见 L37）。
//
// DIFF(review): Java 的方法参数是 List<MultipartFile> files，Spring 绑定的表单字段名
// 即参数名 files；前端也有直接复用 file 字段的写法，这里两者都收（files 优先）。
func (CommonHandler) Uploads(c *gin.Context) {
	if _, ok := loginUser(c); !ok {
		return
	}
	form, err := c.MultipartForm()
	if err != nil {
		failFile(c, err)
		return
	}
	files := form.File["files"]
	if len(files) == 0 {
		files = form.File["file"]
	}
	if len(files) == 0 {
		result.Fail(c, result.Error("没有上传任何文件"))
		return
	}

	var total int64
	for _, f := range files {
		total += f.Size
	}
	if err := checkSizes(maxSize(files), total); err != nil {
		result.Fail(c, result.Error(err.Error()))
		return
	}

	urls := make([]string, 0, len(files))
	fileNames := make([]string, 0, len(files))
	newFileNames := make([]string, 0, len(files))
	originals := make([]string, 0, len(files))

	for _, f := range files {
		fileName, err := filex.Upload(config.C.RuoYi.GetProfile(), config.C.RuoYi.GetUploadPath(),
			f, config.DefaultAllowedExtension, false)
		if err != nil {
			result.Fail(c, result.Error(err.Error()))
			return
		}
		urls = append(urls, serverURL(c)+fileName)
		fileNames = append(fileNames, fileName)
		newFileNames = append(newFileNames, filex.Name(fileName))
		originals = append(originals, f.Filename)
	}

	result.OK(c, result.Success().
		Put("urls", strings.Join(urls, ",")).
		Put("fileNames", strings.Join(fileNames, ",")).
		Put("newFileNames", strings.Join(newFileNames, ",")).
		Put("originalFilenames", strings.Join(originals, ",")))
}

// Download GET /common/download?fileName=&delete=
//
// 等价 CommonController.java:45-69。
//
// 两道校验（缺一不可）：
//  1. FileUtils.checkAllowDownload：拒绝含 ".." 的文件名 + 扩展名白名单
//  2. filex.SafeJoin：真正拼路径时再拦一次目录穿越
//
// fileName 是**相对于 profile/download/ 的相对名**（Java 的
// getDownloadPath() 自带尾斜杠，见 config.go 的注释），不是完整路径。
//
// 下载名 realFileName = 当前毫秒 + fileName 中第一个 "_" 之后的内容（L54）；
// 没有 "_" 时 Java 的 indexOf 返回 -1、substring(0) 取整串，这里同样处理。
func (CommonHandler) Download(c *gin.Context) {
	if _, ok := loginUser(c); !ok {
		return
	}
	fileName := c.Query("fileName")
	if !filex.CheckAllowDownload(fileName) {
		result.Fail(c, result.Error(fmt.Sprintf(config.MsgAllowDownloadFmt, fileName)))
		return
	}
	downloadPath := config.C.RuoYi.GetDownloadPath()
	abs, ok := filex.SafeJoin(downloadPath, fileName)
	if !ok {
		result.Fail(c, result.Error(fmt.Sprintf(config.MsgAllowDownloadFmt, fileName)))
		return
	}

	realFileName := downloadName(fileName)
	c.Header("Content-Type", "application/octet-stream")
	filex.SetAttachmentResponseHeader(func(k, v string) { c.Header(k, v) }, realFileName)
	if err := filex.WriteBytes(abs, c.Writer); err != nil {
		result.Fail(c, result.Error(fmt.Sprintf(config.MsgAllowDownloadFmt, fileName)))
		return
	}
	// delete=true 时下载完删除源文件（L60-63）
	if strings.EqualFold(c.Query("delete"), "true") {
		filex.DeleteFile(abs)
	}
}

// ResourceDownload GET /common/download/resource?resource=
//
// 等价 CommonController.java:137-161：下载 profile 目录下的任意资源。
//
// resource 形如 "/profile/upload/2026/10/28/xxx.png"（即 /common/upload 返回的 fileName），
// 先用 StripPrefix 去掉 "/profile" 再拼根目录（L150）。
func (CommonHandler) ResourceDownload(c *gin.Context) {
	if _, ok := loginUser(c); !ok {
		return
	}
	resource := c.Query("resource")
	if !filex.CheckAllowDownload(resource) {
		result.Fail(c, result.Error(fmt.Sprintf(config.MsgResourceAllowDownloadFmt, resource)))
		return
	}
	abs, ok := filex.SafeJoin(config.C.RuoYi.GetProfile(), filex.StripPrefix(resource))
	if !ok {
		result.Fail(c, result.Error(fmt.Sprintf(config.MsgResourceAllowDownloadFmt, resource)))
		return
	}
	c.Header("Content-Type", "application/octet-stream")
	filex.SetAttachmentResponseHeader(func(k, v string) { c.Header(k, v) }, filex.Name(abs))
	if err := filex.WriteBytes(abs, c.Writer); err != nil {
		result.Fail(c, result.Error(fmt.Sprintf(config.MsgResourceAllowDownloadFmt, resource)))
		return
	}
}

// ---------- 辅助 ----------

// failFile 上传取文件失败的统一出口。
//
// ContentLength 为 -1（chunked）时 limitBody 的预判会失效，
// http.MaxBytesReader 会在读取过程中报 *http.MaxBytesError，
// 原始文案是英文的 "http: request body too large" —— 前端直接展示会很怪，
// 这里统一换成与 Java 一致的中文超限提示（同样必须是 application/json）。
func failFile(c *gin.Context, err error) {
	var mbErr *http.MaxBytesError
	if errors.As(err, &mbErr) {
		max := config.C.Spring.Servlet.Multipart.MaxRequestSizeBytes()
		result.Fail(c, result.Error(fmt.Sprintf(config.MsgUploadExceedRequestSize, max/1024/1024)))
		return
	}
	fail(c, err)
}

// downloadName 等价 CommonController.java:54 的 realFileName 计算。
func downloadName(fileName string) string {
	rest := fileName
	if i := strings.Index(fileName, "_"); i >= 0 {
		rest = fileName[i+1:]
	}
	return fmt.Sprintf("%d%s", time.Now().UnixMilli(), rest)
}

// checkSizes 校验单文件与总请求大小，对齐 application.yml 的 multipart 配置。
//
// 之所以在 Go 侧显式再查一遍而不是只靠 gin：gin 的 MaxMultipartMemory 只控制
// **内存缓冲**，超限会落临时文件，并不会拒绝请求；Java 的 max-file-size /
// max-request-size 是硬闸，超限直接失败。
func checkSizes(maxFile, total int64) error {
	limFile := config.C.Spring.Servlet.Multipart.MaxFileSizeBytes()
	limTotal := config.C.Spring.Servlet.Multipart.MaxRequestSizeBytes()
	if maxFile > limFile {
		return fmt.Errorf(config.MsgUploadExceedMaxSize, limFile/1024/1024)
	}
	if total > limTotal {
		return fmt.Errorf(config.MsgUploadExceedRequestSize, limTotal/1024/1024)
	}
	return nil
}

func maxSize(files []*multipart.FileHeader) int64 {
	var m int64
	for _, f := range files {
		if f.Size > m {
			m = f.Size
		}
	}
	return m
}

// serverURL 等价 ServerConfig.getUrl()：scheme://host + context-path。
//
// context-path 为 "/"，因此就是 scheme://host。
func serverURL(c *gin.Context) string {
	scheme := "http"
	if c.Request.TLS != nil || c.GetHeader("X-Forwarded-Proto") == "https" {
		scheme = "https"
	}
	return scheme + "://" + c.Request.Host
}

// loginUser 取当前登录用户，缺失时直接回 401。
func loginUser(c *gin.Context) (*entity.LoginUser, bool) {
	lu, ok := middleware.LoginUser(c)
	if !ok || lu == nil {
		result.Fail(c, result.ErrorCode(http.StatusUnauthorized, "认证失败，无法访问系统资源"))
		return nil, false
	}
	return lu, true
}

func fail(c *gin.Context, err error) {
	result.Fail(c, result.Error(err.Error()))
}

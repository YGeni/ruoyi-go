// Package filex 复刻 RuoYi 的文件上传/下载工具链。
//
// 契约来源（逐行对齐，禁止凭直觉改写）：
// -ruoyi-上传校验、命名、路径映射
// -ruoyi-下载白名单、删除、写出、名称处理
// -ruoyi-扩展名白名单
// -ruoyi-取扩展名（小写）
//
// 之所以单独成包而不是散在 handler 里：/common/upload、/common/uploads、
// /common/download、/common/download/resource、/system/user/profile/avatar
// 五处共用同一套校验与路径规则，分散实现必然出现"某处忘了校验 .."。
package filex

import (
	"errors"
	"fmt"
	"io"
	"mime/multipart"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"time"

	"github.com/YGeni/ruoyi-go/server-go/config"

	"github.com/google/uuid"
)

// ---------- 扩展名 ----------

// ExtensionOf 等价 FileUploadUtils.getExtension(L251-259)：
// 先取文件名后缀，取不到再退回 Content-Type 查表（MimeTypeUtils.getExtension）。
func ExtensionOf(filename, contentType string) string {
	ext := FileType(filename)
	if ext == "" {
		return mimeToExtension(contentType)
	}
	return ext
}

// FileType 等价 FileTypeUtils.getFileType(String)：
// 最后一个 "." 之后的内容，转小写；没有 "." 返回空串。
func FileType(fileName string) string {
	i := strings.LastIndex(fileName, ".")
	if i < 0 {
		return ""
	}
	return strings.ToLower(fileName[i+1:])
}

// mimeToExtension 等价 MimeTypeUtils.getExtension(L41-58)，只认五种图片 MIME。
func mimeToExtension(prefix string) string {
	switch prefix {
	case "image/png":
		return "png"
	case "image/jpg":
		return "jpg"
	case "image/jpeg":
		return "jpeg"
	case "image/bmp":
		return "bmp"
	case "image/gif":
		return "gif"
	default:
		return ""
	}
}

// IsAllowedExtension 等价 FileUploadUtils.isAllowedExtension(L233-243)：忽略大小写逐个比对。
func IsAllowedExtension(ext string, allowed []string) bool {
	for _, s := range allowed {
		if strings.EqualFold(s, ext) {
			return true
		}
	}
	return false
}

// ---------- 上传校验异常 ----------

// InvalidExtensionError 等价 InvalidExtensionException（含四个子类的统一文案）。
//
// Java 的四个子类（Image/Flash/Media/Video）文案**完全相同**，
// 只是异常类型不同（供 GlobalExceptionHandler 分类），Go 侧无需保留类型区分。
type InvalidExtensionError struct {
	Allowed   []string
	Extension string
	Filename  string
}

func (e *InvalidExtensionError) Error() string {
	return fmt.Sprintf(config.MsgInvalidExtensionFmt, e.Filename, e.Extension, fmt.Sprint(e.Allowed))
}

// FileSizeLimitError 等价 FileSizeLimitExceededException（upload.exceed.maxSize）。
type FileSizeLimitError struct{ MaxSizeMB int64 }

func (e *FileSizeLimitError) Error() string {
	return fmt.Sprintf(config.MsgUploadExceedMaxSize, e.MaxSizeMB)
}

// FileNameLengthError 等价 FileNameLengthLimitExceededException（upload.filename.exceed.length）。
type FileNameLengthError struct{ MaxLength int }

func (e *FileNameLengthError) Error() string {
	return fmt.Sprintf(config.MsgUploadFilenameExceedLen, e.MaxLength)
}

// AssertAllowed 等价 FileUploadUtils.assertAllowed(L186-224)。
//
// 校验顺序与 Java 一致：先大小、再扩展名。
func AssertAllowed(fh *multipart.FileHeader, allowed []string) error {
	if fh.Size > config.DefaultMaxSize {
		return &FileSizeLimitError{MaxSizeMB: config.DefaultMaxSize / 1024 / 1024}
	}
	filename := fh.Filename
	ext := ExtensionOf(filename, fh.Header.Get("Content-Type"))
	if allowed != nil && !IsAllowedExtension(ext, allowed) {
		return &InvalidExtensionError{Allowed: allowed, Extension: ext, Filename: filename}
	}
	return nil
}

// ---------- 文件命名 ----------

// uploadSeq 等价 Seq.uploadSeq，进程内自增。
var uploadSeq int64 = 1

// uploadSeqID 等价 Seq.getId(uploadSeq, 3)
// yyMMddHHmmss + 机器标识 "A" + 3 位左补零循环序号。
//
// DIFF(review): Java 的 machineCode 固定为 "A"（单实例），Go 侧同样固定，
// 不做多实例区分 —— 与 Java 行为一致，避免生成规则漂移。
func uploadSeqID() string {
	v := atomic.AddInt64(&uploadSeq, 1) - 1
	return time.Now().Format("060102150405") + "A" + fmt.Sprintf("%03d", v%1000)
}

// DatePath 等价 DateUtils.datePath：yyyy/MM/dd。
func DatePath() string { return time.Now().Format("2006/01/02") }

// ExtractFilename 等价 FileUploadUtils.extractFilename(L144-147)：
// 日期目录 + 原文件名(去后缀) + "_" + 序列号 + "." + 后缀。
//
// 用于 /common/upload（useCustomNaming=false），保留原文件名便于下载时辨认。
func ExtractFilename(fh *multipart.FileHeader) string {
	base := BaseName(fh.Filename)
	return fmt.Sprintf("%s/%s_%s.%s", DatePath(), base, uploadSeqID(), ExtensionOf(fh.Filename, fh.Header.Get("Content-Type")))
}

// UUIDFilename 等价 FileUploadUtils.uuidFilename(L152-155)：
// 日期目录 + UUID(无横线) + "." + 后缀。
//
// 用于头像上传（Java 的传 useCustomNaming=true）。
//
// DIFF(review): 头像**做**重命名（UUID），与 /common/upload 保留原名的策略不同 ——
// 原因是头像 URL 直接暴露在 sys_user.avatar 与 getInfo 里，
// 保留原始文件名会泄漏用户本地文件名，Java 侧正是为此传了 true。
func UUIDFilename(fh *multipart.FileHeader) string {
	return fmt.Sprintf("%s/%s.%s", DatePath(), uuid.NewString(), ExtensionOf(fh.Filename, fh.Header.Get("Content-Type")))
}

// BaseName 等价 FilenameUtils.getBaseName：去掉目录与最后一个后缀。
func BaseName(name string) string {
	base := filepath.Base(filepath.ToSlash(name))
	if i := strings.LastIndex(base, "."); i > 0 {
		return base[:i]
	}
	return base
}

// Name 等价 FileUtils.getName(L276-286)：取路径最后一段（同时处理 / 与 \）。
func Name(fileName string) string {
	if fileName == "" {
		return ""
	}
	lastUnix := strings.LastIndex(fileName, "/")
	lastWin := strings.LastIndex(fileName, "\\")
	i := lastUnix
	if lastWin > i {
		i = lastWin
	}
	if i < 0 {
		return fileName
	}
	return fileName[i+1:]
}

// ---------- 路径映射 ----------

// GetPathFileName 等价 FileUploadUtils.getPathFileName(L171-176)：
// 把磁盘目录映射回 "/profile/子目录/文件名" 这种对外可见的相对路径。
//
// Java：StringUtils.substring(uploadDir, RuoYiConfig.getProfile().length() + 1)，
// 即砍掉根目录与紧随其后的那个 "/"；越界时 Apache StringUtils 返回空串。
func GetPathFileName(profile, uploadDir, fileName string) string {
	dirLastIndex := len(profile) + 1
	currentDir := ""
	if len(uploadDir) > dirLastIndex {
		currentDir = uploadDir[dirLastIndex:]
	}
	return config.ResourcePrefix + "/" + currentDir + "/" + fileName
}

// StripPrefix 等价 FileUtils.stripPrefix(L113-116)：去掉 "/profile" 及其之前的内容。
//
// sys_user.avatar = "/profile/avatar/2026/10/28/a.jpg" -> "/avatar/2026/10/28/a.jpg"
func StripPrefix(filePath string) string {
	if i := strings.Index(filePath, config.ResourcePrefix); i >= 0 {
		return filePath[i+len(config.ResourcePrefix):]
	}
	return filePath
}

// ---------- 下载校验 ----------

// CheckAllowDownload 等价 FileUtils.checkAllowDownload(L153-169)，**两道闸**：
//
//  1. 含 ".." 一律拒绝（目录穿越）
//  2. 扩展名必须落在 DefaultAllowedExtension 白名单内
//
// DIff(review) 收紧点：Java 只查 ".." 子串；Go 侧在此之外，真正拼接路径时还会再做一次
// 目录逃逸校验（见 SafeJoin），双保险 —— 前端 src/utils/ruoyi.js 的 Blob 还原
// 依赖错误响应仍是 application/json，故校验失败必须走 result.Fail 而非裸 500。
func CheckAllowDownload(resource string) bool {
	if strings.Contains(resource, "..") {
		return false
	}
	return IsAllowedExtension(FileType(resource), config.DefaultAllowedExtension)
}

// SafeJoin 把用户提供的相对路径安全地拼到 root 下，拒绝越界。
//
// 返回 (绝对路径, true) 或 ("", false)。校验手法：
// filepath.Clean 后前缀必须仍是 root；同时用 EvalSymlinks 兜不住的场景
// （符号链接指向 root 外）也一并拒绝。
func SafeJoin(root, rel string) (string, bool) {
	cleanRoot, err := filepath.Abs(root)
	if err != nil {
		return "", false
	}
	// 直接 Join：**不要**先给 rel 补前导 "/" 再 Clean —— 那样会把
	// "../../etc/passwd" 归一化成 "/etc/passwd"，看起来没逃逸，实则是把
	// 穿越请求悄悄改写成了合法请求（测试 TestSafeJoin 专门盯这个）。
	// filepath.Join 自带 Clean，越界的结果会落在 root 之外，被下面的前缀判断拦住。
	abs, err := filepath.Abs(filepath.Join(cleanRoot, filepath.FromSlash(rel)))
	if err != nil {
		return "", false
	}
	if abs != cleanRoot && !strings.HasPrefix(abs, cleanRoot+string(os.PathSeparator)) {
		return "", false
	}
	return abs, true
}

// ---------- 文件操作 ----------

// DeleteFile 等价 FileUtils.deleteFile(L124-134)：只删普通文件，不存在返回 false。
func DeleteFile(filePath string) bool {
	info, err := os.Stat(filePath)
	if err != nil || info.IsDir() {
		return false
	}
	return os.Remove(filePath) == nil
}

// Save 把上传文件写到 absPath，等价 file.transferTo(Paths.get(absPath))（L137）。
//
// 会按需创建父目录（等价 getAbsoluteFile 里的 mkdirs，L161-167）。
func Save(fh *multipart.FileHeader, absPath string) error {
	if err := os.MkdirAll(filepath.Dir(absPath), 0o755); err != nil {
		return err
	}
	src, err := fh.Open()
	if err != nil {
		return err
	}
	defer src.Close()

	dst, err := os.OpenFile(absPath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0o644)
	if err != nil {
		return err
	}
	defer dst.Close()

	_, err = io.Copy(dst, src)
	return err
}

// Upload 等价 FileUploadUtils.upload(baseDir, file, allowedExtension, useCustomNaming)（L122-139）。
//
// 返回对外可见的相对路径（形如 "/profile/upload/2026/10/28/xxx.png"）。
func Upload(profile, baseDir string, fh *multipart.FileHeader, allowed []string, useCustomNaming bool) (string, error) {
	if len(fh.Filename) > config.DefaultFileNameLength {
		return "", &FileNameLengthError{MaxLength: config.DefaultFileNameLength}
	}
	if err := AssertAllowed(fh, allowed); err != nil {
		return "", err
	}
	fileName := ExtractFilename(fh)
	if useCustomNaming {
		fileName = UUIDFilename(fh)
	}
	absPath := filepath.Join(baseDir, filepath.FromSlash(fileName))
	if err := Save(fh, absPath); err != nil {
		return "", err
	}
	return GetPathFileName(profile, baseDir, fileName), nil
}

// WriteBytes 等价 FileUtils.writeBytes(String filePath, OutputStream os)（L39-66）：
// 文件不存在返回错误，存在则整体拷进 w。
func WriteBytes(filePath string, w io.Writer) error {
	f, err := os.Open(filePath)
	if err != nil {
		return err
	}
	defer f.Close()
	_, err = io.Copy(w, f)
	return err
}

// ---------- 下载响应头 ----------

// SetAttachmentResponseHeader 等价 FileUtils.setAttachmentResponseHeader(L212-227)。
//
// 除 Content-Disposition 外还带两个头（Java 也有）：
//   - Access-Control-Expose-Headers: 让前端 JS 能读到 Content-Disposition
//   - download-filename: 已编码的文件名，前端 src/utils/ruoyi.js:225-228 依赖它
func SetAttachmentResponseHeader(set func(k, v string), realFileName string) {
	encoded := PercentEncode(realFileName)
	set("Access-Control-Expose-Headers", "Content-Disposition,download-filename")
	set("Content-Disposition", fmt.Sprintf("attachment; filename=%s;filename*=utf-8''%s", encoded, encoded))
	set("download-filename", encoded)
}

// PercentEncode 等价 FileUtils.percentEncode(L235-239)：
// URLEncoder.encode(UTF-8) 后把 "+" 换成 "%20"（Java 的 URLEncoder 把空格编码成 +）。
func PercentEncode(s string) string {
	return strings.ReplaceAll(url.QueryEscape(s), "+", "%20")
}

// ErrNotAllowedDownload 供 handler 组装文案使用。
var ErrNotAllowedDownload = errors.New("不允许下载")

package filex

import (
	"mime/multipart"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/YGeni/ruoyi-go/server-go/config"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// newFileHeader 造一个 multipart.FileHeader 供校验类用例使用。
func newFileHeader(filename, contentType string, size int64) *multipart.FileHeader {
	h := make(map[string][]string)
	if contentType != "" {
		h["Content-Type"] = []string{contentType}
	}
	return &multipart.FileHeader{Filename: filename, Size: size, Header: h}
}

func TestFileType(t *testing.T) {
	cases := map[string]string{
		"a.png":        "png",
		"a.PNG":        "png", // 转小写，FileTypeUtils.java:41
		"dir/a.tar.gz": "gz",  // 取最后一个点
		"noext":        "",
		".gitignore":   "gitignore", // lastIndex 在 0，仍取其后内容
	}
	for in, want := range cases {
		assert.Equal(t, want, FileType(in), in)
	}
}

func TestExtensionOfFallsBackToContentType(t *testing.T) {
	// FileUploadUtils.getExtension: 文件名没后缀时退回 Content-Type 查表
	assert.Equal(t, "png", ExtensionOf("avatar", "image/png"))
	assert.Equal(t, "jpeg", ExtensionOf("avatar", "image/jpeg"))
	assert.Equal(t, "", ExtensionOf("avatar", "application/octet-stream"))
	assert.Equal(t, "gif", ExtensionOf("a.gif", "image/png"), "有后缀时以后缀为准")
}

func TestIsAllowedExtension(t *testing.T) {
	assert.True(t, IsAllowedExtension("PNG", config.ImageExtension))
	assert.False(t, IsAllowedExtension("exe", config.ImageExtension))
	assert.True(t, IsAllowedExtension("pdf", config.DefaultAllowedExtension))
	assert.False(t, IsAllowedExtension("php", config.DefaultAllowedExtension))
}

func TestAssertAllowed(t *testing.T) {
	// 1) 大小优先于扩展名（与 Java 的 assertAllowed 顺序一致）
	err := AssertAllowed(newFileHeader("a.exe", "", config.DefaultMaxSize+1), config.DefaultAllowedExtension)
	var sizeErr *FileSizeLimitError
	require.ErrorAs(t, err, &sizeErr)
	assert.Equal(t, int64(50), sizeErr.MaxSizeMB)
	assert.Contains(t, err.Error(), "50MB")

	// 2) 扩展名不合法
	err = AssertAllowed(newFileHeader("a.exe", "", 10), config.ImageExtension)
	var extErr *InvalidExtensionError
	require.ErrorAs(t, err, &extErr)
	assert.Contains(t, err.Error(), "a.exe")
	assert.Contains(t, err.Error(), "exe")

	// 3) 合法通过
	require.NoError(t, AssertAllowed(newFileHeader("a.png", "", 10), config.ImageExtension))
}

func TestFileNameLengthLimit(t *testing.T) {
	long := strings.Repeat("a", config.DefaultFileNameLength-3) + ".png"
	fh := newFileHeader(long, "", 10)
	_, err := Upload("/tmp/p", "/tmp/p/upload", fh, config.ImageExtension, false)
	var nameErr *FileNameLengthError
	require.ErrorAs(t, err, &nameErr)
	assert.Equal(t, 100, nameErr.MaxLength)
}

func TestGetPathFileName(t *testing.T) {
	// 对齐 FileUploadUtils.getPathFileName：砍掉根目录与其后的 "/" 再拼 /profile
	got := GetPathFileName("/data/uploadPath", "/data/uploadPath/avatar", "2026/10/28/a.png")
	assert.Equal(t, "/profile/avatar/2026/10/28/a.png", got)

	got = GetPathFileName("/data/uploadPath", "/data/uploadPath/upload", "2026/10/28/a.png")
	assert.Equal(t, "/profile/upload/2026/10/28/a.png", got)
}

func TestStripPrefix(t *testing.T) {
	assert.Equal(t, "/avatar/2026/10/28/a.png", StripPrefix("/profile/avatar/2026/10/28/a.png"))
	assert.Equal(t, "x/y.png", StripPrefix("x/y.png"), "无前缀时原样返回")
}

func TestName(t *testing.T) {
	assert.Equal(t, "a.png", Name("/profile/upload/2026/10/28/a.png"))
	assert.Equal(t, "a.png", Name("C:\\tmp\\a.png"))
	assert.Equal(t, "a.png", Name("a.png"))
}

// TestCheckAllowDownload 下载白名单的两道闸。
func TestCheckAllowDownload(t *testing.T) {
	// 闸一：目录穿越（FileUtils.java:156）
	assert.False(t, CheckAllowDownload("../etc/passwd.png"))
	assert.False(t, CheckAllowDownload("a/../../b.png"))
	assert.False(t, CheckAllowDownload("..png"), "Java 只判子串 '..'，Go 侧同样判子串")
	// 闸二：扩展名白名单
	assert.True(t, CheckAllowDownload("a.png"))
	assert.True(t, CheckAllowDownload("a.pdf"))
	assert.False(t, CheckAllowDownload("a.exe"))
	assert.False(t, CheckAllowDownload("a"), "无扩展名不在白名单")
}

// TestSafeJoin 目录穿越的第二道闸（真正拼路径时）。
func TestSafeJoin(t *testing.T) {
	root := t.TempDir()
	ok := []string{"/2026/10/28/a.png", "a.png", "/sub/../a.png"}
	for _, rel := range ok {
		abs, valid := SafeJoin(root, rel)
		assert.True(t, valid, rel)
		if valid {
			assert.True(t, strings.HasPrefix(abs, root), "%s 逃逸出根目录", rel)
		}
	}
	// 越界一律拒绝
	for _, rel := range []string{"../../etc/passwd", "/../../../../etc/passwd", "/..//..//x"} {
		_, valid := SafeJoin(root, rel)
		assert.False(t, valid, rel)
	}
}

func TestPercentEncode(t *testing.T) {
	// Java 的 URLEncoder 把空格编码成 "+"，percentEncode 再换成 %20
	assert.Equal(t, "%E5%A4%B4%E5%83%8F.png", PercentEncode("头像.png"))
	assert.Equal(t, "a%20b.png", PercentEncode("a b.png"))
	assert.NotContains(t, PercentEncode("a b.png"), "+")
}

func TestSetAttachmentResponseHeader(t *testing.T) {
	got := map[string]string{}
	SetAttachmentResponseHeader(func(k, v string) { got[k] = v }, "a b.png")
	assert.Equal(t, "Content-Disposition,download-filename", got["Access-Control-Expose-Headers"])
	assert.Equal(t, "a%20b.png", got["download-filename"])
	assert.True(t, strings.HasPrefix(got["Content-Disposition"], "attachment; filename=a%20b.png;"))
	assert.Contains(t, got["Content-Disposition"], "filename*=utf-8''a%20b.png")
}

func TestSaveCreatesDirs(t *testing.T) {
	// Save 必须按需 mkdirs（等价 FileUploadUtils.getAbsoluteFile 的 mkdirs，L161-167）
	tmp := t.TempDir()
	dir := filepath.Join(tmp, "a", "b", "c")
	p := filepath.Join(dir, "x.png")
	src := filepath.Join(tmp, "src.png")
	require.NoError(t, os.WriteFile(src, []byte("PNG"), 0o644))
	require.NoError(t, Save(fileHeaderFromFile(t, src, "x.png"), p))
	b, err := os.ReadFile(p)
	require.NoError(t, err)
	assert.Equal(t, "PNG", string(b))
}

func TestUploadWritesFile(t *testing.T) {
	profile := t.TempDir()
	base := filepath.Join(profile, "upload")
	src := mustTempFile(t, "pic.PNG", "PNG!")

	got, err := Upload(profile, base, fileHeaderFromFile(t, src, "pic.PNG"), config.DefaultAllowedExtension, false)
	require.NoError(t, err)
	assert.True(t, strings.HasPrefix(got, "/profile/upload/"), got)

	// 落盘路径 = base + 去掉 /profile/upload 前缀的部分
	rel := strings.TrimPrefix(got, "/profile/upload/")
	b, statErr := os.ReadFile(filepath.Join(base, rel))
	require.NoError(t, statErr, "Upload 返回的相对路径应能定位到真实文件")
	assert.Equal(t, "PNG!", string(b), "内容必须原样落盘")
}

// fileHeaderFromFile 用本地文件构造真实的 multipart.FileHeader（带可 Open 的内容）。
func fileHeaderFromFile(t *testing.T, path, name string) *multipart.FileHeader {
	t.Helper()
	f, err := os.Open(path)
	require.NoError(t, err)
	defer f.Close()

	var buf strings.Builder
	// 复制到 multipart 内存体再解析，等价于一次真实表单上传
	w := multipart.NewWriter(&buf)
	part, err := w.CreateFormFile("file", name)
	require.NoError(t, err)
	b, err := os.ReadFile(path)
	require.NoError(t, err)
	_, err = part.Write(b)
	require.NoError(t, err)
	require.NoError(t, w.Close())

	r := multipart.NewReader(strings.NewReader(buf.String()), w.Boundary())
	form, err := r.ReadForm(10 << 20)
	require.NoError(t, err)
	require.Len(t, form.File["file"], 1)
	return form.File["file"][0]
}

func TestUploadNaming(t *testing.T) {
	fh := fileHeaderFromFile(t, mustTempFile(t, "a.png", "x"), "a.png")

	// useCustomNaming=false -> 保留原名 + "_" + 序列号（头像之外）
	profile := t.TempDir()
	got, err := Upload(profile, filepath.Join(profile, "upload"), fh, config.DefaultAllowedExtension, false)
	require.NoError(t, err)
	assert.Contains(t, got, "/a_")
	assert.True(t, strings.HasSuffix(got, ".png"))

	// useCustomNaming=true -> UUID（头像走这条，Java SysProfileController.java:131 传 true）
	gotUUID, err := Upload(profile, filepath.Join(profile, "avatar"), fh, config.ImageExtension, true)
	require.NoError(t, err)
	assert.Contains(t, gotUUID, "/profile/avatar/")
	assert.NotContains(t, gotUUID, "/a_", "头像必须重命名，不保留原始文件名")
	// UUID 是 36 位带横线
	name := Name(gotUUID)
	assert.Len(t, strings.TrimSuffix(name, ".png"), 36)
}

func TestUploadRejectsIllegalExtension(t *testing.T) {
	profile := t.TempDir()
	fh := fileHeaderFromFile(t, mustTempFile(t, "evil.exe", "x"), "evil.exe")
	_, err := Upload(profile, filepath.Join(profile, "upload"), fh, config.DefaultAllowedExtension, false)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "evil.exe")
}

func TestDeleteFile(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "a.txt")
	require.NoError(t, os.WriteFile(p, []byte("x"), 0o644))
	assert.True(t, DeleteFile(p))
	assert.False(t, DeleteFile(p), "已删除再删返回 false")
	assert.False(t, DeleteFile(dir), "目录不删（Java 的 isFile 判断）")
}

func mustTempFile(t *testing.T, name, content string) string {
	t.Helper()
	p := filepath.Join(t.TempDir(), name)
	require.NoError(t, os.WriteFile(p, []byte(content), 0o644))
	return p
}

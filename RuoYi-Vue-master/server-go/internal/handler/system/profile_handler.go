package system

import (
	"github.com/YGeni/ruoyi-go/server-go/internal/model/entity"
	"github.com/YGeni/ruoyi-go/server-go/internal/service"
	"github.com/YGeni/ruoyi-go/server-go/pkg/result"

	"github.com/gin-gonic/gin"
)

// ProfileHandler 个人信息，对应。
type ProfileHandler struct{}

// GetProfile GET /system/user/profile
//
//	AjaxResult ajax = AjaxResult.success(user);      // user 落在 data 里
//	ajax.put("roleGroup", ...);
//	ajax.put("postGroup", ...);
//
// **三个键缺一不可**：前端 src/views/system/user/profile/index.vue:82-83
// 直接读 response.roleGroup / response.postGroup，缺键页面就是空白。
// user 本身在 data 里（前端读 response.data），不能与 roleGroup 平铺混淆。
func (ProfileHandler) GetProfile(c *gin.Context) {
	lu, ok := loginUser(c)
	if !ok {
		return
	}
	roleGroup, err := service.SelectUserRoleGroup(usernameOf(lu))
	if err != nil {
		fail(c, err)
		return
	}
	postGroup, err := service.SelectUserPostGroup(usernameOf(lu))
	if err != nil {
		fail(c, err)
		return
	}
	result.OK(c, result.SuccessData(lu.User).
		Put("roleGroup", roleGroup).
		Put("postGroup", postGroup))
}

// UpdateProfile PUT /system/user/profile
//
// Java 接收整个 SysUser 但**只取四个字段**覆盖 LoginUser（L67-70），
// 这里同样只绑这四个 —— 前端传别的字段（status / deptId / roleIds 等）一律忽略，
// 这是 profile 接口的安全边界。
func (ProfileHandler) UpdateProfile(c *gin.Context) {
	lu, ok := loginUser(c)
	if !ok {
		return
	}
	var body struct {
		NickName    string `json:"nickName"`
		Phonenumber string `json:"phonenumber"`
		Email       string `json:"email"`
		Sex         string `json:"sex"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		fail(c, err)
		return
	}
	in := &entity.SysUser{
		UserID:      lu.UserID,
		NickName:    body.NickName,
		Phonenumber: body.Phonenumber,
		Email:       body.Email,
		Sex:         body.Sex,
	}
	if err := service.UpdateProfile(lu, in); err != nil {
		result.Fail(c, result.Error(err.Error()))
		return
	}
	result.OK(c, result.Success())
}

// UpdatePwd PUT /system/user/profile/updatePwd
//
// 等价：入参是 Map<String,String>
// 即 **oldPassword / newPassword 两个字段**，不是 JSON DTO。
func (ProfileHandler) UpdatePwd(c *gin.Context) {
	lu, ok := loginUser(c)
	if !ok {
		return
	}
	var body struct {
		OldPassword string `json:"oldPassword"`
		NewPassword string `json:"newPassword"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		fail(c, err)
		return
	}
	if err := service.UpdatePwd(lu, body.OldPassword, body.NewPassword); err != nil {
		result.Fail(c, result.Error(err.Error()))
		return
	}
	result.OK(c, result.Success())
}

// Avatar POST /system/user/profile/avatar
//
// 两处契约必须照做：
//  1. 表单字段名是 **avatarfile**（不是 file），Java 是 @RequestParam("avatarfile")
//  2. 返回体必须有 **imgUrl**（前端 src/views/system/user/profile/userAvatar.vue:136 读它）
//
// 另外两个副作用已在 service.UpdateAvatar 里完成：
// 同步 Redis LoginUser 的头像（不然不退出重登看不到新头像）、
// 删除被替换掉的旧头像文件（不然磁盘只增不减）。
func (ProfileHandler) Avatar(c *gin.Context) {
	lu, ok := loginUser(c)
	if !ok {
		return
	}
	fh, err := c.FormFile("avatarfile")
	if err != nil {
		result.Fail(c, result.Error("上传图片异常，请联系管理员"))
		return
	}
	avatar, err := service.UpdateAvatar(lu, fh)
	if err != nil {
		result.Fail(c, result.Error(err.Error()))
		return
	}
	result.OK(c, result.Success().Put("imgUrl", avatar))
}

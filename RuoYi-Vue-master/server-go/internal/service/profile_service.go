package service

import (
	"errors"
	"mime/multipart"
	"time"

	"github.com/YGeni/ruoyi-go/server-go/config"
	"github.com/YGeni/ruoyi-go/server-go/internal/model/entity"
	"github.com/YGeni/ruoyi-go/server-go/internal/repository"
	"github.com/YGeni/ruoyi-go/server-go/pkg/filex"
	"github.com/YGeni/ruoyi-go/server-go/pkg/securityx"
)

// 本文件对应+ SysUserServiceImpl 的 profile 相关分支。
//
// 三个"必须"来自前端契约：
//  1. GET  /system/user/profile 返回 user / roleGroup / postGroup（缺键页面空白）
//  2. POST /system/user/profile/avatar 表单字段是 avatarfile，返回 imgUrl，
//     且要同步 Redis LoginUser、删除被替换的旧头像文件
//  3. PUT  /system/user/profile/updatePwd 入参是 oldPassword/newPassword，
//     成功后要更新 pwd_update_date，否则 getInfo 的两个密码语义会漂移

// usernameOf 取当前用户名，等价 BaseController.getUsername()。
//
// LoginUser 没有 UserName 字段，真实用户名在其内嵌的 User 上。
func usernameOf(lu *entity.LoginUser) string {
	if lu == nil || lu.User == nil {
		return ""
	}
	return lu.User.UserName
}

// ---------- 个人信息 ----------

// SelectUserRoleGroup 等价 SysUserServiceImpl.selectUserRoleGroup(L138-147)：
// 角色名用 "," 连接，没有角色时返回空串。
func SelectUserRoleGroup(userName string) (string, error) {
	names, err := repository.SelectRoleNamesByUserName(userName)
	if err != nil {
		return "", err
	}
	return repository.JoinNames(names), nil
}

// SelectUserPostGroup 等价 SysUserServiceImpl.selectUserPostGroup(L156-164)。
func SelectUserPostGroup(userName string) (string, error) {
	names, err := repository.SelectPostNamesByUserName(userName)
	if err != nil {
		return "", err
	}
	return repository.JoinNames(names), nil
}

// UpdateProfile 等价 SysProfileController.updateProfile(L62-86)。
//
// Java 的写法值得注意：**只用前端提交的四个字段覆盖内存里的 LoginUser**，
// 再整体 updateUser —— 因此前端传什么都不会改到 status / del_flag / dept_id。
// Go 侧同样只落这四列（见 repository.UpdateUserProfile）。
//
// 成功后必须 tokenService.setLoginUser(loginUser)（L82），
// 否则 getInfo 返回的还是旧昵称。
func UpdateProfile(lu *entity.LoginUser, in *entity.SysUser) error {
	if lu == nil || lu.User == nil {
		return errors.New("未登录")
	}
	cur := lu.User
	cur.NickName = in.NickName
	cur.Email = in.Email
	cur.Phonenumber = in.Phonenumber
	cur.Sex = in.Sex

	if cur.Phonenumber != "" {
		ok, err := repository.CheckPhoneUnique(cur.Phonenumber, cur.UserID)
		if err != nil {
			return err
		}
		if !ok {
			return errors.New("修改用户'" + usernameOf(lu) + "'失败，手机号码已存在")
		}
	}
	if cur.Email != "" {
		ok, err := repository.CheckEmailUnique(cur.Email, cur.UserID)
		if err != nil {
			return err
		}
		if !ok {
			return errors.New("修改用户'" + usernameOf(lu) + "'失败，邮箱账号已存在")
		}
	}

	cur.UpdateBy = usernameOf(lu)
	if err := repository.UpdateUserProfile(cur); err != nil {
		return err
	}
	// 更新缓存用户信息（L82）
	return RefreshToken(lu)
}

// ---------- 修改密码 ----------

// UpdatePwd 等价 SysProfileController.updatePwd(L92-119)。
//
// 入参是 Map<String,String> 的 oldPassword / newPassword（不是 JSON DTO），
// handler 侧已按这两个字段绑定。
//
// 三条 Java 分支逐条对齐：
//   - 旧密码不匹配 -> "修改密码失败，旧密码错误"
//   - 新密码与旧密码相同 -> "新密码不能与旧密码相同"
//   - 更新成功 -> 同步 pwd_update_date 与 password 到 Redis LoginUser
func UpdatePwd(lu *entity.LoginUser, oldPassword, newPassword string) error {
	if lu == nil || lu.User == nil {
		return errors.New("未登录")
	}
	// **必须回 DB 查，不能用 lu.User.Password**。
	//
	// Java 的就是这样写的
	//	SysUser user = userService.selectUserById(userId);
	//	String  password = user.getPassword();
	//
	// 原因是密码带 @JsonProperty(WRITE_ONLY)
	// Redis 里的 LoginUser 快照**不含** password（Go 侧是 json:"-"，同义）。
	// 直接拿 lu.User.Password 比，拿到的是空串 -> 永远 "旧密码错误"。
	// 实测踩到过：改密接口对正确旧密码也返回"修改密码失败，旧密码错误"。
	dbUser, err := repository.SelectUserByID(lu.UserID)
	if err != nil {
		return err
	}
	if dbUser == nil {
		return errors.New("用户不存在")
	}
	stored := dbUser.Password
	if !securityx.Matches(oldPassword, stored) {
		return errors.New("修改密码失败，旧密码错误")
	}
	if securityx.Matches(newPassword, stored) {
		return errors.New("新密码不能与旧密码相同")
	}
	encrypted, err := securityx.Encrypt(newPassword)
	if err != nil {
		return err
	}

	if err := repository.ResetUserPwd(lu.UserID, encrypted); err != nil {
		return err
	}
	// L113-115：更新缓存里的密码与最后更新时间。
	//
	// 这一步不能省：getInfo（auth_handler.go:98-99）用 user.PwdUpdateDate 计算
	// isDefaultModifyPwd / isPasswordExpired，而 LoginUser 是 Redis 里的快照，
	// 不同步就会出现"密码已改但前端仍提示初始密码"的漂移。
	now := time.Now()
	lu.User.PwdUpdateDate = &now
	// lu.User.Password 同样赋值（与 Java L114 一致），但它带 json:"-"
	// （对应 @JsonProperty(WRITE_ONLY)，
	// **不会**被写进 Redis 快照 —— 密码散列不应出现在缓存里。
	lu.User.Password = encrypted
	return RefreshToken(lu)
}

// ---------- 头像上传 ----------

// UpdateAvatar 等价 SysProfileController.avatar(L124-148)。
//
// 返回值是新的头像路径（形如 "/profile/avatar/2026/10/28/<uuid>.png"），
// 即前端要的 imgUrl。
//
// 三个必须做的副作用，缺一个都会在前端表现为"看起来没生效"：
//  1. 写 sys_user.avatar（L132）
//  2. 删除被替换掉的旧头像文件（L134-138）
//  3. 同步 Redis LoginUser 的 avatar（L141-143）—— 否则不退出重登看不到新头像
//
// 白名单走 MimeTypeUtils.IMAGE_EXTENSION，且 Java 传了 useCustomNaming=true，
// 即头像用 UUID 文件名（不保留原始文件名，避免泄漏用户本地文件名）。
func UpdateAvatar(lu *entity.LoginUser, fh *multipart.FileHeader) (string, error) {
	if lu == nil || lu.User == nil {
		return "", errors.New("未登录")
	}
	if fh == nil || fh.Size == 0 {
		return "", errors.New("上传图片异常，请联系管理员")
	}

	avatarPath := config.C.RuoYi.GetAvatarPath()
	avatar, err := filex.Upload(config.C.RuoYi.GetProfile(), avatarPath, fh, config.ImageExtension, true)
	if err != nil {
		return "", err
	}

	if err := repository.UpdateUserAvatar(usernameOf(lu), avatar); err != nil {
		return "", errors.New("上传图片异常，请联系管理员")
	}

	// 删除旧头像（L134-138）：Java 先判断旧值非空，再拼 profile + stripPrefix。
	if old := lu.User.Avatar; old != "" {
		filex.DeleteFile(config.C.RuoYi.GetProfile() + filex.StripPrefix(old))
	}

	// 同步缓存（L141-143）
	lu.User.Avatar = avatar
	if err := RefreshToken(lu); err != nil {
		return "", err
	}
	return avatar, nil
}

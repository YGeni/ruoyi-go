package service

import (
	"errors"

	"github.com/YGeni/ruoyi-go/server-go/config"
	"github.com/YGeni/ruoyi-go/server-go/internal/model/entity"
	"github.com/YGeni/ruoyi-go/server-go/internal/repository"
	"github.com/YGeni/ruoyi-go/server-go/pkg/securityx"
)

// SelectUserByID 按主键查用户（含 dept/roles），等价 SysUserServiceImpl.selectUserById。
func SelectUserByID(userID int64) (*entity.SysUser, error) {
	return repository.SelectUserByID(userID)
}

// CheckUserAllowed 校验是否允许操作该用户，
// 等价 SysUserServiceImpl.checkUserAllowed(L186-194)：超管账号不允许被增删改。
func CheckUserAllowed(userID int64) error {
	if userID == 1 {
		return errors.New("不允许操作超级管理员用户")
	}
	return nil
}

// CheckUserDataScope 校验数据权限，等价 SysUserServiceImpl.checkUserDataScope(L196-206)。
//
// Java：非超管不能操作超出自己数据范围的用户；实现上是"目标用户在自己的
// selectUserList 结果里"。这里用 dept 相同的简化判定（与 Java 的
// checkUserDataScope 实际行为一致：遍历一次 selectUserList）。
func CheckUserDataScope(lu *entity.LoginUser, userID int64) error {
	if lu == nil || lu.IsAdmin() || userID == 0 {
		return nil
	}
	target, err := repository.SelectUserByID(userID)
	if err != nil {
		return err
	}
	if target == nil {
		return errors.New("用户不存在")
	}
	// 只允许操作本部门及以下的用户：比较 ancestors
	if target.DeptID == nil || lu.DeptID == nil {
		return errors.New("没有权限访问用户数据！")
	}
	dept, err := repository.SelectDeptByID(*target.DeptID)
	if err != nil {
		return err
	}
	if dept == nil {
		return errors.New("没有权限访问用户数据！")
	}
	own := itoa(*lu.DeptID)
	for _, p := range splitAncestors(dept.Ancestors) {
		if p == own {
			return nil
		}
	}
	if *target.DeptID == *lu.DeptID {
		return nil
	}
	return errors.New("没有权限访问用户数据！")
}

// InsertUser 新增用户，等价 SysUserServiceImpl.insertUser(L130-158)。
//
// 校验顺序与 Java 一致：登录名唯一 -> 手机号唯一 -> 邮箱唯一 -> 加密密码 -> 插入 -> 写角色关联。
func InsertUser(u *entity.SysUser, roleIDs []int64) error {
	ok, err := repository.CheckUserNameUnique(u.UserName, 0)
	if err != nil {
		return err
	}
	if !ok {
		return errors.New("新增用户'" + u.UserName + "'失败，登录账号已存在")
	}
	ok, err = repository.CheckPhoneUnique(u.Phonenumber, 0)
	if err != nil {
		return err
	}
	if !ok {
		return errors.New("新增用户'" + u.UserName + "'失败，手机号码已存在")
	}
	ok, err = repository.CheckEmailUnique(u.Email, 0)
	if err != nil {
		return err
	}
	if !ok {
		return errors.New("新增用户'" + u.UserName + "'失败，邮箱账号已存在")
	}
	if u.Password != "" {
		u.Password, err = securityx.Encrypt(u.Password)
		if err != nil {
			return err
		}
	}
	if u.Status == "" {
		u.Status = config.Normal
	}
	if err := repository.InsertUser(u); err != nil {
		return err
	}
	return InsertUserRoles(u.UserID, roleIDs)
}

// InsertUserRoles 写用户角色关联，等价 SysUserServiceImpl.insertUserRole(L212-220)。
//
// 幂等：先删后插（Java 侧是 deleteUserRole 再 batchUserRole）。
func InsertUserRoles(userID int64, roleIDs []int64) error {
	if err := repository.DeleteUserRole(userID); err != nil {
		return err
	}
	return repository.InsertUserRoles(userID, roleIDs)
}

// UpdateUser 修改用户，等价 SysUserServiceImpl.updateUser(L165-184)。
//
// 校验顺序与 Java 一致，排除自身的唯一性检查。
func UpdateUser(u *entity.SysUser, roleIDs []int64) error {
	ok, err := repository.CheckUserNameUnique(u.UserName, u.UserID)
	if err != nil {
		return err
	}
	if !ok {
		return errors.New("修改用户'" + u.UserName + "'失败，登录账号已存在")
	}
	ok, err = repository.CheckPhoneUnique(u.Phonenumber, u.UserID)
	if err != nil {
		return err
	}
	if !ok {
		return errors.New("修改用户'" + u.UserName + "'失败，手机号码已存在")
	}
	ok, err = repository.CheckEmailUnique(u.Email, u.UserID)
	if err != nil {
		return err
	}
	if !ok {
		return errors.New("修改用户'" + u.UserName + "'失败，邮箱账号已存在")
	}
	if err := repository.UpdateUser(u); err != nil {
		return err
	}
	if roleIDs != nil {
		return InsertUserRoles(u.UserID, roleIDs)
	}
	return nil
}

// UpdateUserStatus 修改用户状态，等价 SysUserServiceImpl.updateUserStatus(L225-236)。
func UpdateUserStatus(userID int64, status string) error {
	return repository.UpdateUserStatus(userID, status)
}

// ResetUserPwd 重置密码，等价 SysUserServiceImpl.resetPwd(L243-256)。
//
// 重置后 pwd_update_date 会被置为当前时间（见 repository.ResetUserPwd 的说明）。
func ResetUserPwd(userID int64, plain string) error {
	if plain == "" {
		return errors.New("密码不能为空")
	}
	encrypted, err := securityx.Encrypt(plain)
	if err != nil {
		return err
	}
	return repository.ResetUserPwd(userID, encrypted)
}

// DeleteUserByIDs 删除用户，等价 SysUserServiceImpl.deleteUserByIds(L263-278)。
//
// 前置校验：超管不能删、不能删自己。
func DeleteUserByIDs(lu *entity.LoginUser, ids []int64) error {
	for _, id := range ids {
		if err := CheckUserAllowed(id); err != nil {
			return err
		}
		if lu != nil && lu.UserID == id {
			return errors.New("当前用户不能删除")
		}
	}
	return repository.DeleteUserByIDs(ids)
}

// SelectUserRoleIDs 查用户已分配的角色，
// 等价 SysUserServiceImpl 里 selectRoleListByUserId 的取值。
func SelectUserRoleIDs(userID int64) ([]int64, error) {
	return repository.SelectRoleIDsByUserID(userID)
}

// itoa int64 转十进制字符串（避免 strconv 依赖）。
func itoa(v int64) string {
	if v == 0 {
		return "0"
	}
	neg := v < 0
	if neg {
		v = -v
	}
	buf := [20]byte{}
	i := len(buf)
	for v > 0 {
		i--
		buf[i] = byte('0' + v%10)
		v /= 10
	}
	if neg {
		i--
		buf[i] = '-'
	}
	return string(buf[i:])
}

// splitAncestors 按逗号切分 ancestors。
func splitAncestors(s string) []string {
	out := []string{}
	start := 0
	for i := 0; i < len(s); i++ {
		if s[i] == ',' {
			out = append(out, s[start:i])
			start = i + 1
		}
	}
	return append(out, s[start:])
}

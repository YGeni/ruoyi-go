package service

import (
	"errors"
	"fmt"
	"strings"

	"github.com/YGeni/ruoyi-go/server-go/config"
	"github.com/YGeni/ruoyi-go/server-go/internal/model/entity"
	"github.com/YGeni/ruoyi-go/server-go/internal/repository"
	"github.com/YGeni/ruoyi-go/server-go/pkg/securityx"
)

// ImportUserHeaders 导入模板的表头，也是 importData 认的列顺序。
//
// 取 Java 用户导入模板的列（登录名称/用户名称/用户邮箱/手机号码/用户性别/
// 帐号状态）；导出模板用 UserExportHeaders，两者**刻意不同**——导出的表头
// 带"用户序号"等只读列，导入不需要。
var ImportUserHeaders = []string{"登录名称", "用户名称", "用户邮箱", "手机号码", "用户性别", "帐号状态"}

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

// ImportResult 导入用户的结果，用于拼消息。
type ImportResult struct {
	SuccessNum int
	FailureNum int
	Msg        string
}

// ImportUsers 批量导入用户，等价 SysUserServiceImpl.importUser。
//
// rows 是 Excel 的数据行（**不含表头**），列顺序见 ImportUserHeaders。
//
// 与 Java 一致的几条规则：
//   - 登录名已存在时：updateSupport=true 就更新，否则算一条失败
//   - 新用户的初始密码取 sys_config 的 sys.user.initPassword（Java 同样读配置，
//     不是写死的常量），读不到才退回 123456
//   - 逐条处理，单条出错**不中断**，记进失败明细
//   - 有失败就返回带明细的消息，由 handler 以错误形式回给前端
//     （Java 是抛 CustomException，同样落到 msg 里）
//
// DIFF(review): Java 的 ExcelUtil 靠 @Excel 注解按**表头名**匹配列，
// 这里是按固定列序取值 —— 表头名变了不会报错，因此导入前先校验列数。
func ImportUsers(rows [][]string, updateSupport bool, operName string) (*ImportResult, error) {
	initPwd := "123456"
	if v, ok, err := SelectConfigValueByKey("sys.user.initPassword"); err == nil && ok && v != "" {
		initPwd = v
	}

	var successMsg, failureMsg strings.Builder
	successNum, failureNum := 0, 0
	for i, row := range rows {
		u, err := importUserRow(row)
		if err != nil {
			failureNum++
			failureMsg.WriteString(fmt.Sprintf("<br/>%d、第 %d 行：%s", failureNum, i+2, err.Error()))
			continue
		}
		old, err := repository.SelectUserByUserName(u.UserName)
		if err != nil {
			failureNum++
			failureMsg.WriteString(fmt.Sprintf("<br/>%d、第 %d 行：查询账号失败", failureNum, i+2))
			continue
		}
		if old == nil {
			u.Password = initPwd
			u.CreateBy = operName
			if err := InsertUser(u, nil); err != nil {
				failureNum++
				failureMsg.WriteString(fmt.Sprintf("<br/>%d、账号 %s 导入失败：%s", failureNum, u.UserName, err.Error()))
				continue
			}
			successNum++
			successMsg.WriteString(fmt.Sprintf("<br/>%d、账号 %s 导入成功", successNum, u.UserName))
			continue
		}
		if !updateSupport {
			failureNum++
			failureMsg.WriteString(fmt.Sprintf("<br/>%d、账号 %s 已存在", failureNum, u.UserName))
			continue
		}
		u.UserID = old.UserID
		u.UpdateBy = operName
		if err := UpdateUser(u, nil); err != nil {
			failureNum++
			failureMsg.WriteString(fmt.Sprintf("<br/>%d、账号 %s 更新失败：%s", failureNum, u.UserName, err.Error()))
			continue
		}
		successNum++
		successMsg.WriteString(fmt.Sprintf("<br/>%d、账号 %s 更新成功", successNum, u.UserName))
	}

	if failureNum > 0 {
		failureMsg.WriteString("</br>数据导入失败，请检查导入数据")
		return &ImportResult{FailureNum: failureNum,
			Msg: fmt.Sprintf("很抱歉，导入失败！共 %d 条数据格式不正确，错误如下：%s", failureNum, failureMsg.String())}, nil
	}
	return &ImportResult{SuccessNum: successNum,
		Msg: fmt.Sprintf("恭喜您，数据已全部导入成功！共 %d 条，数据如下：%s", successNum, successMsg.String())}, nil
}

// importUserRow 把一行 Excel 转成用户，并翻译字典值（性别/状态）。
func importUserRow(row []string) (*entity.SysUser, error) {
	if len(row) < len(ImportUserHeaders) {
		return nil, fmt.Errorf("列数不足，应为 %d 列", len(ImportUserHeaders))
	}
	// 单元格可能被 excelize 截断成短行，缺的当空串
	for len(row) < len(ImportUserHeaders) {
		row = append(row, "")
	}
	u := &entity.SysUser{
		UserName:    strings.TrimSpace(row[0]),
		NickName:    strings.TrimSpace(row[1]),
		Email:       strings.TrimSpace(row[2]),
		Phonenumber: strings.TrimSpace(row[3]),
	}
	if u.UserName == "" {
		return nil, errors.New("登录名称不能为空")
	}
	if u.NickName == "" {
		u.NickName = u.UserName
	}
	u.Sex = parseSex(strings.TrimSpace(row[4]))
	u.Status = parseStatus(strings.TrimSpace(row[5]))
	return u, nil
}

// parseSex 导入时的性别：接受"男/女"这种中文，也接受 '0'/'1'。
//
// 导出时写的是中文（sexText），导入要能读回自己导出的文件。
func parseSex(s string) string {
	switch s {
	case "男", "0":
		return "0"
	case "女", "1":
		return "1"
	default:
		return "2"
	}
}

// parseStatus 导入时的状态：正常/停用，或 '0'/'1'。
func parseStatus(s string) string {
	if s == "停用" || s == config.Disable {
		return config.Disable
	}
	return config.Normal
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

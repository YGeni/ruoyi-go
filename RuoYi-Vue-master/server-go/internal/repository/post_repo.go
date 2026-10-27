package repository

import (
	"errors"

	"github.com/YGeni/ruoyi-go/server-go/internal/model/entity"
	"github.com/YGeni/ruoyi-go/server-go/internal/model/query"
	"github.com/YGeni/ruoyi-go/server-go/pkg/db"
	"github.com/YGeni/ruoyi-go/server-go/pkg/page"

	"gorm.io/gorm"
)

// postListQuery 构造 selectPostList 的基础查询（SysPostMapper.xml:25-38）。
//
// Java：
//
//	select * from sys_post
//	where 1=1 [and post_code like ?] [and status=?] [and post_name like ?]
//
// 岗位**没有** @DataScope 注解（SysPostServiceImpl.selectPostList 无数据权限），
// 因此 scope 恒为 nil。
func postListQuery(q *query.PostList) *gorm.DB {
	tx := db.DB.Table("sys_post")
	if q.PostCode != "" {
		tx = tx.Where("post_code LIKE ?", "%"+q.PostCode+"%")
	}
	if q.PostName != "" {
		tx = tx.Where("post_name LIKE ?", "%"+q.PostName+"%")
	}
	if q.Status != "" {
		tx = tx.Where("status = ?", q.Status)
	}
	return tx
}

// SelectPostList 分页查询岗位，等价 SysPostMapper.xml selectPostList。
func SelectPostList(q *query.PostList, p *page.PageDomain) ([]entity.SysPost, int64, error) {
	var posts []entity.SysPost
	total, err := page.Paginate(postListQuery(q), p, "", "", &posts)
	if err != nil {
		return nil, 0, err
	}
	return posts, total, nil
}

// SelectPostAll 全部岗位（optionselect 用），等价 SysPostMapper.xml selectPostAll。
func SelectPostAll() ([]entity.SysPost, error) {
	var posts []entity.SysPost
	err := db.DB.Order("post_sort").Find(&posts).Error
	return posts, err
}

// SelectPostByID 按主键查岗位，等价 SysPostMapper.xml selectPostById。
func SelectPostByID(postID int64) (*entity.SysPost, error) {
	var x entity.SysPost
	err := db.DB.Where("post_id = ?", postID).First(&x).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &x, nil
}

// CheckPostNameUnique 岗位名是否可用，等价 SysPostServiceImpl.checkPostNameUnique。
func CheckPostNameUnique(name string, excludePostID int64) (bool, error) {
	return postNotExists("post_name", name, excludePostID)
}

// CheckPostCodeUnique 岗位编码是否可用，等价 SysPostServiceImpl.checkPostCodeUnique。
func CheckPostCodeUnique(code string, excludePostID int64) (bool, error) {
	return postNotExists("post_code", code, excludePostID)
}

func postNotExists(col, val string, excludePostID int64) (bool, error) {
	tx := db.DB.Table("sys_post").Where(col+" = ?", val)
	if excludePostID != 0 {
		tx = tx.Where("post_id != ?", excludePostID)
	}
	var n int64
	if err := tx.Count(&n).Error; err != nil {
		return false, err
	}
	return n == 0, nil
}

// InsertPost 新增岗位，等价 SysPostMapper.xml insertPost。
func InsertPost(x *entity.SysPost) error {
	return db.DB.Create(x).Error
}

// UpdatePost 修改岗位，等价 SysPostMapper.xml updatePost 的 <set> 动态列。
func UpdatePost(x *entity.SysPost) error {
	sets := map[string]any{}
	if x.PostCode != "" {
		sets["post_code"] = x.PostCode
	}
	if x.PostName != "" {
		sets["post_name"] = x.PostName
	}
	if x.PostSort != 0 {
		sets["post_sort"] = x.PostSort
	}
	if x.Status != "" {
		sets["status"] = x.Status
	}
	if x.Remark != "" {
		sets["remark"] = x.Remark
	}
	if x.UpdateBy != "" {
		sets["update_by"] = x.UpdateBy
	}
	if len(sets) == 0 {
		return nil
	}
	return db.DB.Table("sys_post").Where("post_id = ?", x.PostID).Updates(sets).Error
}

// DeletePostByIDs **物理删除**岗位，等价 SysPostMapper.xml deletePostByIds。
//
// 与用户/角色/部门的逻辑删除（del_flag='2'）不同：岗位、字典、参数三张表
// Java 侧都是真删，此处保持一致。
func DeletePostByIDs(ids []int64) error {
	if len(ids) == 0 {
		return nil
	}
	return db.DB.Table("sys_post").Where("post_id IN ?", ids).Delete(nil).Error
}

// CountUserPosts 统计岗位关联的用户数，
// 等价 SysUserPostMapper.xml countUserPostById（删除前"已分配不能删"校验）。
func CountUserPosts(postID int64) (int64, error) {
	var n int64
	err := db.DB.Table("sys_user_post").Where("post_id = ?", postID).Count(&n).Error
	return n, err
}

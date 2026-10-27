package service

import (
	"errors"

	"github.com/YGeni/ruoyi-go/server-go/internal/model/entity"
	"github.com/YGeni/ruoyi-go/server-go/internal/model/query"
	"github.com/YGeni/ruoyi-go/server-go/internal/repository"
	"github.com/YGeni/ruoyi-go/server-go/pkg/page"
)

// SelectPostList 岗位列表，等价 SysPostServiceImpl.selectPostList。
//
// 岗位**没有** @DataScope 注解（不在 DataScope 五个落点内），
// 因此不传 scope —— 所有登录用户可见全部岗位。
func SelectPostList(q *query.PostList, p *page.PageDomain) ([]entity.SysPost, int64, error) {
	return repository.SelectPostList(q, p)
}

// SelectPostAll 全部岗位（optionselect 用），等价 SysPostServiceImpl.selectPostAll。
func SelectPostAll() ([]entity.SysPost, error) {
	return repository.SelectPostAll()
}

// SelectPostByID 按主键查岗位，等价 SysPostServiceImpl.selectPostById。
func SelectPostByID(postID int64) (*entity.SysPost, error) {
	return repository.SelectPostByID(postID)
}

// InsertPost 新增岗位，等价 SysPostServiceImpl.insertPost(L85-99)。
func InsertPost(x *entity.SysPost) error {
	ok, err := repository.CheckPostNameUnique(x.PostName, 0)
	if err != nil {
		return err
	}
	if !ok {
		return errors.New("新增岗位'" + x.PostName + "'失败，岗位名称已存在")
	}
	ok, err = repository.CheckPostCodeUnique(x.PostCode, 0)
	if err != nil {
		return err
	}
	if !ok {
		return errors.New("新增岗位'" + x.PostName + "'失败，岗位编码已存在")
	}
	return repository.InsertPost(x)
}

// UpdatePost 修改岗位，等价 SysPostServiceImpl.updatePost(L106-122)。
func UpdatePost(x *entity.SysPost) error {
	ok, err := repository.CheckPostNameUnique(x.PostName, x.PostID)
	if err != nil {
		return err
	}
	if !ok {
		return errors.New("修改岗位'" + x.PostName + "'失败，岗位名称已存在")
	}
	ok, err = repository.CheckPostCodeUnique(x.PostCode, x.PostID)
	if err != nil {
		return err
	}
	if !ok {
		return errors.New("修改岗位'" + x.PostName + "'失败，岗位编码已存在")
	}
	return repository.UpdatePost(x)
}

// DeletePostByIDs 删除岗位，等价 SysPostServiceImpl.deletePostByIds(L129-141)。
//
// 前置校验：岗位已分配给用户时不允许删除。
func DeletePostByIDs(ids []int64) error {
	for _, id := range ids {
		n, err := repository.CountUserPosts(id)
		if err != nil {
			return err
		}
		if n > 0 {
			return errors.New("岗位已分配，不能删除")
		}
	}
	return repository.DeletePostByIDs(ids)
}

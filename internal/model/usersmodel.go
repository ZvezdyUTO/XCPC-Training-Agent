package model

import (
	"aATA/internal/domain"
	"context"

	"gorm.io/gorm"
)

type (
	UsersModel interface {
		Insert(ctx context.Context, data *Users) error
		SystemUser() (*Users, error)
		FindOne(ctx context.Context, id int64) (*Users, error)
		List(ctx context.Context, req *domain.UserListReq) ([]*Users, int64, error)
		Update(ctx context.Context, data *Users) error
		Delete(ctx context.Context, id int64) error
		// DeleteByID 按学号硬删除用户，并依赖数据库级联删除其训练、比赛和同步状态数据。
		DeleteByID(ctx context.Context, id string) error
		FindByID(id string) (*Users, error)
	}

	defaultUsers struct {
		db *gorm.DB
	}
)

func NewUsersModel(db *gorm.DB) UsersModel {
	return &defaultUsers{
		db: db,
	}
}

func (m *defaultUsers) model() *gorm.DB {
	return m.db.Model(&Users{})
}

func (m *defaultUsers) Insert(ctx context.Context, data *Users) error {
	return m.db.Create(data).Error
}

func (m *defaultUsers) List(ctx context.Context, req *domain.UserListReq) ([]*Users, int64, error) {
	var (
		users []*Users
		total int64
	)

	db := m.db.Model(&Users{}) // 选择从Users这张表开始查询

	if len(req.Ids) > 0 {
		db = db.Where("id IN ?", req.Ids)
	}

	if req.Name != "" {
		db = db.Where("name = ?", req.Name)
	}

	if err := db.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	if req.Page > 0 && req.Count > 0 {
		offset := (req.Page - 1) * req.Count
		db = db.Offset(offset).Limit(req.Count)
	}

	if err := db.Find(&users).Error; err != nil {
		return nil, 0, err
	}

	return users, total, nil
}

func (m *defaultUsers) Update(ctx context.Context, data *Users) error {
	return m.db.Save(data).Error
}

func (m *defaultUsers) Delete(ctx context.Context, id int64) error {
	return m.db.Delete(&Users{}, "id = ?", id).Error
}

func (m *defaultUsers) DeleteByID(ctx context.Context, id string) error {
	return m.db.WithContext(ctx).Unscoped().Delete(&Users{}, "id = ?", id).Error
}

func (m *defaultUsers) FindByID(id string) (*Users, error) {
	var res Users
	err := m.db.Where("id =?", id).First(&res).Error

	switch err {
	case nil:
		return &res, nil
	case gorm.ErrRecordNotFound:
		return nil, ErrNotFound
	default:
		return nil, err
	}

}

func (m *defaultUsers) FindOne(ctx context.Context, id int64) (*Users, error) {
	var res Users
	err := m.db.First(&res, "id = ?", id).Error

	switch err {
	case nil:
		return &res, nil
	case gorm.ErrRecordNotFound:
		return nil, nil
	default:
		return nil, err
	}
}

func (m *defaultUsers) SystemUser() (*Users, error) {
	var res Users
	err := m.db.Where("is_system = ?", IsSystemUser).First(&res).Error

	switch err {
	case nil:
		return &res, nil
	case gorm.ErrRecordNotFound:
		return nil, ErrNotFound
	default:
		return nil, err
	}
}

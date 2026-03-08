package model

import (
	"aATA/internal/domain"
	"time"

	"gorm.io/gorm"
)

type UserStatus int

const (
	UserStatusNormal UserStatus = iota + 1
	UserStatusFreeze
	UserStatusLogout
)

const IsSystemUser = 1

type Users struct {
	Id       string     `gorm:"column:id"`
	Name     string     `gorm:"column:name"`
	Password string     `gorm:"column:password"`
	Status   UserStatus `gorm:"column:status"`
	IsSystem int64      `gorm:"column:is_system"`

	CFHandle string `gorm:"column:cf_handle"`
	ACHandle string `gorm:"column:ac_handle"`

	CreateAt time.Time      `gorm:"column:create_at;autoCreateTime"`
	UpdateAt time.Time      `gorm:"column:update_at;autoUpdateTime"`
	DeleteAt gorm.DeletedAt `gorm:"column:delete_at"`
}

func (m *Users) ToDomainUser() *domain.User {
	return &domain.User{
		Id:     m.Id,
		Name:   m.Name,
		Status: int64(m.Status),
	}
}

func (m *Users) TableName() string {
	return "users"
}

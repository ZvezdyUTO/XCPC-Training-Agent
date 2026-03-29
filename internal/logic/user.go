package logic

import (
	"aATA/internal/app/apperr"
	"aATA/internal/model"
	"aATA/pkg/logx"
	"context"
	"errors"
	"fmt"

	"aATA/internal/domain"
	"aATA/pkg/encrypt"
)

type User interface {
	// Login 用户登录逻辑
	Login(ctx context.Context, req *domain.LoginReq) (resp *domain.LoginResp, err error)
	// Register 用户注册逻辑
	Register(ctx context.Context, req *domain.RegisterReq) (resp *domain.RegisterResp, err error)

	// Info 获取用户信息
	Info(ctx context.Context, req int64) (resp *domain.User, err error)
	// Create 管理员创建用户
	Create(ctx context.Context, req []domain.User) (*domain.BatchCreateUsersResp, error)
	// List 管理员查看用户列表
	List(ctx context.Context, d *domain.UserListReq) (*domain.UserListResp, error)

	// DeleteSelf 用户注销自身账号
	DeleteSelf(ctx context.Context, uid int64) error
	// AdminDelete 管理员注销用户账号
	AdminDelete(ctx context.Context, adminID, targetUID int64) error

	// UpPassword 更新密码
	UpPassword(ctx context.Context, uid int64, req *domain.UpPasswordReq) (err error)
}

type user struct {
	usersModel model.UsersModel
}

func NewUser(usersModel model.UsersModel) User {
	return &user{usersModel: usersModel}
}

func (l *user) Login(ctx context.Context, req *domain.LoginReq) (resp *domain.LoginResp, err error) {
	userEntity, err := l.usersModel.FindByID(req.Username)
	if err != nil {
		logx.Errors(ctx, "user", "login_failed", logx.Fields{
			"username": req.Username,
			"reason":   err.Error(),
		})
		return nil, err
	}

	// 使用 encrypt 库验证密码哈希值是否正确
	if !encrypt.VaildPasswordHash(req.Password, userEntity.Password) {
		logx.Infos(ctx, "user", "login_failed", logx.Fields{
			"stage":    "password_check",
			"username": req.Username,
		})
		return nil, apperr.ErrPasswordInvalid
	}

	logx.Infos(ctx, "user", "login_success", logx.Fields{
		"user_id": userEntity.Id,
	})

	// 登录成功，返回登录值
	return &domain.LoginResp{
		Id:       userEntity.Id,
		Name:     userEntity.Name,
		Status:   int64(userEntity.Status),
		IsSystem: userEntity.IsSystem,
	}, nil
}

func (l *user) Register(ctx context.Context, req *domain.RegisterReq) (*domain.RegisterResp, error) {
	// 先检查有没有重复的用户或者用户名
	userEntity, err := l.usersModel.FindByID(req.Id)
	if err != nil {
		// 用户不存在是正常情况，继续走注册
		if !errors.Is(err, model.ErrNotFound) {
			return nil, err
		}
	}
	if userEntity != nil {
		return nil, apperr.ErrUserAlreadyExists
	}

	// 检查两次密码是否正确
	if req.Password != req.Password2 {
		return nil, apperr.ErrPasswordMismatch
	}

	// 设置新用户信息，并且插入，若有报错则记录
	newUser := &domain.User{
		Id:       req.Id,
		Name:     req.Name,
		Password: req.Password,
		Status:   int64(model.UserStatusNormal),
		IsSystem: 0,
	}

	if err := l.createUser(ctx, newUser, "register"); err != nil {
		logx.Errors(ctx, "user", "register_failed", logx.Fields{
			"stage": "create_user",
			"name":  req.Name,
			"error": err.Error(),
		})
		return nil, err
	}

	logx.Infos(ctx, "user", "register_success", logx.Fields{
		"user_id": newUser.Id,
	})

	return &domain.RegisterResp{
		Id:     newUser.Id,
		Name:   newUser.Name,
		Status: int(newUser.Status),
	}, nil
}

func (l *user) Create(ctx context.Context, users []domain.User) (*domain.BatchCreateUsersResp, error) {

	resp := &domain.BatchCreateUsersResp{
		Total:   len(users),
		Success: 0,
		Failed:  make([]domain.BatchCreateFailItem, 0),
	}

	if len(users) == 0 {
		return resp, nil
	}

	for _, u := range users {

		if u.IsSystem == 1 {
			resp.Failed = append(resp.Failed, domain.BatchCreateFailItem{
				StudentID: u.Id,
				Error:     "cannot create system user",
			})
			continue
		}

		exist, err := l.usersModel.FindByID(u.Id)
		if err != nil && !errors.Is(err, model.ErrNotFound) {
			resp.Failed = append(resp.Failed, domain.BatchCreateFailItem{
				StudentID: u.Id,
				Error:     err.Error(),
			})
			continue
		}
		if exist != nil {
			resp.Failed = append(resp.Failed, domain.BatchCreateFailItem{
				StudentID: u.Id,
				Error:     "user already exists",
			})
			continue
		}

		pwdHash, err := encrypt.GenPasswordHash([]byte(u.Password))
		if err != nil {
			resp.Failed = append(resp.Failed, domain.BatchCreateFailItem{
				StudentID: u.Id,
				Error:     err.Error(),
			})
			continue
		}

		err = l.usersModel.Insert(ctx, &model.Users{
			Id:       u.Id,
			Name:     u.Name,
			Password: string(pwdHash),
			Status:   model.UserStatusNormal,
			IsSystem: 0,
			CFHandle: u.CFHandle,
			ACHandle: u.ACHandle,
		})
		if err != nil {
			resp.Failed = append(resp.Failed, domain.BatchCreateFailItem{
				StudentID: u.Id,
				Error:     err.Error(),
			})
			continue
		}

		resp.Success++
	}

	return resp, nil
}

func (l *user) createUser(ctx context.Context, req *domain.User, from string) error {
	// 处理密码等细节，最后插入数据库
	passwordHash, err := encrypt.GenPasswordHash([]byte(req.Password))
	if err != nil {
		return fmt.Errorf("gen password hash failed: %w", err)
	}

	if err := l.usersModel.Insert(ctx, &model.Users{
		Id:       req.Id,
		Name:     req.Name,
		Password: string(passwordHash),
		Status:   model.UserStatus(req.Status),
		IsSystem: req.IsSystem,
		CFHandle: req.CFHandle,
		ACHandle: req.ACHandle,
	}); err != nil {
		// 一律视为系统异常
		return fmt.Errorf("insert user failed: %w", err)
	}

	return nil
}

func (l *user) Info(ctx context.Context, req int64) (resp *domain.User, err error) {
	user, err := l.usersModel.FindOne(ctx, req)
	if err != nil {
		return nil, err
	}
	return user.ToDomainUser(), nil
}

func (l *user) UpPassword(ctx context.Context, uid int64, req *domain.UpPasswordReq) error {
	// 查询用户
	userEntity, err := l.usersModel.FindOne(ctx, uid)
	if err != nil {
		return err
	}

	// 校验旧密码
	ok := encrypt.VaildPasswordHash(
		userEntity.Password,
		req.OldPwd,
	)
	if !ok {
		logx.Infos(ctx, "user", "up_password_failed", logx.Fields{
			"stage": "password_check",
			"uid":   uid,
		})
		return apperr.ErrPasswordInvalid
	}

	// 更改新密码
	if req.NewPwd == "" {
		return apperr.ErrPasswordEmpty
	}
	if req.NewPwd == req.OldPwd {
		return apperr.ErrPasswordSame
	}

	newHash, err := encrypt.GenPasswordHash([]byte(req.NewPwd))
	if err != nil {
		return err
	}

	userEntity.Password = string(newHash)

	return l.usersModel.Update(ctx, userEntity)
}

func (l *user) DeleteSelf(ctx context.Context, uid int64) error {
	err := l.delete(ctx, uid)
	if err != nil {
		if errors.Is(err, apperr.ErrUserNotFound) {
			return err
		}
		logx.Errors(ctx, "user", "delete_self_failed", logx.Fields{
			"stage":   "delete_self",
			"user_id": uid,
			"error":   err.Error(),
		})
		return err
	}
	logx.Infos(ctx, "user", "delete_self_success", logx.Fields{
		"stage":   "delete_self",
		"user_id": uid,
	})
	return nil
}

func (l *user) AdminDelete(ctx context.Context, adminID, targetUID int64) error {
	err := l.delete(ctx, targetUID)
	if err != nil {
		if errors.Is(err, apperr.ErrUserNotFound) {
			return err
		}

		logx.Errors(ctx, "admin", "admin_delete_failed", logx.Fields{
			"stage":      "delete_admin",
			"admin_id":   adminID,
			"target_uid": targetUID,
			"error":      err,
		})
		return err
	}
	logx.Infos(ctx, "admin", "admin_delete_success", logx.Fields{
		"stage":      "delete_admin",
		"admin_id":   adminID,
		"target_uid": targetUID,
	})
	return nil
}

func (l *user) delete(ctx context.Context, uid int64) error {
	userEntity, err := l.usersModel.FindOne(ctx, uid)
	if err != nil {
		// DB 查询失败 → 系统异常
		return fmt.Errorf("find user failed: %w", err)
	}

	if userEntity == nil {
		return apperr.ErrUserNotFound
	}

	if err := l.usersModel.Delete(ctx, uid); err != nil {
		// 删除失败 → 系统异常
		return fmt.Errorf("delete user failed: %w", err)
	}

	return nil
}

func (l *user) List(ctx context.Context, req *domain.UserListReq) (*domain.UserListResp, error) {
	// 直接查表完事了
	users, total, err := l.usersModel.List(ctx, req)
	if err != nil {
		logx.Errors(ctx, "user", "admin_get_list_failed", logx.Fields{
			"stage": "admin_get_list",
			"list":  req.Ids,
			"error": err.Error(),
		})
		return nil, err
	}

	// 返回一个切片，所有用户的列表
	resp := &domain.UserListResp{
		Count: total,
		List:  make([]*domain.User, 0, len(users)),
	}

	for _, u := range users {
		resp.List = append(resp.List, u.ToDomainUser())
	}

	return resp, nil
}

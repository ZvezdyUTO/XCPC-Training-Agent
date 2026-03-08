package domain

type AdminBatchCreateUsersReq struct {
	Users []User `json:"users" binding:"required,dive"`
}

type BatchCreateUsersResp struct {
	Total   int                   `json:"total"`
	Success int                   `json:"success"`
	Failed  []BatchCreateFailItem `json:"failed"`
}

type BatchCreateFailItem struct {
	StudentID string `json:"student_id"`
	Error     string `json:"error"`
}

type User struct {
	Id       string `json:"id"`
	Name     string `json:"name"`
	Password string `json:"password"`
	Status   int64  `json:"status"`
	IsSystem int64  `json:"is_system"`

	CFHandle string `json:"cf_handle"`
	ACHandle string `json:"ac_handle"`

	CreateAt int64 `json:"create_at"`
	UpdateAt int64 `json:"update_at"`
	DeleteAt int64 `json:"delete_at"`
}

type IdReq struct {
	Id int `json:"id" from:"id"`
}

type LoginReq struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

type LoginResp struct {
	Token    string `json:"token"`
	Id       string `json:"id"`
	Name     string `json:"name"`
	Status   int64  `json:"status"`
	IsSystem int64  `json:"is_system"`
}

type RegisterReq struct {
	Id        string `json:"id"`
	Name      string `json:"name"`
	Password  string `json:"password"`
	Password2 string `json:"password2"`
}

type RegisterResp struct {
	Token  string `json:"token"`
	Id     string `json:"id"`
	Name   string `json:"name"`
	Status int    `json:"status"`
}

type UserListReq struct {
	Ids   []string `json:"ids,omitempty" query:"ids"`
	Name  string   `json:"name,omitempty" query:"name"`
	Page  int      `json:"page,omitempty" query:"page"`
	Count int      `json:"count,omitempty" query:"count"`
}

type UserListResp struct {
	Count int64 `json:"count"`
	List  []*User
}

type UpPasswordReq struct {
	Id     string `json:"id"`
	OldPwd string `json:"oldPwd"`
	NewPwd string `json:"newPwd"`
}

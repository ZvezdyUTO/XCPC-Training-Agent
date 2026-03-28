package api

import (
	"aATA/internal/logic"
	agentservice "aATA/internal/logic/agent/service"
	"aATA/internal/logic/student_data"
	"aATA/internal/svc"
	"time"
)

// initHandler 实例化 Logic 层和 Handler 层，执行路由分发与依赖装配。
func initHandler(svc *svc.ServiceContext) []Handler {
	// 实例化 Logic，拆分一类并且使用构造函数注入，使每个模块值依赖自身接口，并且保持边界清晰。
	loc, _ := time.LoadLocation("Asia/Shanghai")
	var (
		userLogic     = logic.NewUser(svc.UsersModel)
		trainingLogic = student_data.NewTrainingLogic(
			svc.UsersModel,
			svc.ContestModel,
			svc.DailyModel,
			svc.StudentSyncStateModel,
			svc.Crawler,
			loc,
		)
		agentLogic = agentservice.New(svc)
	)

	// 实例化 Handler，将创建好的 Logic 实例注入
	var (
		userSelf   = NewUserSelf(svc, userLogic)
		adminUser  = NewAdminUser(svc, userLogic)
		userPublic = NewUserPublic(svc, userLogic)
		adminOp    = NewAdminOperator(svc, trainingLogic)
		adminAgent = NewAdminAgent(svc, agentLogic)
	)

	// 将所有实例化的 Handler 放入切片中返回
	// 这样组装，明确了项目中各个组件之间的依赖关系，通过模块化管理，将功能进行拆分，且为上层提供了一个整齐的处理器列表
	return []Handler{
		userSelf,
		adminUser,
		userPublic,
		adminOp,
		adminAgent,
	}
}

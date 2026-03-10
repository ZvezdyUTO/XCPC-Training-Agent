package task

import (
	"aATA/internal/logic/student_data"
	"aATA/internal/svc"
	"aATA/pkg/logx"
	"context"
	"time"

	"github.com/jasonlvhit/gocron"
)

type DailyTrainingSync struct {
	svc      *svc.ServiceContext
	training student_data.TrainingLogic
	loc      *time.Location
}

func NewDailyTrainingSync(svc *svc.ServiceContext, loc *time.Location) *DailyTrainingSync {
	return &DailyTrainingSync{
		svc: svc,
		training: student_data.NewTrainingLogic(
			svc.UsersModel,
			svc.ContestModel,
			svc.DailyModel,
			svc.Crawler,
			loc,
		),
		loc: loc,
	}
}

// Fix 启动前可做一些校验/预热（这里先留空）
func (s *DailyTrainingSync) Fix(ctx context.Context) {}

// Register 注册到 gocron
func (s *DailyTrainingSync) Register(ctx context.Context) {
	// gocron 的 At 使用本地时区（time.Local），与 NewTrainingLogic 保持一致
	_ = gocron.Every(1).Day().At("00:05").Do(func() {
		// 不直接用 ctx（它是 runner 的长生命周期 ctx），每次跑任务都应当派生一个带超时的 ctx
		runCtx, cancel := context.WithTimeout(ctx, 60*time.Minute)
		defer cancel()

		// 运行定时任务
		if err := s.getData(runCtx); err != nil {
			logx.Errors(runCtx, "task", "daily_training_sync_failed", logx.Fields{
				"error": err.Error(),
			})
		}
	})
}

// Stop 退出时收尾
func (s *DailyTrainingSync) Stop(ctx context.Context) {}

func (s *DailyTrainingSync) getData(ctx context.Context) error {
	// 先同步昨日训练数据
	if err := s.training.SyncAllUsersYesterday(ctx); err != nil {
		return err
	}

	return nil
}

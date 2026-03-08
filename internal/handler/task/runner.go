package task

import (
	"aATA/internal/svc"
	"context"
	"time"

	"github.com/jasonlvhit/gocron"
)

type Runner struct {
	svc *svc.ServiceContext
}

func NewRunner(svc *svc.ServiceContext) *Runner {
	return &Runner{svc: svc}
}

type Service interface {
	// Fix 启动前修复
	Fix(ctx context.Context)
	// Register 执行定时任务
	Register(ctx context.Context)
	// Stop 停止时执行最终 flush 收尾
	Stop(ctx context.Context)
}

// Start 阻塞直到 ctx 取消
func (r *Runner) Start(ctx context.Context) error {
	// 这里代表了多个定时任务，只要定时任务包含修复、执行、安全停止逻辑，就能被接入并且启动
	loc, _ := time.LoadLocation("Asia/Shanghai")
	gocron.ChangeLoc(loc)
	services := []Service{
		NewDailyTrainingSync(r.svc, loc),
	}

	for _, s := range services {
		s.Fix(ctx)
	}
	for _, s := range services {
		s.Register(ctx)
	}

	done := make(chan struct{})
	go func() {
		defer close(done) // 等自动任务结束后，关闭管道，发送信息
		<-gocron.Start()
	}()

	select {
	case <-ctx.Done(): // 如果是系统退出
		// 1) 停止未来调度
		gocron.Clear()

		// 2) 等调度器 goroutine 收敛
		// 同步管道，在信息进来之前处于阻塞状态
		// 真正的作用是建立 happens-before 关系，保证 scheduler 彻底停止才执行后续的 Stop()
		<-done

		// 3) 收尾（最终 flush），给一个短超时避免卡死
		stopCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		for _, s := range services {
			s.Stop(stopCtx)
		}
		return ctx.Err()
	case <-done:
		return nil
	}
}

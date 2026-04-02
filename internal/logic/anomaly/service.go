package anomaly

import (
	"aATA/internal/domain"
	"aATA/internal/model"
	"context"
	"fmt"
	"sync"
	"time"
)

const (
	volumeDropAlertType     = "volume_drop_7d"
	inactiveDaysAlertType   = "inactive_days"
	difficultyDropAlertType = "difficulty_drop_7d"
)

// Service 定义训练异常检测的最小入口。
// 当前先实现“训练量突降”规则，后续规则可以按同样方式扩展。
type Service interface {
	DetectAllUsers(ctx context.Context, now time.Time) (int, error)
	ListAlerts(ctx context.Context, req *domain.AdminAlertListReq) (*domain.AdminAlertListResp, error)
	AckAlert(ctx context.Context, id int64) error
	ResolveAlert(ctx context.Context, id int64) error
	ResolveAllAlerts(ctx context.Context) (int64, error)
	GetRuleConfig(ctx context.Context) RuleConfig
	UpdateRuleConfig(ctx context.Context, cfg RuleConfig) error
	PatchRuleConfig(ctx context.Context, patch RuleConfigPatch) (RuleConfig, error)
}

type service struct {
	users   model.UsersModel
	daily   model.DailyTrainingStatsModel
	contest model.ContestRecordModel
	alerts  model.TrainingAlertModel
	configs model.AnomalyRuleConfigModel
	cfg     RuleConfig
	mu      sync.RWMutex
}

// New 创建异常检测服务。
func New(
	users model.UsersModel,
	daily model.DailyTrainingStatsModel,
	contest model.ContestRecordModel,
	alerts model.TrainingAlertModel,
	configs model.AnomalyRuleConfigModel,
) Service {
	return &service{
		users:   users,
		daily:   daily,
		contest: contest,
		alerts:  alerts,
		configs: configs,
		cfg:     defaultRuleConfig(),
	}
}

// DetectAllUsers 对所有普通学生执行异常检测，并写入预警表。
// 返回本次写入（含 upsert 更新）的预警条数。
func (s *service) DetectAllUsers(ctx context.Context, now time.Time) (int, error) {
	if err := s.loadRuleConfigIfExists(ctx); err != nil {
		return 0, err
	}

	cfg := s.getConfig()
	if err := cfg.Validate(); err != nil {
		return 0, fmt.Errorf("异常检测规则配置非法: %w", err)
	}

	users, _, err := s.users.List(ctx, &domain.UserListReq{})
	if err != nil {
		return 0, err
	}

	asOf := dateOnly(now)
	toUpsert := make([]*model.TrainingAlert, 0, len(users))

	for _, user := range users {
		if user.IsSystem == model.IsSystemUser {
			continue
		}
		alert, ok, err := s.detectVolumeDrop(ctx, user.Id, asOf)
		if err != nil {
			return 0, err
		}
		if ok {
			toUpsert = append(toUpsert, alert)
		}

		alert, ok, err = s.detectInactiveDays(ctx, user.Id, asOf)
		if err != nil {
			return 0, err
		}
		if ok {
			toUpsert = append(toUpsert, alert)
		}

		alert, ok, err = s.detectDifficultyDrop(ctx, user.Id, asOf)
		if err != nil {
			return 0, err
		}
		if ok {
			toUpsert = append(toUpsert, alert)
		}
	}

	if len(toUpsert) == 0 {
		return 0, nil
	}
	if err := s.alerts.UpsertMany(ctx, toUpsert); err != nil {
		return 0, err
	}

	return len(toUpsert), nil
}

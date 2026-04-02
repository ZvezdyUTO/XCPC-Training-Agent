package anomaly

import (
	"aATA/internal/domain"
	"aATA/internal/model"
	"context"
	"encoding/json"
	"fmt"
	"time"
)

func (s *service) ListAlerts(
	ctx context.Context,
	req *domain.AdminAlertListReq,
) (*domain.AdminAlertListResp, error) {
	if req == nil {
		req = &domain.AdminAlertListReq{}
	}

	query, err := buildAlertListQuery(req)
	if err != nil {
		return nil, err
	}

	list, total, err := s.alerts.List(ctx, query)
	if err != nil {
		return nil, err
	}

	out := make([]domain.AdminAlertItem, 0, len(list))
	for _, item := range list {
		if item == nil {
			continue
		}
		out = append(out, toDomainAlertItem(item))
	}

	return &domain.AdminAlertListResp{
		Count: total,
		List:  out,
	}, nil
}

func (s *service) AckAlert(ctx context.Context, id int64) error {
	if id <= 0 {
		return fmt.Errorf("预警 ID 必须大于 0")
	}
	return s.alerts.UpdateStatus(ctx, id, model.AlertStatusAck)
}

func (s *service) ResolveAlert(ctx context.Context, id int64) error {
	if id <= 0 {
		return fmt.Errorf("预警 ID 必须大于 0")
	}
	return s.alerts.UpdateStatus(ctx, id, model.AlertStatusResolved)
}

func buildAlertListQuery(req *domain.AdminAlertListReq) (*model.TrainingAlertListQuery, error) {
	query := &model.TrainingAlertListQuery{
		StudentID: req.StudentID,
		Status:    req.Status,
		Severity:  req.Severity,
		Page:      req.Page,
		Count:     req.Count,
	}

	if req.From != "" {
		from, err := time.Parse(time.DateOnly, req.From)
		if err != nil {
			return nil, fmt.Errorf("开始日期格式非法，应为 YYYY-MM-DD")
		}
		query.From = &from
	}
	if req.To != "" {
		to, err := time.Parse(time.DateOnly, req.To)
		if err != nil {
			return nil, fmt.Errorf("结束日期格式非法，应为 YYYY-MM-DD")
		}
		query.To = &to
	}

	if query.From != nil && query.To != nil && query.To.Before(*query.From) {
		return nil, fmt.Errorf("结束日期不能早于开始日期")
	}

	return query, nil
}

func toDomainAlertItem(item *model.TrainingAlert) domain.AdminAlertItem {
	evidence := map[string]interface{}{}
	_ = json.Unmarshal(item.Evidence, &evidence)

	actions := make([]string, 0, 4)
	_ = json.Unmarshal(item.Actions, &actions)

	return domain.AdminAlertItem{
		ID:        item.ID,
		StudentID: item.StudentID,
		AlertDate: item.AlertDate.Format(time.DateOnly),
		AlertType: item.AlertType,
		Severity:  item.Severity,
		Status:    item.Status,
		Title:     item.Title,
		Evidence:  evidence,
		Actions:   actions,
		CreatedAt: item.CreatedAt.Format("2006-01-02 15:04:05"),
		UpdatedAt: item.UpdatedAt.Format("2006-01-02 15:04:05"),
	}
}

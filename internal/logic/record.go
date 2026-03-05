package logic

//
//import (
//	"aATA/internal/domain"
//	"aATA/internal/model"
//	"context"
//	"errors"
//	"time"
//
//	"github.com/go-sql-driver/mysql"
//)
//
//type Record interface {
//
//	// ====== 批量导入 ======
//
//	ImportContestRecords(ctx context.Context, req *domain.ImportContestRecordsReq) error
//	ImportTrainingStats(ctx context.Context, req *domain.ImportTrainingStatsReq) error
//
//	// ====== 删除 ======
//
//	DeleteContestRange(ctx context.Context, req *domain.DeleteContestRangeReq) error
//	DeleteTrainingRange(ctx context.Context, req *domain.DeleteTrainingRangeReq) error
//
//	DeleteContestByID(ctx context.Context, req *domain.DeleteContestByIDReq) error
//	DeleteTrainingByDate(ctx context.Context, req *domain.DeleteTrainingByDateReq) error
//}
//
//type record struct {
//	contestModel  model.ContestRecordModel
//	trainingModel model.DailyTrainingStatsModel
//}
//
//func NewRecord(
//	contestModel model.ContestRecordModel,
//	trainingModel model.DailyTrainingStatsModel,
//) Record {
//	return &record{
//		contestModel:  contestModel,
//		trainingModel: trainingModel,
//	}
//}
//
//func (r *record) ImportContestRecords(
//	ctx context.Context,
//	req *domain.ImportContestRecordsReq,
//) error {
//
//	for _, item := range req.Records {
//		item.StudentID = req.StudentID
//
//		m := model.ToModelContest(item)
//		err := r.contestModel.Insert(ctx, m)
//		if err != nil {
//			if isDuplicateErr(err) {
//				continue
//			}
//			return err
//		}
//	}
//
//	return nil
//}
//
//func (r *record) ImportTrainingStats(
//	ctx context.Context,
//	req *domain.ImportTrainingStatsReq,
//) error {
//
//	for _, item := range req.Stats {
//		item.StudentID = req.StudentID
//		item.Date = normalizeDate(item.Date)
//
//		m := model.ToModelDaily(item)
//		if err := r.trainingModel.Upsert(ctx, m); err != nil {
//			return err
//		}
//	}
//
//	return nil
//}
//
//func (r *record) DeleteContestRange(
//	ctx context.Context,
//	req *domain.DeleteContestRangeReq,
//) error {
//
//	return r.contestModel.DeleteRange(
//		ctx,
//		req.StudentIDs,
//		req.From,
//		req.To,
//	)
//}
//
//func (r *record) DeleteTrainingRange(
//	ctx context.Context,
//	req *domain.DeleteTrainingRangeReq,
//) error {
//
//	return r.trainingModel.DeleteRange(
//		ctx,
//		req.StudentIDs,
//		req.From,
//		req.To,
//	)
//}
//
//func (r *record) DeleteContestByID(
//	ctx context.Context,
//	req *domain.DeleteContestByIDReq,
//) error {
//
//	return r.contestModel.Delete(
//		ctx,
//		req.StudentID,
//		req.Platform,
//		req.ContestID,
//	)
//}
//func (r *record) DeleteTrainingByDate(
//	ctx context.Context,
//	req *domain.DeleteTrainingByDateReq,
//) error {
//
//	return r.trainingModel.DeleteByDate(
//		ctx,
//		req.StudentID,
//		normalizeDate(req.Date),
//	)
//}
//
//func normalizeDate(t time.Time) time.Time {
//	return time.Date(
//		t.Year(),
//		t.Month(),
//		t.Day(),
//		0, 0, 0, 0,
//		t.Location(),
//	)
//}
//
//func isDuplicateErr(err error) bool {
//	var mysqlErr *mysql.MySQLError
//	if errors.As(err, &mysqlErr) {
//		return mysqlErr.Number == 1062
//	}
//	return false
//}

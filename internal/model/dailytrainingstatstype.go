package model

import (
	"aATA/internal/domain"
	"time"

	"gorm.io/gorm"
)

type DailyTrainingStats struct {
	StudentID string    `gorm:"column:student_id;primaryKey"`
	StatDate  time.Time `gorm:"column:stat_date;primaryKey;type:date"`

	// Codeforces 当日新增
	CFNewTotal     int `gorm:"column:cf_new_total"`
	CFNewUndefined int `gorm:"column:cf_new_undefined"`
	CFNew800       int `gorm:"column:cf_new_800"`
	CFNew900       int `gorm:"column:cf_new_900"`
	CFNew1000      int `gorm:"column:cf_new_1000"`
	CFNew1100      int `gorm:"column:cf_new_1100"`
	CFNew1200      int `gorm:"column:cf_new_1200"`
	CFNew1300      int `gorm:"column:cf_new_1300"`
	CFNew1400      int `gorm:"column:cf_new_1400"`
	CFNew1500      int `gorm:"column:cf_new_1500"`
	CFNew1600      int `gorm:"column:cf_new_1600"`
	CFNew1700      int `gorm:"column:cf_new_1700"`
	CFNew1800      int `gorm:"column:cf_new_1800"`
	CFNew1900      int `gorm:"column:cf_new_1900"`
	CFNew2000      int `gorm:"column:cf_new_2000"`
	CFNew2100      int `gorm:"column:cf_new_2100"`
	CFNew2200      int `gorm:"column:cf_new_2200"`
	CFNew2300      int `gorm:"column:cf_new_2300"`
	CFNew2400      int `gorm:"column:cf_new_2400"`
	CFNew2500      int `gorm:"column:cf_new_2500"`
	CFNew2600      int `gorm:"column:cf_new_2600"`
	CFNew2700      int `gorm:"column:cf_new_2700"`
	CFNew2800Plus  int `gorm:"column:cf_new_2800_plus"`

	// AtCoder 当日新增
	ACNewTotal     int `gorm:"column:ac_new_total"`
	ACNewUndefined int `gorm:"column:ac_new_undefined"`
	ACNew0_399     int `gorm:"column:ac_new_0_399"`
	ACNew400_799   int `gorm:"column:ac_new_400_799"`
	ACNew800_1199  int `gorm:"column:ac_new_800_1199"`
	ACNew1200_1599 int `gorm:"column:ac_new_1200_1599"`
	ACNew1600_1999 int `gorm:"column:ac_new_1600_1999"`
	ACNew2000_2399 int `gorm:"column:ac_new_2000_2399"`
	ACNew2400_2799 int `gorm:"column:ac_new_2400_2799"`
	ACNew2800Plus  int `gorm:"column:ac_new_2800_plus"`

	CreatedAt time.Time      `gorm:"column:created_at;autoCreateTime"`
	DeletedAt gorm.DeletedAt `gorm:"column:deleted_at;index"`
}

type RankItem struct {
	StudentID string
	Total     int
}

func ToModelDaily(d *domain.DailyTrainingStats) *DailyTrainingStats {
	return &DailyTrainingStats{
		StudentID: d.StudentID,
		StatDate:  d.Date,

		CFNewTotal:     d.CFNewTotal,
		CFNewUndefined: d.CFNewUndefined,
		CFNew800:       d.CFNew[800],
		CFNew900:       d.CFNew[900],
		CFNew1000:      d.CFNew[1000],
		CFNew1100:      d.CFNew[1100],
		CFNew1200:      d.CFNew[1200],
		CFNew1300:      d.CFNew[1300],
		CFNew1400:      d.CFNew[1400],
		CFNew1500:      d.CFNew[1500],
		CFNew1600:      d.CFNew[1600],
		CFNew1700:      d.CFNew[1700],
		CFNew1800:      d.CFNew[1800],
		CFNew1900:      d.CFNew[1900],
		CFNew2000:      d.CFNew[2000],
		CFNew2100:      d.CFNew[2100],
		CFNew2200:      d.CFNew[2200],
		CFNew2300:      d.CFNew[2300],
		CFNew2400:      d.CFNew[2400],
		CFNew2500:      d.CFNew[2500],
		CFNew2600:      d.CFNew[2600],
		CFNew2700:      d.CFNew[2700],
		CFNew2800Plus:  d.CFNew[2800],

		ACNewTotal:     d.ACNewTotal,
		ACNewUndefined: d.ACNewUndefined,
		ACNew0_399:     d.ACNewRange["0-399"],
		ACNew400_799:   d.ACNewRange["400-799"],
		ACNew800_1199:  d.ACNewRange["800-1199"],
		ACNew1200_1599: d.ACNewRange["1200-1599"],
		ACNew1600_1999: d.ACNewRange["1600-1999"],
		ACNew2000_2399: d.ACNewRange["2000-2399"],
		ACNew2400_2799: d.ACNewRange["2400-2799"],
		ACNew2800Plus:  d.ACNewRange["2800+"],
	}
}

func (DailyTrainingStats) TableName() string {
	return "daily_training_stats"
}

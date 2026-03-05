package model

import (
	"aATA/internal/domain"
	"time"

	"gorm.io/gorm"
)

type ContestRecord struct {
	StudentID string `gorm:"column:student_id;primaryKey";`
	Platform  string `gorm:"column:platform;primaryKey"` // CF / AC
	ContestID string `gorm:"column:contest_id;primaryKey"`

	ContestName string    `gorm:"column:contest_name"`
	ContestDate time.Time `gorm:"column:contest_date;index"`

	ContestRank  int `gorm:"column:contest_rank"`
	OldRating    int `gorm:"column:old_rating"`
	NewRating    int `gorm:"column:new_rating"`
	RatingChange int `gorm:"column:rating_change"`

	Performance int `gorm:"column:performance"`

	CreatedAt time.Time      `gorm:"column:created_at;autoCreateTime"`
	DeletedAt gorm.DeletedAt `gorm:"column:deleted_at;index"`
}

func ToModelContest(d *domain.ContestRecord) *ContestRecord {
	return &ContestRecord{
		StudentID:    d.StudentID,
		Platform:     d.Platform,
		ContestID:    d.ContestID,
		ContestName:  d.Name,
		ContestDate:  d.Date,
		ContestRank:  d.Rank,
		OldRating:    d.OldRating,
		NewRating:    d.NewRating,
		RatingChange: d.RatingChange,
		Performance:  d.Performance,
	}
}

func (ContestRecord) TableName() string {
	return "contest_records"
}

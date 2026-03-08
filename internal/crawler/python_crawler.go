package crawler

import (
	"aATA/internal/domain"
	"context"
	"encoding/json"
	"errors"
	"os/exec"
	"time"
)

type PythonCrawler struct {
	ScriptPath string
	PythonBin  string
}

// FetchResponse 返回查询结果：学号姓名，查询时间段中每日训练记录和比赛记录
type FetchResponse struct {
	StudentID      string                      `json:"student_id"`
	From           string                      `json:"from"`
	To             string                      `json:"to"`
	ContestRecords []domain.ContestRecord      `json:"contest_records"`
	DailyStats     []domain.DailyTrainingStats `json:"daily_stats"`
}

// Crawler 爬虫接口，传入学号、ID以及时间段，返回查询结果
type Crawler interface {
	FetchRange(
		ctx context.Context,
		studentID string,
		cfHandle string,
		acHandle string,
		from time.Time,
		to time.Time,
	) (*FetchResponse, error)
}

func (p *PythonCrawler) FetchRange(
	ctx context.Context,
	studentID string,
	cfHandle string,
	acHandle string,
	from time.Time,
	to time.Time,
) (*FetchResponse, error) {

	cmd := exec.CommandContext(
		ctx,
		p.PythonBin,
		p.ScriptPath,
		"--student", studentID,
		"--cf", cfHandle,
		"--ac", acHandle,
		"--from", from.Format("2006-01-02"),
		"--to", to.Format("2006-01-02"),
	)

	output, err := cmd.CombinedOutput()
	if err != nil {
		return nil, errors.New(string(output))
	}

	var resp FetchResponse
	if err := json.Unmarshal(output, &resp); err != nil {
		return nil, err
	}

	return &resp, nil
}

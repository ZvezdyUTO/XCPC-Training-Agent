package crawler

import (
	"context"
	"testing"
	"time"
)

// TestPythonCrawler_RealNetwork 爬虫部分单元测试，爬取数据并且组装为结构体返回
func TestPythonCrawler_RealNetwork(t *testing.T) {

	c := &PythonCrawler{
		ScriptPath: "crawler_cli.py", // 根据实际路径修改
		PythonBin:  "/home/zvezdyuto/GolandProjects/agentAcmTrainingAnalysis/venv/bin/python",
	}

	from := time.Date(2025, 3, 1, 0, 0, 0, 0, time.Local)
	to := time.Date(2025, 4, 1, 0, 0, 0, 0, time.Local)

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	resp, err := c.FetchRange(
		ctx,
		"xxx",
		"",
		"",
		from,
		to,
	)

	if err != nil {
		t.Fatalf("FetchRange failed: %v", err)
	}

	if resp == nil {
		t.Fatal("resp is nil")
	}

	if resp.StudentID != "xxx" {
		t.Fatalf("unexpected student id: %s", resp.StudentID)
	}

	t.Logf("contest count: %d", len(resp.ContestRecords))
	t.Logf("daily count: %d", len(resp.DailyStats))
}

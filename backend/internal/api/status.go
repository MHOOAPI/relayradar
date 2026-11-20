package api

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"relayradar/internal/store"
)

type statusComponent struct {
	Key    string `json:"key"`
	Name   string `json:"name"`
	Status string `json:"status"` // operational | degraded | outage | unknown
	Detail string `json:"detail"`
}

type statusDay struct {
	Date   string  `json:"date"`
	Status string  `json:"status"`
	Uptime float64 `json:"uptime"`
}

type componentHistory struct {
	Key  string      `json:"key"`
	Days []statusDay `json:"days"`
}

type statusResponse struct {
	Status     string            `json:"status"`
	CheckedAt  time.Time         `json:"checkedAt"`
	StartedAt  time.Time         `json:"startedAt"`
	UptimeSecs int64             `json:"uptimeSeconds"`
	Version    string            `json:"version"`
	Components []statusComponent `json:"components"`
	History    []componentHistory `json:"history"`
	Incidents  []store.Incident  `json:"incidents"`
}

func worstStatus(list []statusComponent) string {
	rank := map[string]int{"operational": 0, "unknown": 1, "degraded": 2, "outage": 3}
	worst := "operational"
	for _, c := range list {
		if rank[c.Status] > rank[worst] {
			worst = c.Status
		}
	}
	return worst
}

// collectStatus evaluates every component against real dependencies.
func (s *Server) collectStatus(ctx context.Context) statusResponse {
	now := time.Now().UTC()
	comps := []statusComponent{
		{Key: "api", Name: "公开 API", Status: "operational", Detail: "请求正常返回"},
	}

	d, err := s.svc.Dataset(ctx)
	dbStatus := statusComponent{Key: "database", Name: "数据存储", Status: "operational", Detail: "快照读取正常"}
	probeStatus := statusComponent{Key: "probes", Name: "探针采集", Status: "unknown", Detail: "暂无数据"}
	if err != nil {
		dbStatus.Status = "outage"
		dbStatus.Detail = "快照读取失败"
	} else {
		probeStatus.Detail = fmt.Sprintf("最近一次数据快照 %s 前", now.Sub(d.TakenAt).Round(time.Minute))
		switch age := now.Sub(d.TakenAt); {
		case age > 24*time.Hour:
			probeStatus.Status = "outage"
			probeStatus.Detail = "超过 24 小时没有新探针"
		case age > 6*time.Hour:
			probeStatus.Status = "degraded"
			probeStatus.Detail = fmt.Sprintf("数据已 %s 未更新", age.Round(time.Hour))
		default:
			probeStatus.Status = "operational"
		}
	}
	comps = append(comps, dbStatus, probeStatus)

	subs, _ := s.svc.Repo().ListSubscriptions(ctx)
	failing := 0
	for _, sub := range subs {
		if sub.Attempts >= 3 {
			failing++
		}
	}
	subStatus := statusComponent{Key: "alerts", Name: "告警投递", Status: "operational", Detail: fmt.Sprintf("%d 个订阅运行中", len(subs))}
	if failing > 0 {
		subStatus.Status = "degraded"
		subStatus.Detail = fmt.Sprintf("%d 个订阅连续失败，等待重试", failing)
	}
	comps = append(comps, subStatus)

	auditStatus := statusComponent{Key: "audit", Name: "接口检测", Status: "operational", Detail: "检测服务可用"}
	if !s.cfg.AuditEnabled {
		auditStatus.Status = "degraded"
		auditStatus.Detail = "检测功能在此部署中关闭"
	}
	emailStatus := statusComponent{Key: "email", Name: "邮件通道", Status: "operational", Detail: "SMTP 已配置"}
	if !s.cfg.SMTPConfigured() {
		emailStatus.Status = "unknown"
		emailStatus.Detail = "SMTP 未配置，邮件订阅不可用"
	}
	comps = append(comps, auditStatus, emailStatus)

	return statusResponse{
		Status:     worstStatus(comps),
		CheckedAt:  now,
		StartedAt:  s.startedAt,
		UptimeSecs: int64(time.Since(s.startedAt).Seconds()),
		Version:    "2.0",
		Components: comps,
	}
}

func (s *Server) handleStatus(w http.ResponseWriter, r *http.Request) {
	out := s.collectStatus(r.Context())

	since := time.Now().UTC().AddDate(0, 0, -89)
	if rows, err := s.svc.Repo().StatusHistory(r.Context(), since); err == nil {
		out.History = buildStatusHistory(rows, since)
	} else {
		out.History = []componentHistory{}
	}
	if incidents, err := s.svc.Repo().ListIncidents(r.Context(), true); err == nil {
		if incidents == nil {
			incidents = []store.Incident{}
		}
		out.Incidents = incidents
	} else {
		out.Incidents = []store.Incident{}
	}
	writeJSON(w, http.StatusOK, out)
}

type dayAgg struct {
	total int
	ok    int
	worst string
}

func buildStatusHistory(rows []store.StatusSnapshot, since time.Time) []componentHistory {
	const days = 90
	byComp := map[string]map[string]*dayAgg{}
	order := []string{}
	rank := map[string]int{"operational": 0, "unknown": 1, "degraded": 2, "outage": 3}
	for _, row := range rows {
		if _, ok := byComp[row.Component]; !ok {
			byComp[row.Component] = map[string]*dayAgg{}
			order = append(order, row.Component)
		}
		date := row.CheckedAt.UTC().Format("2006-01-02")
		d := byComp[row.Component][date]
		if d == nil {
			d = &dayAgg{worst: "operational"}
			byComp[row.Component][date] = d
		}
		d.total++
		if row.Status == "operational" {
			d.ok++
		}
		if rank[row.Status] > rank[d.worst] {
			d.worst = row.Status
		}
	}
	out := make([]componentHistory, 0, len(order))
	for _, key := range order {
		ch := componentHistory{Key: key, Days: make([]statusDay, 0, days)}
		for i := 0; i < days; i++ {
			date := since.AddDate(0, 0, i).Format("2006-01-02")
			d := byComp[key][date]
			day := statusDay{Date: date, Status: "unknown"}
			if d != nil && d.total > 0 {
				day.Status = d.worst
				day.Uptime = float64(d.ok) / float64(d.total) * 100
			}
			ch.Days = append(ch.Days, day)
		}
		out = append(out, ch)
	}
	return out
}

// StartStatusRecorder writes a component snapshot every five minutes so the
// status page can show a 90-day bar chart.
func (s *Server) StartStatusRecorder(ctx context.Context) {
	record := func() {
		cctx, cancel := context.WithTimeout(ctx, 10*time.Second)
		defer cancel()
		snap := s.collectStatus(cctx)
		_ = s.svc.Repo().PruneComponentStatus(cctx, time.Now().UTC().AddDate(0, 0, -180))
		for _, c := range snap.Components {
			_ = s.svc.Repo().RecordStatus(cctx, store.StatusSnapshot{
				Component: c.Key, Status: c.Status, Detail: c.Detail, CheckedAt: snap.CheckedAt,
			})
		}
	}
	record()
	go func() {
		ticker := time.NewTicker(5 * time.Minute)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				record()
			}
		}
	}()
}

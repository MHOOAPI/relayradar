package api

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"io"
	"net"
	"net/http"
	"regexp"
	"sort"
	"strings"
	"time"

	"relayradar/internal/app"
	"relayradar/internal/store"
)

var domainRe = regexp.MustCompile(`^[a-z0-9]([a-z0-9-]*[a-z0-9])?(\.[a-z0-9]([a-z0-9-]*[a-z0-9])?)+$`)

func randomToken(n int) string {
	b := make([]byte, n)
	if _, err := rand.Read(b); err != nil {
		return "000000"
	}
	return hex.EncodeToString(b)
}

func (s *Server) handleSubmission(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Name    string `json:"name"`
		Domain  string `json:"domain"`
		BaseURL string `json:"baseUrl"`
		Contact string `json:"contact"`
		Note    string `json:"note"`
		Payments []string `json:"payments"`
		Invoice  bool   `json:"invoice"`
		RefundPolicy string `json:"refundPolicy"`
	}
	if err := decodeJSON(r, &body); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	body.Name = strings.TrimSpace(body.Name)
	body.Domain = strings.ToLower(strings.TrimSpace(body.Domain))
	body.BaseURL = strings.TrimSpace(body.BaseURL)
	body.Contact = strings.TrimSpace(body.Contact)
	body.Note = strings.TrimSpace(body.Note)

	switch {
	case len([]rune(body.Name)) < 2 || len([]rune(body.Name)) > 60:
		writeError(w, http.StatusBadRequest, "站点名称需为 2–60 个字符")
		return
	case !domainRe.MatchString(body.Domain):
		writeError(w, http.StatusBadRequest, "域名格式不正确（例如 relay.example.com）")
		return
	case !strings.HasPrefix(body.BaseURL, "https://"):
		writeError(w, http.StatusBadRequest, "Base URL 必须以 https:// 开头")
		return
	case len([]rune(body.Contact)) < 3 || len([]rune(body.Contact)) > 120:
		writeError(w, http.StatusBadRequest, "请填写有效的联系方式（邮箱 / TG）")
		return
	case len([]rune(body.Note)) > 500:
		writeError(w, http.StatusBadRequest, "备注不能超过 500 字")
		return
	}

	sub, err := s.svc.Repo().CreateSubmission(r.Context(), store.Submission{
		Name: body.Name, Domain: body.Domain, BaseURL: body.BaseURL,
		Contact: body.Contact, Note: body.Note,
		Payments: body.Payments, Invoice: body.Invoice, RefundPolicy: strings.TrimSpace(body.RefundPolicy),
	})
	if err != nil {
		s.fail(w, http.StatusInternalServerError, err)
		return
	}
	writeJSON(w, http.StatusAccepted, sub)
}

func (s *Server) handleClaim(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Slug  string `json:"slug"`
		Email string `json:"email"`
	}
	if err := decodeJSON(r, &body); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	body.Slug = strings.TrimSpace(body.Slug)
	body.Email = strings.TrimSpace(body.Email)
	if !strings.Contains(body.Email, "@") || len(body.Email) > 120 {
		writeError(w, http.StatusBadRequest, "请填写有效邮箱")
		return
	}
	id, err := s.svc.Repo().StationIDBySlug(r.Context(), body.Slug)
	if err != nil {
		writeError(w, http.StatusNotFound, "站点不存在")
		return
	}
	token := "rr-" + body.Slug + "-" + randomToken(4)
	claim, err := s.svc.Repo().CreateClaim(r.Context(), id, body.Email, token)
	if err != nil {
		s.fail(w, http.StatusInternalServerError, err)
		return
	}
	domain, _ := s.svc.Repo().ClaimDomain(r.Context(), token)
	writeJSON(w, http.StatusCreated, map[string]any{
		"claim": claim,
		"dns": map[string]string{
			"type":  "TXT",
			"name":  "_relayradar." + domain,
			"value": token,
		},
		"file": "https://" + domain + "/.well-known/relayradar.txt",
	})
}

// verifyClaimOwnership checks the TXT record or the well-known file.
func verifyClaimOwnership(ctx context.Context, domain, token string, allowLocal bool) error {
	txts, err := net.DefaultResolver.LookupTXT(ctx, "_relayradar."+domain)
	if err == nil {
		for _, txt := range txts {
			if strings.TrimSpace(txt) == token {
				return nil
			}
		}
	}
	scheme := "https"
	if allowLocal && (strings.HasPrefix(domain, "127.0.0.1") || strings.HasPrefix(domain, "localhost")) {
		scheme = "http"
	}
	req, reqErr := http.NewRequestWithContext(ctx, http.MethodGet, scheme+"://"+domain+"/.well-known/relayradar.txt", nil)
	if reqErr == nil {
		client := &http.Client{Timeout: 5 * time.Second}
		if resp, err := client.Do(req); err == nil {
			body, _ := io.ReadAll(io.LimitReader(resp.Body, 512))
			resp.Body.Close()
			if resp.StatusCode == 200 && strings.TrimSpace(string(body)) == token {
				return nil
			}
		}
	}
	return fmt.Errorf("未找到归属验证：请在 DNS 添加 _relayradar.%s TXT = %s，或把令牌写入 https://%s/.well-known/relayradar.txt", domain, token, domain)
}

func (s *Server) handleClaimEditsMine(w http.ResponseWriter, r *http.Request) {
	limit := 20
	if v := r.URL.Query().Get("limit"); v != "" {
		if n, err := parseInt64(v); err == nil && n > 0 && n <= 100 {
			limit = int(n)
		}
	}
	items, err := s.svc.Repo().ClaimEditsByToken(r.Context(), r.PathValue("token"), limit)
	if err != nil {
		writeError(w, http.StatusNotFound, "认领令牌无效或尚未验证")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"items": items})
}

func (s *Server) handleClaimedStation(w http.ResponseWriter, r *http.Request) {
	cs, err := s.svc.Repo().ClaimedStation(r.Context(), r.PathValue("token"))
	if err != nil {
		writeError(w, http.StatusNotFound, "认领令牌无效或尚未验证")
		return
	}
	writeJSON(w, http.StatusOK, cs)
}

func (s *Server) handleUpdateClaimedStation(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Tagline      string  `json:"tagline"`
		RefundPolicy string  `json:"refundPolicy"`
		Invoice      bool    `json:"invoice"`
		MinTopup     float64 `json:"minTopupCny"`
	}
	if err := decodeJSON(r, &body); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	if len(body.Tagline) > 80 || len(body.RefundPolicy) > 120 {
		writeError(w, http.StatusBadRequest, "简介或退款说明过长")
		return
	}
	if body.MinTopup < 0 || body.MinTopup > 100000 {
		writeError(w, http.StatusBadRequest, "最低充值金额不合法")
		return
	}
	id, err := s.svc.Repo().CreateClaimEdit(r.Context(), r.PathValue("token"), body.Tagline, body.RefundPolicy, body.Invoice, body.MinTopup)
	if err != nil {
		writeError(w, http.StatusNotFound, "认领令牌无效或尚未验证")
		return
	}
	writeJSON(w, http.StatusAccepted, map[string]any{"ok": true, "pending": true, "id": id})
}

func (s *Server) handleAdminClaimEdits(w http.ResponseWriter, r *http.Request) {
	if s.adminOnly(w, r) == nil {
		return
	}
	items, err := s.svc.Repo().ListClaimEdits(r.Context(), r.URL.Query().Get("all") == "1")
	if err != nil {
		s.fail(w, http.StatusInternalServerError, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"items": items})
}

func (s *Server) handleAdminClaimEditDecision(w http.ResponseWriter, r *http.Request) {
	admin := s.adminOnly(w, r)
	if admin == nil {
		return
	}
	id, err := parseInt64(r.PathValue("id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid id")
		return
	}
	var body struct {
		Note string `json:"note"`
	}
	_ = decodeJSON(r, &body)
	approve := strings.HasSuffix(r.URL.Path, "/approve")
	if approve {
		err = s.svc.Repo().ApproveClaimEdit(r.Context(), id, strings.TrimSpace(body.Note))
	} else {
		err = s.svc.Repo().RejectClaimEdit(r.Context(), id, strings.TrimSpace(body.Note))
	}
	if err != nil {
		writeError(w, http.StatusNotFound, "edit not found")
		return
	}
	action := "claim.edit.reject"
	verdict := "未通过"
	if approve {
		action = "claim.edit.approve"
		verdict = "已通过"
	}
	target := fmt.Sprintf("claim edit #%d", id)
	if edits, err := s.svc.Repo().ListClaimEdits(r.Context(), true); err == nil {
		for _, e := range edits {
			if e.ID == id && e.StationSlug != "" {
				target = "station:" + e.StationSlug + " · " + target
				break
			}
		}
	}
	s.logAdminAction(r, admin, action, target, verdict)
	s.svc.Invalidate()
	if email, err := s.svc.Repo().ClaimEditEmail(r.Context(), id); err == nil && email != "" {
		subject := "RelayRadar 站点资料审核结果"
		text := "你在 RelayRadar 提交的站点资料修改" + verdict + "。\n\n" +
			"审核编号：#" + fmt.Sprintf("%d", id) + "\n" +
			func() string {
				if strings.TrimSpace(body.Note) != "" {
					return "审核意见：" + strings.TrimSpace(body.Note) + "\n"
				}
				return ""
			}() +
			"如果是通过，新的简介、退款、发票与最低充值已经生效；未通过可以修改后重新提交。\n\n" +
			strings.TrimRight(s.cfg.PublicBaseURL, "/") + "/submit?claim=1"
		if err := s.sendAuthMail(email, subject, text); err != nil {
			s.log.Warn("send claim review mail failed", "err", err)
		}
	}
	writeJSON(w, http.StatusOK, map[string]any{"ok": true})
}

func (s *Server) handleVerifyClaim(w http.ResponseWriter, r *http.Request) {
	if !s.verifyRL.allow(clientIP(r)) {
		writeError(w, http.StatusTooManyRequests, "too many verification attempts, retry later")
		return
	}
	token := r.PathValue("token")
	domain, err := s.svc.Repo().ClaimDomain(r.Context(), token)
	if err != nil {
		writeError(w, http.StatusNotFound, "校验令牌无效或已过期")
		return
	}
	if !s.cfg.ClaimDevBypass {
		if err := verifyClaimOwnership(r.Context(), domain, token, s.cfg.AllowLocalTargets); err != nil {
			writeError(w, http.StatusBadRequest, err.Error())
			return
		}
	}
	claim, slug, err := s.svc.Repo().VerifyClaim(r.Context(), token)
	if err != nil {
		writeError(w, http.StatusNotFound, "校验令牌无效或已过期")
		return
	}
	s.svc.Invalidate()
	writeJSON(w, http.StatusOK, map[string]any{
		"verified": true,
		"station":  slug,
		"claim":    claim,
	})
}

// handleLLMs serves a plain-text brief that AI agents can read directly.
func (s *Server) handleLLMs(w http.ResponseWriter, r *http.Request) {
	d, err := s.svc.Dataset(r.Context())
	if err != nil {
		s.fail(w, http.StatusInternalServerError, err)
		return
	}

	var b strings.Builder
	b.WriteString("# RelayRadar — 大模型 API 中转站独立监测\n\n")
	b.WriteString("> 用连续探针、速度基准、思考签名抽检与真实倍率排序的中转站榜单。\n")
	b.WriteString("> 数据为演示数据集；评分方法见 /methodology，接口见 /api-docs。\n\n")
	fmt.Fprintf(&b, "Data timestamp: %s\n", d.TakenAt.UTC().Format("2006-01-02T15:04:05Z"))
	fmt.Fprintf(&b, "Stations: %d · Models: %d\n\n", len(d.Stations), len(d.Models))

	b.WriteString("## 榜单（按 rank score 降序）\n\n")
	b.WriteString("| # | 站点 | 排名分 | 原始可用率 | 速度指数 | 纯度 | 最低输入价 | 起充 |\n")
	b.WriteString("|---|---|---|---|---|---|---|---|\n")
	for i, st := range d.Stations {
		if i >= 15 {
			break
		}
		price := "n/a"
		if st.FromPriceCNY >= 0 {
			price = fmt.Sprintf("¥%.2f/M", st.FromPriceCNY)
		}
		fmt.Fprintf(&b, "| %d | %s (`%s`) | %.1f | %.1f%% | %.0f | %s(%d) | %s | ¥%.0f |\n",
			st.Rank, st.Name, st.Slug, st.RankScore, st.UptimePct, st.Speed,
			st.PurityTier, int(st.Purity), price, st.MinTopup)
	}

	b.WriteString("\n## 模型与全网最低价\n\n")
	models := append([]appModelLite(nil), modelLites(d)...)
	sort.SliceStable(models, func(i, j int) bool { return models[i].ReleasedAt.After(models[j].ReleasedAt) })
	for _, m := range models {
		free := ""
		if m.Free {
			free = " · 有免费额度"
		}
		fmt.Fprintf(&b, "- %s (`%s`) — 官方 $%.2f/$%.2f per 1M，全网最低 ¥%.2f%s\n",
			m.Name, m.ID, m.InPrice, m.OutPrice, m.MinCNY, free)
	}

	b.WriteString("\n## 接口\n\n")
	b.WriteString("- `GET /api/v1/stations` — 榜单（参数 service/tier/q/sort）\n")
	b.WriteString("- `GET /api/v1/stations/{slug}` — 站点详情、分组、模型实价\n")
	b.WriteString("- `GET /api/v1/models` — 模型目录\n")
	b.WriteString("- `POST /api/v1/audit` — 探测任意 OpenAI 兼容端点（SSRF 守卫）\n")
	b.WriteString("- `GET /api/v1/audit/{id}` — 分享检测报告\n")
	b.WriteString("- `GET /llms-full.txt` — 全量站点与模型快照\n")
	b.WriteString("- `GET /md/station/{slug}` · `/md/model/{id}` — Markdown 版本\n")
	b.WriteString("- `GET /guides` — 选择、计费与验真指南\n\n")

	b.WriteString("## 指南\n\n")
	for _, g := range guides {
		fmt.Fprintf(&b, "- [%s](%s/guides/%s) — %s\n", g.Title, s.cfg.PublicBaseURL, g.Slug, g.Desc)
	}
	b.WriteString("\n")
	b.WriteString("## 说明\n\n")
	b.WriteString("- 纯度只对 Claude 分组做签名抽检，不参与排名。\n")
	b.WriteString("- 价格 = 官方 USD × 站点汇率 × 分组倍率，取全站最低输入价。\n")
	b.WriteString("- 本站不收款、不担保售后；引用数据请保留时间戳。\n")

	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte(b.String()))
}

type appModelLite struct {
	ID         string
	Name       string
	InPrice    float64
	OutPrice   float64
	MinCNY     float64
	Free       bool
	ReleasedAt time.Time
}

func modelLites(d *app.Dataset) []appModelLite {
	out := make([]appModelLite, 0, len(d.Models))
	for _, m := range d.Models {
		min := m.MinInputCNY
		if min < 0 {
			min = 0
		}
		out = append(out, appModelLite{
			ID: m.ID, Name: m.Name, InPrice: m.InputUSD, OutPrice: m.OutputUSD,
			MinCNY: min, Free: m.FreeTier, ReleasedAt: m.ReleasedAt,
		})
	}
	return out
}

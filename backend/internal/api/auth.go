package api

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"mime"
	"net"
	"net/http"
	"net/smtp"
	"regexp"
	"strconv"
	"strings"
	"time"

	"relayradar/internal/auth"
	"relayradar/internal/store"
)

func tokenHash(token string) string {
	sum := sha256.Sum256([]byte(token))
	return hex.EncodeToString(sum[:])
}

func (s *Server) sendAuthMail(to, subject, body string) error {
	cfg := s.cfg
	if !cfg.SMTPConfigured() {
		return fmt.Errorf("smtp is not configured")
	}
	msg := "From: RelayRadar <" + cfg.SMTPFrom + ">\r\n" +
		"To: " + to + "\r\n" +
		"Subject: " + mime.QEncoding.Encode("UTF-8", subject) + "\r\n" +
		"MIME-Version: 1.0\r\n" +
		"Content-Type: text/plain; charset=UTF-8\r\n\r\n" + body
	host, _, err := net.SplitHostPort(cfg.SMTPAddr)
	if err != nil {
		host = cfg.SMTPAddr
	}
	var smtpAuth smtp.Auth
	if cfg.SMTPUser != "" {
		smtpAuth = smtp.PlainAuth("", cfg.SMTPUser, cfg.SMTPPass, host)
	}
	return smtp.SendMail(cfg.SMTPAddr, smtpAuth, cfg.SMTPFrom, []string{to}, []byte(msg))
}

func (s *Server) issueToken(r *http.Request, userID int64, kind string, ttl time.Duration) (string, error) {
	token := randToken(32)
	err := s.svc.Repo().CreateAuthToken(r.Context(), tokenHash(token), userID, kind, time.Now().Add(ttl))
	return token, err
}

func (s *Server) handleVerifyEmail(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Token string `json:"token"`
	}
	if err := decodeJSON(r, &body); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	if body.Token == "" {
		writeError(w, http.StatusBadRequest, "token is required")
		return
	}
	userID, err := s.svc.Repo().ConsumeAuthToken(r.Context(), tokenHash(body.Token), "verify")
	if err != nil {
		writeError(w, http.StatusBadRequest, "verification link is invalid or expired")
		return
	}
	if err := s.svc.Repo().SetUserVerified(r.Context(), userID); err != nil {
		s.fail(w, http.StatusInternalServerError, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"ok": true})
}

func (s *Server) handleForgotPassword(w http.ResponseWriter, r *http.Request) {
	if !s.authRL.allow(clientIP(r)) {
		writeError(w, http.StatusTooManyRequests, "too many attempts, retry later")
		return
	}
	if !s.requireCaptcha(w, r) {
		return
	}
	var body struct {
		Email string `json:"email"`
	}
	if err := decodeJSON(r, &body); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	email := strings.ToLower(strings.TrimSpace(body.Email))
	// Always return the same response so the endpoint cannot enumerate accounts.
	if u, err := s.svc.Repo().UserByEmail(r.Context(), email); err == nil {
		if token, err := s.issueToken(r, u.ID, "reset", time.Hour); err == nil {
			link := strings.TrimRight(s.cfg.PublicBaseURL, "/") + "/reset?token=" + token
			body := "有人请求重置 RelayRadar 账号密码。\n\n" + link + "\n\n链接 1 小时内有效；如果不是你本人操作，请忽略这封邮件。"
			if err := s.sendAuthMail(u.Email, "重置 RelayRadar 密码", body); err != nil {
				s.log.Warn("send reset mail failed", "err", err)
			}
		}
	}
	writeJSON(w, http.StatusOK, map[string]any{"ok": true})
}

func (s *Server) handleResetPassword(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Token    string `json:"token"`
		Password string `json:"password"`
	}
	if err := decodeJSON(r, &body); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	if len(body.Password) < 8 || len(body.Password) > 128 {
		writeError(w, http.StatusBadRequest, "password must be 8-128 characters")
		return
	}
	userID, err := s.svc.Repo().ConsumeAuthToken(r.Context(), tokenHash(body.Token), "reset")
	if err != nil {
		writeError(w, http.StatusBadRequest, "reset link is invalid or expired")
		return
	}
	if err := s.svc.Repo().UpdatePassword(r.Context(), userID, auth.HashPassword(body.Password)); err != nil {
		s.fail(w, http.StatusInternalServerError, err)
		return
	}
	_ = s.svc.Repo().DeleteUserSessions(r.Context(), userID)
	writeJSON(w, http.StatusOK, map[string]any{"ok": true})
}

const sessionCookie = "rr_session"

var emailRe = regexp.MustCompile(`^[^@\s]+@[^@\s]+\.[^@\s]+$`)

func (s *Server) setSession(w http.ResponseWriter, token string) {
	http.SetCookie(w, &http.Cookie{
		Name: sessionCookie, Value: token, Path: "/",
		HttpOnly: true, SameSite: http.SameSiteLaxMode,
		Secure: strings.HasPrefix(s.cfg.PublicBaseURL, "https://"),
		MaxAge: 30 * 24 * 3600,
	})
}

func (s *Server) currentUser(r *http.Request) *store.User {
	c, err := r.Cookie(sessionCookie)
	if err != nil || c.Value == "" {
		return nil
	}
	sum := sha256.Sum256([]byte(c.Value))
	u, err := s.svc.Repo().UserBySession(r.Context(), hex.EncodeToString(sum[:]))
	if err != nil {
		return nil
	}
	return &u
}

func (s *Server) handleRegister(w http.ResponseWriter, r *http.Request) {
	if !s.authRL.allow(clientIP(r)) {
		writeError(w, http.StatusTooManyRequests, "too many attempts, retry later")
		return
	}
	if !s.requireCaptcha(w, r) {
		return
	}
	var body struct {
		Email    string `json:"email"`
		Password string `json:"password"`
		Name     string `json:"name"`
	}
	if err := decodeJSON(r, &body); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	email := strings.ToLower(strings.TrimSpace(body.Email))
	name := strings.TrimSpace(body.Name)
	if !emailRe.MatchString(email) || len(email) > 200 {
		writeError(w, http.StatusBadRequest, "invalid email")
		return
	}
	if len(body.Password) < 8 || len(body.Password) > 128 {
		writeError(w, http.StatusBadRequest, "password must be 8-128 characters")
		return
	}
	if len(name) > 60 {
		writeError(w, http.StatusBadRequest, "name is too long")
		return
	}
	if name == "" {
		name = strings.Split(email, "@")[0]
	}
	role := "user"
	for _, admin := range s.cfg.AdminEmails {
		if admin == email {
			role = "admin"
		}
	}
	u, err := s.svc.Repo().CreateUser(r.Context(), email, name, auth.HashPassword(body.Password), role)
	if err != nil {
		writeError(w, http.StatusConflict, "email already registered")
		return
	}
	token := randToken(32)
	sum := sha256.Sum256([]byte(token))
	if err := s.svc.Repo().CreateSession(r.Context(), hex.EncodeToString(sum[:]), u.ID, time.Now().Add(30*24*time.Hour)); err != nil {
		s.fail(w, http.StatusInternalServerError, err)
		return
	}
	s.setSession(w, token)
	if vt, err := s.issueToken(r, u.ID, "verify", 24*time.Hour); err == nil {
		link := strings.TrimRight(s.cfg.PublicBaseURL, "/") + "/verify?token=" + vt
		body := "欢迎使用 RelayRadar。\n\n请打开下面的链接验证邮箱（24 小时内有效）：\n" + link + "\n\n如果不是你注册的账号，请忽略这封邮件。"
		if err := s.sendAuthMail(u.Email, "验证 RelayRadar 邮箱", body); err != nil {
			s.log.Warn("send verification mail failed", "err", err)
		}
	}
	writeJSON(w, http.StatusCreated, map[string]any{"user": u})
}

func (s *Server) handleLogin(w http.ResponseWriter, r *http.Request) {
	if !s.authRL.allow(clientIP(r)) {
		writeError(w, http.StatusTooManyRequests, "too many attempts, retry later")
		return
	}
	var body struct {
		Email    string `json:"email"`
		Password string `json:"password"`
	}
	if err := decodeJSON(r, &body); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	email := strings.ToLower(strings.TrimSpace(body.Email))
	u, err := s.svc.Repo().UserByEmail(r.Context(), email)
	if err != nil || !auth.VerifyPassword(body.Password, u.PasswordHash) {
		writeError(w, http.StatusUnauthorized, "invalid email or password")
		return
	}
	token := randToken(32)
	sum := sha256.Sum256([]byte(token))
	if err := s.svc.Repo().CreateSession(r.Context(), hex.EncodeToString(sum[:]), u.ID, time.Now().Add(30*24*time.Hour)); err != nil {
		s.fail(w, http.StatusInternalServerError, err)
		return
	}
	s.setSession(w, token)
	writeJSON(w, http.StatusOK, map[string]any{"user": u})
}

func (s *Server) handleLogout(w http.ResponseWriter, r *http.Request) {
	if c, err := r.Cookie(sessionCookie); err == nil && c.Value != "" {
		sum := sha256.Sum256([]byte(c.Value))
		_ = s.svc.Repo().DeleteSession(r.Context(), hex.EncodeToString(sum[:]))
	}
	http.SetCookie(w, &http.Cookie{Name: sessionCookie, Value: "", Path: "/", HttpOnly: true, MaxAge: -1})
	writeJSON(w, http.StatusOK, map[string]any{"ok": true})
}

func (s *Server) handleMe(w http.ResponseWriter, r *http.Request) {
	u := s.currentUser(r)
	writeJSON(w, http.StatusOK, map[string]any{"user": u})
}

func (s *Server) adminOnly(w http.ResponseWriter, r *http.Request) *store.User {
	u := s.currentUser(r)
	if u == nil {
		writeError(w, http.StatusUnauthorized, "sign in required")
		return nil
	}
	if u.Role != "admin" {
		writeError(w, http.StatusForbidden, "admin role required")
		return nil
	}
	return u
}

func (s *Server) handleAdminSummary(w http.ResponseWriter, r *http.Request) {
	if s.adminOnly(w, r) == nil {
		return
	}
	counts, updated, err := s.svc.Repo().Counts(r.Context())
	if err != nil {
		s.fail(w, http.StatusInternalServerError, err)
		return
	}
	subs, _ := s.svc.Repo().ListSubscriptions(r.Context())
	submissions, _ := s.svc.Repo().ListSubmissions(r.Context())
	pending := 0
	for _, sub := range submissions {
		if sub.Status == "pending" || sub.Status == "" {
			pending++
		}
	}
	failing := 0
	for _, sub := range subs {
		if sub.Attempts >= 3 {
			failing++
		}
	}
	claimed := 0
	staleProbes := 0
	if d, derr := s.svc.Dataset(r.Context()); derr == nil {
		for _, st := range d.Stations {
			if st.Claimed {
				claimed++
			}
			if st.LastProbeAt.IsZero() || time.Since(st.LastProbeAt) > 6*time.Hour {
				staleProbes++
			}
		}
	}
	pendingEdits := 0
	if edits, eerr := s.svc.Repo().ListClaimEdits(r.Context(), false); eerr == nil {
		pendingEdits = len(edits)
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"counts": counts, "updatedAt": updated,
		"subscriptions": len(subs), "submissions": len(submissions), "pendingSubmissions": pending,
		"failingSubscriptions": failing, "claimedStations": claimed, "pendingClaimEdits": pendingEdits,
		"staleProbes": staleProbes,
	})
}

func (s *Server) handleAdminSubmissions(w http.ResponseWriter, r *http.Request) {
	if s.adminOnly(w, r) == nil {
		return
	}
	items, err := s.svc.Repo().ListSubmissions(r.Context())
	if err != nil {
		s.fail(w, http.StatusInternalServerError, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"items": items})
}

func (s *Server) handleAdminSubmissionStatus(w http.ResponseWriter, r *http.Request) {
	admin := s.adminOnly(w, r)
	if admin == nil {
		return
	}
	var body struct {
		Status       string   `json:"status"`
		Payments     []string `json:"payments"`
		Invoice      bool     `json:"invoice"`
		RefundPolicy string   `json:"refundPolicy"`
	}
	if err := decodeJSON(r, &body); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	switch body.Status {
	case "pending", "approved", "rejected":
	default:
		writeError(w, http.StatusBadRequest, "status must be pending, approved or rejected")
		return
	}
	id, err := parseInt64(r.PathValue("id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid id")
		return
	}
	if err := s.svc.Repo().UpdateSubmissionTerms(r.Context(), id, body.Payments, body.Invoice, body.RefundPolicy); err != nil {
		writeError(w, http.StatusNotFound, "submission not found")
		return
	}
	s.logAdminAction(r, admin, "submission.terms", fmt.Sprintf("submission #%d", id), strings.Join(body.Payments, ","))
	if err := s.svc.Repo().UpdateSubmissionStatus(r.Context(), id, body.Status); err != nil {
		writeError(w, http.StatusNotFound, "submission not found")
		return
	}
	if body.Status == "approved" {
		_ = s.svc.Repo().ApplyTermsToStation(r.Context(), id)
	}
	s.logAdminAction(r, admin, "submission.status", fmt.Sprintf("submission #%d", id), body.Status)
	writeJSON(w, http.StatusOK, map[string]any{"ok": true})
}

func (s *Server) handleAdminDeleteReview(w http.ResponseWriter, r *http.Request) {
	admin := s.adminOnly(w, r)
	if admin == nil {
		return
	}
	id, err := parseInt64(r.PathValue("id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid id")
		return
	}
	if err := s.svc.Repo().DeleteReview(r.Context(), id); err != nil {
		writeError(w, http.StatusNotFound, "review not found")
		return
	}
	s.logAdminAction(r, admin, "review.delete", fmt.Sprintf("review #%d", id), "deleted")
	writeJSON(w, http.StatusOK, map[string]any{"ok": true})
}

func (s *Server) handleAdminSubscriptions(w http.ResponseWriter, r *http.Request) {
	if s.adminOnly(w, r) == nil {
		return
	}
	subs, err := s.svc.Repo().ListSubscriptions(r.Context())
	if err != nil {
		s.fail(w, http.StatusInternalServerError, err)
		return
	}
	items := make([]map[string]any, 0, len(subs))
	for _, sub := range subs {
		items = append(items, map[string]any{
			"id": sub.ID, "channel": sub.Channel, "url": sub.URL, "email": sub.Email,
			"kinds": sub.Kinds, "enabled": sub.Enabled, "attempts": sub.Attempts,
			"lastError": sub.LastError, "createdAt": sub.CreatedAt, "lastSentAt": sub.LastSentAt,
		})
	}
	writeJSON(w, http.StatusOK, map[string]any{"items": items})
}

func (s *Server) handleAdminUsers(w http.ResponseWriter, r *http.Request) {
	if s.adminOnly(w, r) == nil {
		return
	}
	users, err := s.svc.Repo().ListUsers(r.Context())
	if err != nil {
		s.fail(w, http.StatusInternalServerError, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"items": users})
}

func (s *Server) handleAdminUserRole(w http.ResponseWriter, r *http.Request) {
	admin := s.adminOnly(w, r)
	if admin == nil {
		return
	}
	var body struct {
		Role string `json:"role"`
	}
	if err := decodeJSON(r, &body); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	if body.Role != "admin" && body.Role != "user" {
		writeError(w, http.StatusBadRequest, "role must be admin or user")
		return
	}
	id, err := parseInt64(r.PathValue("id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid id")
		return
	}
	if id == admin.ID && body.Role != "admin" {
		writeError(w, http.StatusBadRequest, "cannot demote yourself")
		return
	}
	if err := s.svc.Repo().UpdateUserRole(r.Context(), id, body.Role); err != nil {
		writeError(w, http.StatusNotFound, "user not found")
		return
	}
	s.logAdminAction(r, admin, "user.role", fmt.Sprintf("user #%d", id), body.Role)
	writeJSON(w, http.StatusOK, map[string]any{"ok": true})
}

func (s *Server) handleAdminAudit(w http.ResponseWriter, r *http.Request) {
	if s.adminOnly(w, r) == nil {
		return
	}
	limit := 100
	if v := r.URL.Query().Get("limit"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 && n <= 500 {
			limit = n
		}
	}
	items, err := s.svc.Repo().ListAdminActions(r.Context(), limit)
	if err != nil {
		s.fail(w, http.StatusInternalServerError, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"items": items})
}

func (s *Server) logAdminAction(r *http.Request, admin *store.User, action, target, detail string) {
	if admin == nil {
		return
	}
	if err := s.svc.Repo().RecordAdminAction(r.Context(), store.AdminAction{
		ActorID: admin.ID, ActorEmail: admin.Email, Action: action,
		Target: target, Detail: detail, IP: clientIP(r),
	}); err != nil {
		s.log.Warn("record admin action failed", "action", action, "err", err)
	}
}

func (s *Server) handleAdminReviews(w http.ResponseWriter, r *http.Request) {
	if s.adminOnly(w, r) == nil {
		return
	}
	reviews, err := s.svc.Repo().ListAllReviews(r.Context())
	if err != nil {
		s.fail(w, http.StatusInternalServerError, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"items": reviews})
}

func parseInt64(s string) (int64, error) {
	return strconv.ParseInt(s, 10, 64)
}

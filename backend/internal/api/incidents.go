package api

import (
	"encoding/xml"
	"fmt"
	"net/http"
	"strings"
	"time"

	"relayradar/internal/store"
)

var incidentStatuses = map[string]bool{"investigating": true, "identified": true, "monitoring": true, "resolved": true}

func (s *Server) handleIncidents(w http.ResponseWriter, r *http.Request) {
	all := r.URL.Query().Get("all") == "1"
	items, err := s.svc.Repo().ListIncidents(r.Context(), all)
	if err != nil {
		s.fail(w, http.StatusInternalServerError, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"items": items})
}

func (s *Server) handleAdminIncidents(w http.ResponseWriter, r *http.Request) {
	if s.adminOnly(w, r) == nil {
		return
	}
	items, err := s.svc.Repo().ListIncidents(r.Context(), true)
	if err != nil {
		s.fail(w, http.StatusInternalServerError, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"items": items})
}

func (s *Server) handleAdminIncidentCreate(w http.ResponseWriter, r *http.Request) {
	admin := s.adminOnly(w, r)
	if admin == nil {
		return
	}
	var body struct {
		Title     string `json:"title"`
		Status    string `json:"status"`
		Component string `json:"component"`
		Body      string `json:"body"`
	}
	if err := decodeJSON(r, &body); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	if strings.TrimSpace(body.Title) == "" {
		writeError(w, http.StatusBadRequest, "title is required")
		return
	}
	if body.Status == "" {
		body.Status = "investigating"
	}
	if !incidentStatuses[body.Status] {
		writeError(w, http.StatusBadRequest, "invalid status")
		return
	}
	if body.Component == "" {
		body.Component = "api"
	}
	id, err := s.svc.Repo().CreateIncident(r.Context(), store.Incident{
		Title: strings.TrimSpace(body.Title), Status: body.Status,
		Component: body.Component, Body: strings.TrimSpace(body.Body), StartedAt: time.Now().UTC(),
	})
	if err != nil {
		s.fail(w, http.StatusInternalServerError, err)
		return
	}
	s.logAdminAction(r, admin, "incident.create", fmt.Sprintf("incident #%d", id), body.Title)
	writeJSON(w, http.StatusCreated, map[string]any{"id": id})
}

func (s *Server) handleAdminIncidentUpdate(w http.ResponseWriter, r *http.Request) {
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
		Status string `json:"status"`
		Body   string `json:"body"`
	}
	if err := decodeJSON(r, &body); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	if !incidentStatuses[body.Status] {
		writeError(w, http.StatusBadRequest, "invalid status")
		return
	}
	var resolvedAt *time.Time
	if body.Status == "resolved" {
		now := time.Now().UTC()
		resolvedAt = &now
	}
	if err := s.svc.Repo().UpdateIncident(r.Context(), id, body.Status, strings.TrimSpace(body.Body), resolvedAt); err != nil {
		writeError(w, http.StatusNotFound, "incident not found")
		return
	}
	s.logAdminAction(r, admin, "incident.update", fmt.Sprintf("incident #%d", id), body.Status)
	writeJSON(w, http.StatusOK, map[string]any{"ok": true})
}

func (s *Server) handleAdminIncidentDelete(w http.ResponseWriter, r *http.Request) {
	admin := s.adminOnly(w, r)
	if admin == nil {
		return
	}
	id, err := parseInt64(r.PathValue("id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid id")
		return
	}
	if err := s.svc.Repo().DeleteIncident(r.Context(), id); err != nil {
		writeError(w, http.StatusNotFound, "incident not found")
		return
	}
	s.logAdminAction(r, admin, "incident.delete", fmt.Sprintf("incident #%d", id), "deleted")
	writeJSON(w, http.StatusOK, map[string]any{"ok": true})
}

// handleStatusRSS exposes incidents as a small RSS feed for status subscribers.
func (s *Server) handleStatusRSS(w http.ResponseWriter, r *http.Request) {
	items, _ := s.svc.Repo().ListIncidents(r.Context(), true)
	type rssItem struct {
		Title   string `xml:"title"`
		Link    string `xml:"link"`
		GUID    string `xml:"guid"`
		PubDate string `xml:"pubDate"`
		Desc    string `xml:"description"`
	}
	type rss struct {
		XMLName xml.Name  `xml:"rss"`
		Version string    `xml:"version,attr"`
		Channel struct {
			Title string    `xml:"title"`
			Link  string    `xml:"link"`
			Desc  string    `xml:"description"`
			Items []rssItem `xml:"item"`
		} `xml:"channel"`
	}
	base := strings.TrimRight(s.cfg.PublicBaseURL, "/")
	feed := rss{Version: "2.0"}
	feed.Channel.Title = "RelayRadar 服务状态"
	feed.Channel.Link = base + "/status"
	feed.Channel.Desc = "服务事故与维护记录"
	for _, i := range items {
		feed.Channel.Items = append(feed.Channel.Items, rssItem{
			Title:   i.Title, Link: base + "/status", GUID: fmt.Sprintf("%s/status#incident-%d", base, i.ID),
			PubDate: i.StartedAt.Format(time.RFC1123Z), Desc: i.Body,
		})
	}
	w.Header().Set("Content-Type", "application/rss+xml; charset=utf-8")
	_, _ = w.Write([]byte(xml.Header))
	_ = xml.NewEncoder(w).Encode(feed)
}

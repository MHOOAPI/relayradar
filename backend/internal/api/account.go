package api

import (
	"encoding/json"
	"net/http"
	"strings"

	"relayradar/internal/store"
)

// Account center: everything a signed-in user owns, keyed by their e-mail.
// Subscriptions, submissions and claims are matched by e-mail because that is
// the only identity those tables carry; ownership is checked here, never on the
// client.

func (s *Server) handleAccount(w http.ResponseWriter, r *http.Request) {
	u := s.currentUser(r)
	if u == nil {
		writeError(w, http.StatusUnauthorized, "not signed in")
		return
	}
	ctx := r.Context()

	subs := make([]store.Subscription, 0)
	allSubs, err := s.svc.Repo().ListSubscriptions(ctx)
	if err == nil {
		for _, sub := range allSubs {
			if strings.EqualFold(sub.Email, u.Email) {
				subs = append(subs, sub)
			}
		}
	}
	active := 0
	for _, sub := range subs {
		if sub.Enabled {
			active++
		}
	}

	mySubs := make([]store.Submission, 0)
	pending := 0
	if all, err := s.svc.Repo().ListSubmissions(ctx); err == nil {
		for _, sub := range all {
			if strings.EqualFold(sub.Contact, u.Email) {
				mySubs = append(mySubs, sub)
				if sub.Status == "pending" {
					pending++
				}
			}
		}
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"user": u,
		"stats": map[string]int{
			"subscriptions": len(subs),
			"active":        active,
			"submissions":   len(mySubs),
			"pending":       pending,
		},
		"subscriptions": subs,
		"submissions":   mySubs,
	})
}

// handleAccountSubscription toggles or deletes one of the signed-in user's
// subscriptions after checking that the subscription belongs to their e-mail.
func (s *Server) handleAccountSubscription(w http.ResponseWriter, r *http.Request) {
	u := s.currentUser(r)
	if u == nil {
		writeError(w, http.StatusUnauthorized, "not signed in")
		return
	}
	ctx := r.Context()
	id := r.PathValue("id")
	sub, err := s.svc.Repo().GetSubscription(ctx, id)
	if err != nil || !strings.EqualFold(sub.Email, u.Email) {
		writeError(w, http.StatusNotFound, "subscription not found")
		return
	}

	switch r.Method {
	case http.MethodPatch:
		var body struct {
			Enabled *bool `json:"enabled"`
		}
		if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<12)).Decode(&body); err != nil {
			writeError(w, http.StatusBadRequest, "invalid body")
			return
		}
		if body.Enabled != nil {
			sub.Enabled = *body.Enabled
		}
		if err := s.svc.Repo().UpdateSubscription(ctx, sub); err != nil {
			s.fail(w, http.StatusInternalServerError, err)
			return
		}
	case http.MethodDelete:
		if err := s.svc.Repo().DeleteSubscription(ctx, sub.ID); err != nil {
			s.fail(w, http.StatusInternalServerError, err)
			return
		}
	default:
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"ok": true})
}

package store

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"strings"
	"sync"
	"time"
)

// Repo is the read/write surface the API needs. Both the PostgreSQL store and
// the in-memory demo store implement it.
var errNotFound = errors.New("not found")

type Repo interface {
	Snapshot(ctx context.Context) (*Snapshot, error)
	StationIDBySlug(ctx context.Context, slug string) (int64, error)
	ListReviews(ctx context.Context, stationID int64) ([]Review, error)
	InsertReview(ctx context.Context, stationID int64, author string, rating int, body string) (Review, error)
	InsertAudit(ctx context.Context, a AuditReport) error
	GetAudit(ctx context.Context, id string) (AuditReport, error)
	CreateSubmission(ctx context.Context, sub Submission) (Submission, error)
	CreateClaim(ctx context.Context, stationID int64, email, token string) (Claim, error)
	VerifyClaim(ctx context.Context, token string) (Claim, string, error)
	ClaimedStation(ctx context.Context, token string) (ClaimedStation, error)
	UpdateClaimedStation(ctx context.Context, token, tagline, refundPolicy string, invoice bool, minTopup float64) error
	ClaimDomain(ctx context.Context, token string) (string, error)
	CreateClaimEdit(ctx context.Context, token, tagline, refundPolicy string, invoice bool, minTopup float64) (int64, error)
	ListClaimEdits(ctx context.Context, includeDecided bool) ([]ClaimEdit, error)
	ApproveClaimEdit(ctx context.Context, id int64, note string) error
	RejectClaimEdit(ctx context.Context, id int64, note string) error
	ClaimEditEmail(ctx context.Context, id int64) (string, error)
	ClaimEditsByToken(ctx context.Context, token string, limit int) ([]ClaimEdit, error)
	CreateSubscription(ctx context.Context, sub Subscription) error
	ListSubscriptions(ctx context.Context) ([]Subscription, error)
	GetSubscription(ctx context.Context, id string) (Subscription, error)
	UpdateSubscription(ctx context.Context, sub Subscription) error
	SetDeliveryState(ctx context.Context, id string, attempts int, lastErr string, next *time.Time, lastSent *time.Time) error
	TouchSubscription(ctx context.Context, id string, at time.Time) error
	DeleteSubscription(ctx context.Context, id string) error
	RecordOutbound(ctx context.Context, slug string) error
	OutboundCount(ctx context.Context, slug string) (int, error)
	CreateUser(ctx context.Context, email, name, passwordHash, role string) (User, error)
	UserByEmail(ctx context.Context, email string) (User, error)
	CreateSession(ctx context.Context, tokenHash string, userID int64, expires time.Time) error
	UserBySession(ctx context.Context, tokenHash string) (User, error)
	DeleteSession(ctx context.Context, tokenHash string) error
	ListSubmissions(ctx context.Context) ([]Submission, error)
	UpdateSubmissionStatus(ctx context.Context, id int64, status string) error
	UpdateStationIcon(ctx context.Context, slug, iconURL string) error
	UpdateSubmissionTerms(ctx context.Context, id int64, payments []string, invoice bool, refundPolicy string) error
	ApplyTermsToStation(ctx context.Context, id int64) error
	DeleteReview(ctx context.Context, id int64) error
	ListUsers(ctx context.Context) ([]User, error)
	ListAllReviews(ctx context.Context) ([]AdminReview, error)
	SnapshotPriceDaily(ctx context.Context, day time.Time) (int, error)
	PriceDailyHistory(ctx context.Context, modelID string, days int) ([]PriceDailyPoint, error)
	StationPriceHistory(ctx context.Context, slug, modelID string, days int) ([]PriceDailyPoint, error)
	UpdateUserRole(ctx context.Context, id int64, role string) error
	SetUserVerified(ctx context.Context, id int64) error
	CreateAuthToken(ctx context.Context, tokenHash string, userID int64, kind string, expires time.Time) error
	ConsumeAuthToken(ctx context.Context, tokenHash, kind string) (int64, error)
	UpdatePassword(ctx context.Context, id int64, passwordHash string) error
	DeleteUserSessions(ctx context.Context, userID int64) error
	RecordDelivery(ctx context.Context, d Delivery) (int64, error)
	ListDeliveries(ctx context.Context, subscriptionID string, limit int) ([]Delivery, error)
	GetDelivery(ctx context.Context, id int64) (Delivery, error)
	RecordAdminAction(ctx context.Context, a AdminAction) error
	ListAdminActions(ctx context.Context, limit int) ([]AdminAction, error)
	CreateFreeKey(ctx context.Context, k FreeKey) (int64, error)
	ListFreeKeys(ctx context.Context, includeInvalid bool, q string) ([]FreeKey, error)
	GetFreeKey(ctx context.Context, id int64) (FreeKey, error)
	IncrementFreeKeyClaims(ctx context.Context, id int64) error
	DeleteFreeKey(ctx context.Context, id int64) error
	RecordStatus(ctx context.Context, s StatusSnapshot) error
	StatusHistory(ctx context.Context, since time.Time) ([]StatusSnapshot, error)
	CreateIncident(ctx context.Context, i Incident) (int64, error)
	ListIncidents(ctx context.Context, includeResolved bool) ([]Incident, error)
	UpdateIncident(ctx context.Context, id int64, status, body string, resolvedAt *time.Time) error
	DeleteIncident(ctx context.Context, id int64) error
	RecordProviderStatus(ctx context.Context, p ProviderSnapshot) error
	ProviderStatusHistory(ctx context.Context, since time.Time) ([]ProviderSnapshot, error)
	RecordReachability(ctx context.Context, rows []Reachability) error
	ListReachability(ctx context.Context) (map[string]Reachability, error)
	ReachabilityHistory(ctx context.Context, slug string, since time.Time) ([]Reachability, error)
	GroupByStationService(ctx context.Context, slug, service string) (int64, error)
	InsertProbe(ctx context.Context, groupID int64, status string, latencyMS int, at time.Time) error
	PruneProviderStatus(ctx context.Context, before time.Time) error
	PruneComponentStatus(ctx context.Context, before time.Time) error
	Counts(ctx context.Context) (map[string]int, time.Time, error)
}

var _ Repo = (*Store)(nil)

// Memory is a volatile store used for local previews when PostgreSQL is not
// available. It holds the same deterministic demo dataset.
type Memory struct {
	mu            sync.RWMutex
	snap          *Snapshot
	reviews       []Review
	audits        map[string]AuditReport
	submissions   []Submission
	claims        map[string]Claim
	subscriptions map[string]Subscription
	nextReviewID  int64
	nextSubID     int64
	nextClaimID   int64
	outbound      []outboundClick
	priceDaily    []PriceDaily
	users         map[int64]User
	userByEmail   map[string]int64
	sessions      map[string]memSession
	authTokens    map[string]memToken
	deliveries    []Delivery
	nextDelivery  int64
	adminActions  []AdminAction
	nextAdminAct  int64
	freeKeys      map[int64]FreeKey
	nextFreeKey   int64
	statuses      []StatusSnapshot
	incidents     map[int64]Incident
	nextIncident  int64
	providers     []ProviderSnapshot
	reachability  map[string]Reachability
	reachHistory  []Reachability
	claimEdits    []ClaimEdit
	nextClaimEdit int64
	nextUserID    int64
}

type memToken struct {
	UserID    int64
	Kind      string
	ExpiresAt time.Time
}

type memSession struct {
	UserID    int64
	ExpiresAt time.Time
}

type outboundClick struct {
	Slug string
	At   time.Time
}

func NewMemory() *Memory {
	snap, reviews := BuildSeedSnapshot()
	sort.SliceStable(reviews, func(i, j int) bool { return reviews[i].CreatedAt.After(reviews[j].CreatedAt) })
	return &Memory{
		snap:          snap,
		reviews:       reviews,
		audits:        map[string]AuditReport{},
		claims:        map[string]Claim{},
		subscriptions: map[string]Subscription{},
		users:         map[int64]User{},
		priceDaily:    []PriceDaily{},
		userByEmail:   map[string]int64{},
		sessions:      map[string]memSession{},
		authTokens:    map[string]memToken{},
		nextDelivery:  1,
		nextAdminAct:  1,
		freeKeys:      map[int64]FreeKey{},
		nextFreeKey:   1,
		incidents:     map[int64]Incident{},
		nextIncident:  1,
		nextUserID:    1,
		nextReviewID:  int64(len(reviews) + 1),
		nextSubID:     1,
		nextClaimID:   1,
	}
}

func (m *Memory) Snapshot(ctx context.Context) (*Snapshot, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	m.snap.TakenAt = time.Now().UTC()
	return m.snap, nil
}

func (m *Memory) StationIDBySlug(ctx context.Context, slug string) (int64, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	for _, st := range m.snap.Stations {
		if st.Slug == slug {
			return st.ID, nil
		}
	}
	return 0, errors.New("station not found")
}

func (m *Memory) ListReviews(ctx context.Context, stationID int64) ([]Review, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	out := make([]Review, 0)
	for _, r := range m.reviews {
		if r.StationID == stationID {
			out = append(out, r)
		}
	}
	return out, nil
}

func (m *Memory) InsertReview(ctx context.Context, stationID int64, author string, rating int, body string) (Review, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	rv := Review{
		ID: m.nextReviewID, StationID: stationID, Author: author,
		Rating: rating, Body: body, Verified: false, CreatedAt: time.Now().UTC(),
	}
	m.nextReviewID++
	m.reviews = append([]Review{rv}, m.reviews...)
	return rv, nil
}

func (m *Memory) InsertAudit(ctx context.Context, a AuditReport) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.audits[a.ID] = a
	return nil
}

func (m *Memory) GetAudit(ctx context.Context, id string) (AuditReport, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	a, ok := m.audits[id]
	if !ok {
		return AuditReport{}, errors.New("report not found")
	}
	return a, nil
}

func (m *Memory) CreateSubmission(ctx context.Context, sub Submission) (Submission, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	sub.ID = m.nextSubID
	m.nextSubID++
	if sub.Status == "" {
		sub.Status = "pending"
	}
	sub.CreatedAt = time.Now().UTC()
	m.submissions = append(m.submissions, sub)
	return sub, nil
}

func (m *Memory) CreateClaim(ctx context.Context, stationID int64, email, token string) (Claim, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	c := Claim{
		ID: m.nextClaimID, StationID: stationID, Email: email, Token: token,
		CreatedAt: time.Now().UTC(),
	}
	m.nextClaimID++
	m.claims[token] = c
	return c, nil
}

func (m *Memory) VerifyClaim(ctx context.Context, token string) (Claim, string, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	c, ok := m.claims[token]
	if !ok {
		return Claim{}, "", errors.New("claim token not found")
	}
	c.Verified = true
	m.claims[token] = c
	for i := range m.snap.Stations {
		if m.snap.Stations[i].ID == c.StationID {
			m.snap.Stations[i].Claimed = true
			return c, m.snap.Stations[i].Slug, nil
		}
	}
	return Claim{}, "", errors.New("station not found")
}

func (m *Memory) ClaimedStation(ctx context.Context, token string) (ClaimedStation, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	c, ok := m.claims[token]
	if !ok || !c.Verified {
		return ClaimedStation{}, errors.New("claim token not verified")
	}
	for _, st := range m.snap.Stations {
		if st.ID == c.StationID {
			cs := ClaimedStation{Slug: st.Slug, Name: st.Name, Tagline: st.Tagline, RefundPolicy: st.RefundPolicy, Invoice: st.Invoice, MinTopup: st.MinTopupCNY}
			for i := len(m.claimEdits) - 1; i >= 0; i-- {
				if m.claimEdits[i].StationSlug == st.Slug {
					cs.ReviewStatus = m.claimEdits[i].Status
					cs.ReviewNote = m.claimEdits[i].ReviewNote
					break
				}
			}
			return cs, nil
		}
	}
	return ClaimedStation{}, errors.New("station not found")
}

func (m *Memory) UpdateClaimedStation(ctx context.Context, token, tagline, refundPolicy string, invoice bool, minTopup float64) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	c, ok := m.claims[token]
	if !ok || !c.Verified {
		return errors.New("claim token not verified")
	}
	for i := range m.snap.Stations {
		if m.snap.Stations[i].ID == c.StationID {
			m.snap.Stations[i].Tagline = tagline
			m.snap.Stations[i].RefundPolicy = refundPolicy
			m.snap.Stations[i].Invoice = invoice
			m.snap.Stations[i].MinTopupCNY = minTopup
			return nil
		}
	}
	return errors.New("station not found")
}

func (m *Memory) ClaimDomain(ctx context.Context, token string) (string, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	c, ok := m.claims[token]
	if !ok {
		return "", errors.New("claim token not found")
	}
	for _, st := range m.snap.Stations {
		if st.ID == c.StationID {
			return st.Domain, nil
		}
	}
	return "", errors.New("station not found")
}

func (m *Memory) CreateClaimEdit(ctx context.Context, token, tagline, refundPolicy string, invoice bool, minTopup float64) (int64, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	c, ok := m.claims[token]
	if !ok || !c.Verified {
		return 0, errors.New("claim token not verified")
	}
	var slug, name string
	found := false
	for _, st := range m.snap.Stations {
		if st.ID == c.StationID {
			slug, name, found = st.Slug, st.Name, true
			break
		}
	}
	if !found {
		return 0, errors.New("station not found")
	}
	m.nextClaimEdit++
	edit := ClaimEdit{
		ID: m.nextClaimEdit, StationSlug: slug, StationName: name,
		Tagline: tagline, RefundPolicy: refundPolicy, Invoice: invoice, MinTopup: minTopup,
		Status: "pending", CreatedAt: time.Now(),
	}
	m.claimEdits = append(m.claimEdits, edit)
	return edit.ID, nil
}

func (m *Memory) ListClaimEdits(ctx context.Context, includeDecided bool) ([]ClaimEdit, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	out := make([]ClaimEdit, 0)
	for _, e := range m.claimEdits {
		if includeDecided || e.Status == "pending" {
			out = append(out, e)
		}
	}
	return out, nil
}

func (m *Memory) ApproveClaimEdit(ctx context.Context, id int64, note string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	for i := range m.claimEdits {
		if m.claimEdits[i].ID != id || m.claimEdits[i].Status != "pending" {
			continue
		}
		e := m.claimEdits[i]
		for j := range m.snap.Stations {
			if m.snap.Stations[j].Slug == e.StationSlug {
				m.snap.Stations[j].Tagline = e.Tagline
				m.snap.Stations[j].RefundPolicy = e.RefundPolicy
				m.snap.Stations[j].Invoice = e.Invoice
				m.snap.Stations[j].MinTopupCNY = e.MinTopup
			}
		}
		m.claimEdits[i].Status = "approved"
		m.claimEdits[i].ReviewNote = note
		return nil
	}
	return errNotFound
}

func (m *Memory) RejectClaimEdit(ctx context.Context, id int64, note string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	for i := range m.claimEdits {
		if m.claimEdits[i].ID == id && m.claimEdits[i].Status == "pending" {
			m.claimEdits[i].Status = "rejected"
			m.claimEdits[i].ReviewNote = note
			return nil
		}
	}
	return errNotFound
}

func (m *Memory) ClaimEditEmail(ctx context.Context, id int64) (string, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	for _, e := range m.claimEdits {
		if e.ID != id {
			continue
		}
		for _, st := range m.snap.Stations {
			if st.Slug != e.StationSlug {
				continue
			}
			for _, c := range m.claims {
				if c.StationID == st.ID {
					return c.Email, nil
				}
			}
		}
	}
	return "", errNotFound
}

func (m *Memory) ClaimEditsByToken(ctx context.Context, token string, limit int) ([]ClaimEdit, error) {
	if limit <= 0 {
		limit = 20
	}
	m.mu.RLock()
	defer m.mu.RUnlock()
	c, ok := m.claims[token]
	if !ok || !c.Verified {
		return nil, errNotFound
	}
	slug := ""
	for _, st := range m.snap.Stations {
		if st.ID == c.StationID {
			slug = st.Slug
			break
		}
	}
	out := make([]ClaimEdit, 0)
	for i := len(m.claimEdits) - 1; i >= 0 && len(out) < limit; i-- {
		if m.claimEdits[i].StationSlug == slug {
			out = append(out, m.claimEdits[i])
		}
	}
	return out, nil
}

func (m *Memory) CreateSubscription(ctx context.Context, sub Subscription) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.subscriptions[sub.ID] = sub
	return nil
}

func (m *Memory) ListSubscriptions(ctx context.Context) ([]Subscription, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	out := make([]Subscription, 0, len(m.subscriptions))
	for _, s := range m.subscriptions {
		out = append(out, s)
	}
	sort.SliceStable(out, func(i, j int) bool { return out[i].CreatedAt.Before(out[j].CreatedAt) })
	return out, nil
}

func (m *Memory) GetSubscription(ctx context.Context, id string) (Subscription, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	sub, ok := m.subscriptions[id]
	if !ok {
		return Subscription{}, errNotFound
	}
	return sub, nil
}

func (m *Memory) UpdateSubscription(ctx context.Context, sub Subscription) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	cur, ok := m.subscriptions[sub.ID]
	if !ok {
		return errNotFound
	}
	cur.Kinds = sub.Kinds
	cur.Stations = sub.Stations
	cur.Enabled = sub.Enabled
	m.subscriptions[sub.ID] = cur
	return nil
}

func (m *Memory) SetDeliveryState(ctx context.Context, id string, attempts int, lastErr string, next *time.Time, lastSent *time.Time) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	sub, ok := m.subscriptions[id]
	if !ok {
		return errNotFound
	}
	sub.Attempts = attempts
	sub.LastError = lastErr
	sub.NextAttemptAt = next
	if lastSent != nil {
		sub.LastSentAt = *lastSent
	}
	m.subscriptions[id] = sub
	return nil
}

func (m *Memory) TouchSubscription(ctx context.Context, id string, at time.Time) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if s, ok := m.subscriptions[id]; ok {
		s.LastSentAt = at
		m.subscriptions[id] = s
	}
	return nil
}

func (m *Memory) DeleteSubscription(ctx context.Context, id string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.subscriptions, id)
	return nil
}

func (m *Memory) RecordOutbound(ctx context.Context, slug string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.outbound = append(m.outbound, outboundClick{Slug: slug, At: time.Now().UTC()})
	return nil
}

func (m *Memory) OutboundCount(ctx context.Context, slug string) (int, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	cutoff := time.Now().AddDate(0, 0, -30)
	n := 0
	for _, c := range m.outbound {
		if c.Slug == slug && c.At.After(cutoff) {
			n++
		}
	}
	return n, nil
}

func (m *Memory) CreateUser(ctx context.Context, email, name, passwordHash, role string) (User, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if _, ok := m.userByEmail[email]; ok {
		return User{}, errors.New("email already registered")
	}
	u := User{ID: m.nextUserID, Email: email, Name: name, PasswordHash: passwordHash, Role: role, CreatedAt: time.Now().UTC()}
	m.nextUserID++
	m.users[u.ID] = u
	m.userByEmail[email] = u.ID
	return u, nil
}

func (m *Memory) UserByEmail(ctx context.Context, email string) (User, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	id, ok := m.userByEmail[email]
	if !ok {
		return User{}, errNotFound
	}
	return m.users[id], nil
}

func (m *Memory) CreateSession(ctx context.Context, tokenHash string, userID int64, expires time.Time) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.sessions[tokenHash] = memSession{UserID: userID, ExpiresAt: expires}
	return nil
}

func (m *Memory) UserBySession(ctx context.Context, tokenHash string) (User, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	s, ok := m.sessions[tokenHash]
	if !ok || s.ExpiresAt.Before(time.Now()) {
		return User{}, errNotFound
	}
	u, ok := m.users[s.UserID]
	if !ok {
		return User{}, errNotFound
	}
	return u, nil
}

func (m *Memory) DeleteSession(ctx context.Context, tokenHash string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.sessions, tokenHash)
	return nil
}

func (m *Memory) ListSubmissions(ctx context.Context) ([]Submission, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	out := make([]Submission, len(m.submissions))
	copy(out, m.submissions)
	return out, nil
}

func (m *Memory) UpdateSubmissionStatus(ctx context.Context, id int64, status string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	for i := range m.submissions {
		if m.submissions[i].ID == id {
			m.submissions[i].Status = status
			return nil
		}
	}
	return errNotFound
}

func (m *Memory) DeleteReview(ctx context.Context, id int64) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	for i := range m.reviews {
		if m.reviews[i].ID == id {
			m.reviews = append(m.reviews[:i], m.reviews[i+1:]...)
			return nil
		}
	}
	return errNotFound
}

func (m *Memory) RecordDelivery(ctx context.Context, d Delivery) (int64, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if d.CreatedAt.IsZero() {
		d.CreatedAt = time.Now()
	}
	d.ID = m.nextDelivery
	m.nextDelivery++
	m.deliveries = append(m.deliveries, d)
	return d.ID, nil
}

func (m *Memory) ListDeliveries(ctx context.Context, subscriptionID string, limit int) ([]Delivery, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	out := make([]Delivery, 0, limit)
	for i := len(m.deliveries) - 1; i >= 0 && len(out) < limit; i-- {
		if m.deliveries[i].SubscriptionID == subscriptionID {
			out = append(out, m.deliveries[i])
		}
	}
	return out, nil
}

func (m *Memory) GetDelivery(ctx context.Context, id int64) (Delivery, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	for _, d := range m.deliveries {
		if d.ID == id {
			return d, nil
		}
	}
	return Delivery{}, errNotFound
}

func (m *Memory) ListUsers(ctx context.Context) ([]User, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	out := make([]User, 0, len(m.users))
	for _, u := range m.users {
		out = append(out, u)
	}
	sort.SliceStable(out, func(i, j int) bool { return out[i].ID < out[j].ID })
	return out, nil
}

func (m *Memory) UpdateUserRole(ctx context.Context, id int64, role string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	u, ok := m.users[id]
	if !ok {
		return errNotFound
	}
	u.Role = role
	m.users[id] = u
	return nil
}

func (m *Memory) SetUserVerified(ctx context.Context, id int64) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	u, ok := m.users[id]
	if !ok {
		return errNotFound
	}
	u.EmailVerified = true
	m.users[id] = u
	return nil
}

func (m *Memory) CreateAuthToken(ctx context.Context, tokenHash string, userID int64, kind string, expires time.Time) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.authTokens[tokenHash] = memToken{UserID: userID, Kind: kind, ExpiresAt: expires}
	return nil
}

func (m *Memory) ConsumeAuthToken(ctx context.Context, tokenHash, kind string) (int64, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	t, ok := m.authTokens[tokenHash]
	if !ok || t.Kind != kind || t.ExpiresAt.Before(time.Now()) {
		return 0, errNotFound
	}
	delete(m.authTokens, tokenHash)
	return t.UserID, nil
}

func (m *Memory) UpdatePassword(ctx context.Context, id int64, passwordHash string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	u, ok := m.users[id]
	if !ok {
		return errNotFound
	}
	u.PasswordHash = passwordHash
	m.users[id] = u
	return nil
}

func (m *Memory) DeleteUserSessions(ctx context.Context, userID int64) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	for k, s := range m.sessions {
		if s.UserID == userID {
			delete(m.sessions, k)
		}
	}
	return nil
}

func (m *Memory) CreateFreeKey(ctx context.Context, k FreeKey) (int64, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if k.CreatedAt.IsZero() {
		k.CreatedAt = time.Now()
	}
	if k.Status == "" {
		k.Status = "unknown"
	}
	k.ID = m.nextFreeKey
	m.nextFreeKey++
	m.freeKeys[k.ID] = k
	return k.ID, nil
}

func (m *Memory) ListFreeKeys(ctx context.Context, includeInvalid bool, q string) ([]FreeKey, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	needle := strings.ToLower(strings.TrimSpace(q))
	out := make([]FreeKey, 0, len(m.freeKeys))
	for _, k := range m.freeKeys {
		if !includeInvalid && k.Status == "invalid" {
			continue
		}
		if needle != "" && !strings.Contains(strings.ToLower(k.Provider+" "+k.BaseURL+" "+k.Models+" "+k.Submitter), needle) {
			continue
		}
		out = append(out, k)
	}
	sort.SliceStable(out, func(i, j int) bool { return out[i].CreatedAt.After(out[j].CreatedAt) })
	return out, nil
}

func (m *Memory) GetFreeKey(ctx context.Context, id int64) (FreeKey, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	k, ok := m.freeKeys[id]
	if !ok {
		return FreeKey{}, errNotFound
	}
	return k, nil
}

func (m *Memory) IncrementFreeKeyClaims(ctx context.Context, id int64) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	k, ok := m.freeKeys[id]
	if !ok {
		return errNotFound
	}
	k.Claims++
	m.freeKeys[id] = k
	return nil
}

func (m *Memory) DeleteFreeKey(ctx context.Context, id int64) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.freeKeys, id)
	return nil
}

func (m *Memory) RecordStatus(ctx context.Context, s StatusSnapshot) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if s.CheckedAt.IsZero() {
		s.CheckedAt = time.Now()
	}
	m.statuses = append(m.statuses, s)
	return nil
}

func (m *Memory) StatusHistory(ctx context.Context, since time.Time) ([]StatusSnapshot, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	out := make([]StatusSnapshot, 0, len(m.statuses))
	for _, s := range m.statuses {
		if s.CheckedAt.After(since) {
			out = append(out, s)
		}
	}
	return out, nil
}

func (m *Memory) CreateIncident(ctx context.Context, i Incident) (int64, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if i.StartedAt.IsZero() {
		i.StartedAt = time.Now()
	}
	i.CreatedAt = time.Now()
	i.ID = m.nextIncident
	m.nextIncident++
	m.incidents[i.ID] = i
	return i.ID, nil
}

func (m *Memory) ListIncidents(ctx context.Context, includeResolved bool) ([]Incident, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	out := make([]Incident, 0, len(m.incidents))
	for _, i := range m.incidents {
		if !includeResolved && i.Status == "resolved" {
			continue
		}
		out = append(out, i)
	}
	sort.SliceStable(out, func(a, b int) bool { return out[a].StartedAt.After(out[b].StartedAt) })
	return out, nil
}

func (m *Memory) UpdateIncident(ctx context.Context, id int64, status, body string, resolvedAt *time.Time) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	i, ok := m.incidents[id]
	if !ok {
		return errNotFound
	}
	i.Status = status
	i.Body = body
	i.ResolvedAt = resolvedAt
	m.incidents[id] = i
	return nil
}

func (m *Memory) DeleteIncident(ctx context.Context, id int64) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.incidents, id)
	return nil
}

func (m *Memory) RecordProviderStatus(ctx context.Context, p ProviderSnapshot) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if p.CheckedAt.IsZero() {
		p.CheckedAt = time.Now()
	}
	m.providers = append(m.providers, p)
	return nil
}

func (m *Memory) ProviderStatusHistory(ctx context.Context, since time.Time) ([]ProviderSnapshot, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	out := make([]ProviderSnapshot, 0, len(m.providers))
	for _, p := range m.providers {
		if p.CheckedAt.After(since) {
			out = append(out, p)
		}
	}
	return out, nil
}

func (m *Memory) RecordReachability(ctx context.Context, rows []Reachability) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.reachability == nil {
		m.reachability = map[string]Reachability{}
	}
	for _, r := range rows {
		m.reachability[r.Slug] = r
		m.reachHistory = append(m.reachHistory, r)
	}
	return nil
}

func (m *Memory) GroupByStationService(ctx context.Context, slug, service string) (int64, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	var stationID int64
	for _, st := range m.snap.Stations {
		if st.Slug == slug {
			stationID = st.ID
			break
		}
	}
	if stationID == 0 {
		return 0, errNotFound
	}
	for _, g := range m.snap.Groups {
		if g.StationID == stationID && g.Service == service {
			return g.ID, nil
		}
	}
	return 0, errNotFound
}

func (m *Memory) InsertProbe(ctx context.Context, groupID int64, status string, latencyMS int, at time.Time) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.snap.Probes = append(m.snap.Probes, Probe{GroupID: groupID, Status: status, LatencyMS: latencyMS, ProbedAt: at})
	return nil
}

func (m *Memory) ReachabilityHistory(ctx context.Context, slug string, since time.Time) ([]Reachability, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	out := make([]Reachability, 0)
	for _, r := range m.reachHistory {
		if r.Slug == slug && r.CheckedAt.After(since) {
			out = append(out, r)
		}
	}
	return out, nil
}

func (m *Memory) ListReachability(ctx context.Context) (map[string]Reachability, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	out := make(map[string]Reachability, len(m.reachability))
	for k, v := range m.reachability {
		out[k] = v
	}
	return out, nil
}

func (m *Memory) PruneProviderStatus(ctx context.Context, before time.Time) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	kept := m.providers[:0]
	for _, p := range m.providers {
		if p.CheckedAt.After(before) {
			kept = append(kept, p)
		}
	}
	m.providers = kept
	return nil
}

func (m *Memory) PruneComponentStatus(ctx context.Context, before time.Time) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	kept := m.statuses[:0]
	for _, s := range m.statuses {
		if s.CheckedAt.After(before) {
			kept = append(kept, s)
		}
	}
	m.statuses = kept
	return nil
}

func (m *Memory) RecordAdminAction(ctx context.Context, a AdminAction) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if a.CreatedAt.IsZero() {
		a.CreatedAt = time.Now()
	}
	a.ID = m.nextAdminAct
	m.nextAdminAct++
	m.adminActions = append(m.adminActions, a)
	return nil
}

func (m *Memory) ListAdminActions(ctx context.Context, limit int) ([]AdminAction, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	out := make([]AdminAction, 0, limit)
	for i := len(m.adminActions) - 1; i >= 0 && len(out) < limit; i-- {
		out = append(out, m.adminActions[i])
	}
	return out, nil
}

func (m *Memory) ListAllReviews(ctx context.Context) ([]AdminReview, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	names := map[int64][2]string{}
	for _, st := range m.snap.Stations {
		names[st.ID] = [2]string{st.Slug, st.Name}
	}
	out := make([]AdminReview, 0, len(m.reviews))
	for _, r := range m.reviews {
		n := names[r.StationID]
		out = append(out, AdminReview{
			ID: r.ID, StationSlug: n[0], StationName: n[1],
			Author: r.Author, Rating: r.Rating, Body: r.Body, CreatedAt: r.CreatedAt,
		})
	}
	sort.SliceStable(out, func(i, j int) bool { return out[i].ID > out[j].ID })
	return out, nil
}

func (m *Memory) Counts(ctx context.Context) (map[string]int, time.Time, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	last := m.snap.TakenAt
	return map[string]int{
		"stations": len(m.snap.Stations),
		"groups":   len(m.snap.Groups),
		"models":   len(m.snap.Models),
		"probes":   len(m.snap.Probes),
	}, last, nil
}

func (m *Memory) UpdateSubmissionTerms(ctx context.Context, id int64, payments []string, invoice bool, refundPolicy string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	for i := range m.submissions {
		if m.submissions[i].ID == id {
			m.submissions[i].Payments = payments
			m.submissions[i].Invoice = invoice
			m.submissions[i].RefundPolicy = refundPolicy
			return nil
		}
	}
	return fmt.Errorf("submission %d not found", id)
}

func (m *Memory) ApplyTermsToStation(ctx context.Context, id int64) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	var sub *Submission
	for i := range m.submissions {
		if m.submissions[i].ID == id {
			sub = &m.submissions[i]
			break
		}
	}
	if sub == nil {
		return fmt.Errorf("submission %d not found", id)
	}
	for i := range m.snap.Stations {
		if strings.EqualFold(m.snap.Stations[i].Domain, sub.Domain) {
			m.snap.Stations[i].Payments = sub.Payments
			m.snap.Stations[i].Invoice = sub.Invoice
			m.snap.Stations[i].RefundPolicy = sub.RefundPolicy
			return nil
		}
	}
	return nil
}

func (m *Memory) UpdateStationIcon(ctx context.Context, slug, iconURL string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	for i := range m.snap.Stations {
		if m.snap.Stations[i].Slug == slug {
			m.snap.Stations[i].IconURL = iconURL
			return nil
		}
	}
	return fmt.Errorf("station %s not found", slug)
}

func sameDay(a, b time.Time) bool {
	ay, am, ad := a.Date()
	by, bm, bd := b.Date()
	return ay == by && am == bm && ad == bd
}

// SnapshotPriceDaily aggregates the in-memory catalogue into one row per model
// and station for the given day, mirroring the Postgres job.
func (m *Memory) SnapshotPriceDaily(ctx context.Context, day time.Time) (int, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	stationSlug := make(map[int64]string, len(m.snap.Stations))
	for _, st := range m.snap.Stations {
		stationSlug[st.ID] = st.Slug
	}
	stationOfGroup := make(map[int64]int64, len(m.snap.Groups))
	for _, g := range m.snap.Groups {
		stationOfGroup[g.ID] = g.StationID
	}
	agg := make(map[string]*PriceDaily)
	for _, gm := range m.snap.GroupModels {
		if gm.InputCNY == nil || *gm.InputCNY <= 0 {
			continue
		}
		slug := stationSlug[stationOfGroup[gm.GroupID]]
		if slug == "" {
			continue
		}
		key := gm.ModelID + "|" + slug
		out := 0.0
		if gm.OutputCNY != nil {
			out = *gm.OutputCNY
		}
		if cur, ok := agg[key]; ok {
			if *gm.InputCNY < cur.MinInputCNY {
				cur.MinInputCNY = *gm.InputCNY
			}
			if out > 0 && (cur.MinOutputCNY == 0 || out < cur.MinOutputCNY) {
				cur.MinOutputCNY = out
			}
			cur.Offers++
			continue
		}
		agg[key] = &PriceDaily{Day: day, ModelID: gm.ModelID, StationSlug: slug, MinInputCNY: *gm.InputCNY, MinOutputCNY: out, Offers: 1}
	}
	kept := m.priceDaily[:0]
	for _, p := range m.priceDaily {
		if !sameDay(p.Day, day) {
			kept = append(kept, p)
		}
	}
	m.priceDaily = kept
	for _, p := range agg {
		m.priceDaily = append(m.priceDaily, *p)
	}
	return len(agg), nil
}

// PriceDailyHistory aggregates stored rows into one point per day.
func (m *Memory) PriceDailyHistory(ctx context.Context, modelID string, days int) ([]PriceDailyPoint, error) {
	if days <= 0 {
		days = 90
	}
	m.mu.RLock()
	defer m.mu.RUnlock()
	cutoff := time.Now().UTC().AddDate(0, 0, -days)
	byDay := map[string]*PriceDailyPoint{}
	for _, p := range m.priceDaily {
		if p.ModelID != modelID || p.Day.Before(cutoff) {
			continue
		}
		key := p.Day.Format("2006-01-02")
		cur := byDay[key]
		if cur == nil {
			cur = &PriceDailyPoint{Day: p.Day, MinInputCNY: p.MinInputCNY, MinOutputCNY: p.MinOutputCNY, Stations: 1, Offers: p.Offers}
			byDay[key] = cur
			continue
		}
		if p.MinInputCNY < cur.MinInputCNY {
			cur.MinInputCNY = p.MinInputCNY
		}
		if p.MinOutputCNY > 0 && (cur.MinOutputCNY == 0 || p.MinOutputCNY < cur.MinOutputCNY) {
			cur.MinOutputCNY = p.MinOutputCNY
		}
		cur.Stations++
		cur.Offers += p.Offers
	}
	out := make([]PriceDailyPoint, 0, len(byDay))
	for _, p := range byDay {
		out = append(out, *p)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Day.Before(out[j].Day) })
	return out, nil
}

// StationPriceHistory returns one station's daily price for a single model.
func (m *Memory) StationPriceHistory(ctx context.Context, slug, modelID string, days int) ([]PriceDailyPoint, error) {
	if days <= 0 {
		days = 90
	}
	m.mu.RLock()
	defer m.mu.RUnlock()
	cutoff := time.Now().UTC().AddDate(0, 0, -days)
	out := make([]PriceDailyPoint, 0, days)
	for _, p := range m.priceDaily {
		if p.StationSlug != slug || p.ModelID != modelID || p.Day.Before(cutoff) {
			continue
		}
		out = append(out, PriceDailyPoint{Day: p.Day, MinInputCNY: p.MinInputCNY, MinOutputCNY: p.MinOutputCNY, Stations: 1, Offers: p.Offers})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Day.Before(out[j].Day) })
	return out, nil
}

package store

import (
	"context"
	"embed"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

//go:embed migrations/*.sql
var migrations embed.FS

type Station struct {
	ID           int64     `json:"-"`
	Slug         string    `json:"slug"`
	Name         string    `json:"name"`
	Domain       string    `json:"domain"`
	BaseURL      string    `json:"baseUrl"`
	Tagline      string    `json:"tagline"`
	RateCNY      float64   `json:"rateCny"`
	MinTopupCNY  float64   `json:"minTopupCny"`
	Payments     []string  `json:"payments"`
	Invoice      bool      `json:"invoice"`
	RefundPolicy string    `json:"refundPolicy"`
	Claimed      bool      `json:"claimed"`
	IconURL      string    `json:"iconUrl"`
	ListedAt     time.Time `json:"listedAt"`
}

type Group struct {
	ID         int64   `json:"-"`
	StationID  int64   `json:"-"`
	Slug       string  `json:"slug"`
	Name       string  `json:"name"`
	Service    string  `json:"service"`
	Multiplier float64 `json:"multiplier"`
}

type Model struct {
	ID             string    `json:"id"`
	Vendor         string    `json:"vendor"`
	VendorName     string    `json:"vendorName"`
	Name           string    `json:"name"`
	Service        string    `json:"service"`
	ContextWindow  int       `json:"contextWindow"`
	InputUSD       float64   `json:"inputUsd"`
	OutputUSD      float64   `json:"outputUsd"`
	ReleasedAt     time.Time `json:"releasedAt"`
	CodingScore    float64   `json:"codingScore"`
	ReasoningScore float64   `json:"reasoningScore"`
	FreeTier       bool      `json:"freeTier"`
	FreeNote       string    `json:"freeNote"`
	Summary        string    `json:"summary"`
	Category       string    `json:"category"`
}

type Submission struct {
	ID        int64     `json:"id"`
	Name      string    `json:"name"`
	Domain    string    `json:"domain"`
	BaseURL   string    `json:"baseUrl"`
	Contact   string    `json:"contact"`
	Note      string    `json:"note"`
	Status    string    `json:"status"`
	CreatedAt time.Time `json:"createdAt"`
}

// ClaimEdit is a pending change to a claimed station profile.
type ClaimEdit struct {
	ID           int64     `json:"id"`
	StationSlug  string    `json:"stationSlug"`
	StationName  string    `json:"stationName"`
	Tagline      string    `json:"tagline"`
	RefundPolicy string    `json:"refundPolicy"`
	Invoice      bool      `json:"invoice"`
	MinTopup     float64   `json:"minTopupCny"`
	Status       string    `json:"status"`
	ReviewNote   string    `json:"reviewNote"`
	CreatedAt    time.Time `json:"createdAt"`
}

// ClaimedStation is the editable profile behind a verified claim token.
type ClaimedStation struct {
	Slug         string  `json:"slug"`
	Name         string  `json:"name"`
	Tagline      string  `json:"tagline"`
	RefundPolicy string  `json:"refundPolicy"`
	Invoice      bool    `json:"invoice"`
	MinTopup     float64 `json:"minTopupCny"`
	ReviewStatus string  `json:"reviewStatus"`
	ReviewNote   string  `json:"reviewNote"`
}

type Claim struct {
	ID        int64     `json:"id"`
	StationID int64     `json:"-"`
	Email     string    `json:"email"`
	Token     string    `json:"token"`
	Verified  bool      `json:"verified"`
	CreatedAt time.Time `json:"createdAt"`
}

type GroupModel struct {
	GroupID      int64    `json:"-"`
	ModelID      string   `json:"modelId"`
	InputCNY     *float64 `json:"inputCny"`
	OutputCNY    *float64 `json:"outputCny"`
	CacheReadCNY *float64 `json:"cacheReadCny"`
}

type Probe struct {
	GroupID   int64     `json:"-"`
	Status    string    `json:"status"`
	LatencyMS int       `json:"latencyMs"`
	ProbedAt  time.Time `json:"probedAt"`
}

type Purity struct {
	GroupID       int64     `json:"-"`
	Score         int       `json:"score"`
	Tier          string    `json:"tier"`
	DetectedModel string    `json:"detectedModel"`
	CheckedAt     time.Time `json:"checkedAt"`
}

type PricePoint struct {
	GroupID    int64     `json:"-"`
	ModelID    string    `json:"modelId"`
	InputCNY   *float64  `json:"inputCny"`
	OutputCNY  *float64  `json:"outputCny"`
	RecordedAt time.Time `json:"recordedAt"`
}

type Review struct {
	ID        int64     `json:"id"`
	StationID int64     `json:"-"`
	Author    string    `json:"author"`
	Rating    int       `json:"rating"`
	Body      string    `json:"body"`
	Verified  bool      `json:"verified"`
	CreatedAt time.Time `json:"createdAt"`
}

// AdminReview is a review joined with its station for moderation.
type AdminReview struct {
	ID          int64     `json:"id"`
	StationSlug string    `json:"stationSlug"`
	StationName string    `json:"stationName"`
	Author      string    `json:"author"`
	Rating      int       `json:"rating"`
	Body        string    `json:"body"`
	CreatedAt   time.Time `json:"createdAt"`
}

// User is an account; PasswordHash is never serialized.
type User struct {
	ID            int64     `json:"id"`
	Email         string    `json:"email"`
	Name          string    `json:"name"`
	PasswordHash  string    `json:"-"`
	Role          string    `json:"role"`
	EmailVerified bool      `json:"emailVerified"`
	CreatedAt     time.Time `json:"createdAt"`
}

// FreeKey is a user-submitted shared API key.
type FreeKey struct {
	ID            int64      `json:"id"`
	Provider      string     `json:"provider"`
	BaseURL       string     `json:"baseUrl"`
	Key           string     `json:"-"`
	Models        string     `json:"models"`
	Note          string     `json:"note"`
	Submitter     string     `json:"submitter"`
	Status        string     `json:"status"`
	Claims        int        `json:"claims"`
	CreatedAt     time.Time  `json:"createdAt"`
	LastCheckedAt *time.Time `json:"lastCheckedAt,omitempty"`
}

// StatusSnapshot is one recorded component check.
type StatusSnapshot struct {
	Component string    `json:"component"`
	Status    string    `json:"status"`
	Detail    string    `json:"detail"`
	CheckedAt time.Time `json:"checkedAt"`
}

// Incident is a public status incident.
type Incident struct {
	ID         int64      `json:"id"`
	Title      string     `json:"title"`
	Status     string     `json:"status"`
	Component  string     `json:"component"`
	Body       string     `json:"body"`
	StartedAt  time.Time  `json:"startedAt"`
	ResolvedAt *time.Time `json:"resolvedAt,omitempty"`
	CreatedAt  time.Time  `json:"createdAt"`
}

// Reachability is the latest credential-free homepage check for a station.
type Reachability struct {
	Slug      string    `json:"slug"`
	Status    string    `json:"status"`
	LatencyMS int       `json:"latencyMs"`
	CheckedAt time.Time `json:"checkedAt"`
}

// ProviderSnapshot is one recorded upstream model provider check.
type ProviderSnapshot struct {
	Provider  string    `json:"provider"`
	Status    string    `json:"status"`
	Detail    string    `json:"detail"`
	CheckedAt time.Time `json:"checkedAt"`
}

// AdminAction is one recorded admin mutation.
type AdminAction struct {
	ID         int64     `json:"id"`
	ActorID    int64     `json:"actorId"`
	ActorEmail string    `json:"actorEmail"`
	Action     string    `json:"action"`
	Target     string    `json:"target"`
	Detail     string    `json:"detail"`
	IP         string    `json:"ip"`
	CreatedAt  time.Time `json:"createdAt"`
}

// Delivery is one webhook/email attempt for an alert subscription.
type Delivery struct {
	ID             int64     `json:"id"`
	SubscriptionID string    `json:"subscriptionId"`
	Event          string    `json:"event"`
	Status         int       `json:"status"`
	OK             bool      `json:"ok"`
	DurationMS     int       `json:"durationMs"`
	Response       string    `json:"response"`
	Payload        string    `json:"-"`
	CreatedAt      time.Time `json:"createdAt"`
}

// Subscription is a persisted webhook alert subscription.
type Subscription struct {
	ID            string     `json:"id"`
	Channel       string     `json:"channel"`
	URL           string     `json:"url"`
	Email         string     `json:"email"`
	Secret        string     `json:"-"`
	Kinds         []string   `json:"kinds"`
	Stations      []string   `json:"stations"`
	Providers     []string   `json:"providers"`
	Enabled       bool       `json:"enabled"`
	Attempts      int        `json:"attempts"`
	LastError     string     `json:"lastError"`
	NextAttemptAt *time.Time `json:"nextAttemptAt,omitempty"`
	CreatedAt     time.Time  `json:"createdAt"`
	LastSentAt    time.Time  `json:"lastSentAt"`
}

type AuditReport struct {	ID        string    `json:"id"`
	TargetURL string    `json:"targetUrl"`
	Model     string    `json:"model"`
	Passed    bool      `json:"passed"`
	Payload   []byte    `json:"-"`
	CreatedAt time.Time `json:"createdAt"`
}

// Snapshot is the full in-memory slice of the database used for scoring.
type Snapshot struct {
	Stations    []Station
	Groups      []Group
	Models      []Model
	GroupModels []GroupModel
	Probes      []Probe
	Purity      []Purity
	Prices      []PricePoint
	TakenAt     time.Time
}

type Store struct {
	pool *pgxpool.Pool
}

func Open(ctx context.Context, dsn string) (*Store, error) {
	cfg, err := pgxpool.ParseConfig(dsn)
	if err != nil {
		return nil, fmt.Errorf("parse dsn: %w", err)
	}
	cfg.MaxConns = 8
	pool, err := pgxpool.NewWithConfig(ctx, cfg)
	if err != nil {
		return nil, err
	}
	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("ping: %w", err)
	}
	return &Store{pool: pool}, nil
}

func (s *Store) Close() { s.pool.Close() }

// SyncModels upserts the embedded model catalog so existing databases pick up
// new models without a full reseed.
func (s *Store) SyncModels(ctx context.Context) error {
	models := CatalogModels()
	if len(models) == 0 {
		return nil
	}
	batch := &pgx.Batch{}
	for _, m := range models {
		batch.Queue(`
			INSERT INTO models (id, vendor, vendor_name, name, service, context_window, input_usd, output_usd,
			                    released_at, coding_score, reasoning_score, free_tier, free_note, summary, category)
			VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15)
			ON CONFLICT (id) DO UPDATE SET
				vendor = EXCLUDED.vendor, vendor_name = EXCLUDED.vendor_name, name = EXCLUDED.name,
				service = EXCLUDED.service, context_window = EXCLUDED.context_window,
				input_usd = EXCLUDED.input_usd, output_usd = EXCLUDED.output_usd,
				released_at = EXCLUDED.released_at, summary = EXCLUDED.summary,
				category = EXCLUDED.category`,
			m.ID, m.Vendor, m.VendorName, m.Name, m.Service, m.ContextWindow, m.InputUSD, m.OutputUSD,
			m.ReleasedAt, m.CodingScore, m.ReasoningScore, m.FreeTier, m.FreeNote, m.Summary, m.Category)
	}
	br := s.pool.SendBatch(ctx, batch)
	defer br.Close()
	for range models {
		if _, err := br.Exec(); err != nil {
			return err
		}
	}
	ids := make([]string, len(models))
	for i, m := range models {
		ids[i] = m.ID
	}
	if _, err := s.pool.Exec(ctx, `DELETE FROM models WHERE NOT (id = ANY($1))`, ids); err != nil {
		return err
	}
	return nil
}

func (s *Store) Migrate(ctx context.Context) error {
	entries, err := migrations.ReadDir("migrations")
	if err != nil {
		return err
	}
	names := make([]string, 0, len(entries))
	for _, e := range entries {
		names = append(names, e.Name())
	}
	sort.Strings(names)
	for _, n := range names {
		b, err := migrations.ReadFile("migrations/" + n)
		if err != nil {
			return err
		}
		if _, err := s.pool.Exec(ctx, string(b)); err != nil {
			return fmt.Errorf("migration %s: %w", n, err)
		}
	}
	return nil
}

func (s *Store) IsEmpty(ctx context.Context) (bool, error) {
	var n int
	if err := s.pool.QueryRow(ctx, `SELECT count(*) FROM stations`).Scan(&n); err != nil {
		return false, err
	}
	return n == 0, nil
}

func (s *Store) Snapshot(ctx context.Context) (*Snapshot, error) {
	snap := &Snapshot{TakenAt: time.Now().UTC()}

	rows, err := s.pool.Query(ctx, `
		SELECT id, slug, name, domain, base_url, tagline, rate_cny, min_topup_cny,
		       payments, invoice, refund_policy, claimed, icon_url, listed_at
		FROM stations ORDER BY id`)
	if err != nil {
		return nil, err
	}
	snap.Stations, err = pgx.CollectRows(rows, func(r pgx.CollectableRow) (Station, error) {
		var st Station
		err := r.Scan(&st.ID, &st.Slug, &st.Name, &st.Domain, &st.BaseURL, &st.Tagline,
			&st.RateCNY, &st.MinTopupCNY, &st.Payments, &st.Invoice, &st.RefundPolicy, &st.Claimed,
			&st.IconURL, &st.ListedAt)
		return st, err
	})
	if err != nil {
		return nil, err
	}

	rows, err = s.pool.Query(ctx, `SELECT id, station_id, slug, name, service, multiplier FROM station_groups ORDER BY id`)
	if err != nil {
		return nil, err
	}
	snap.Groups, err = pgx.CollectRows(rows, func(r pgx.CollectableRow) (Group, error) {
		var g Group
		err := r.Scan(&g.ID, &g.StationID, &g.Slug, &g.Name, &g.Service, &g.Multiplier)
		return g, err
	})
	if err != nil {
		return nil, err
	}

	rows, err = s.pool.Query(ctx, `
		SELECT id, vendor, vendor_name, name, service, context_window, input_usd, output_usd,
		       COALESCE(released_at, CURRENT_DATE), coding_score, reasoning_score,
		       free_tier, free_note, summary, category
		FROM models ORDER BY released_at DESC NULLS LAST, id`)
	if err != nil {
		return nil, err
	}
	snap.Models, err = pgx.CollectRows(rows, func(r pgx.CollectableRow) (Model, error) {
		var m Model
		err := r.Scan(&m.ID, &m.Vendor, &m.VendorName, &m.Name, &m.Service, &m.ContextWindow,
			&m.InputUSD, &m.OutputUSD, &m.ReleasedAt, &m.CodingScore, &m.ReasoningScore,
			&m.FreeTier, &m.FreeNote, &m.Summary, &m.Category)
		return m, err
	})
	if err != nil {
		return nil, err
	}

	rows, err = s.pool.Query(ctx, `SELECT group_id, model_id, input_cny, output_cny, cache_read_cny FROM group_models`)
	if err != nil {
		return nil, err
	}
	snap.GroupModels, err = pgx.CollectRows(rows, func(r pgx.CollectableRow) (GroupModel, error) {
		var gm GroupModel
		err := r.Scan(&gm.GroupID, &gm.ModelID, &gm.InputCNY, &gm.OutputCNY, &gm.CacheReadCNY)
		return gm, err
	})
	if err != nil {
		return nil, err
	}

	rows, err = s.pool.Query(ctx, `
		SELECT group_id, status, COALESCE(latency_ms, 0), probed_at
		FROM probes WHERE probed_at > now() - interval '36 days'`)
	if err != nil {
		return nil, err
	}
	snap.Probes, err = pgx.CollectRows(rows, func(r pgx.CollectableRow) (Probe, error) {
		var p Probe
		err := r.Scan(&p.GroupID, &p.Status, &p.LatencyMS, &p.ProbedAt)
		return p, err
	})
	if err != nil {
		return nil, err
	}

	rows, err = s.pool.Query(ctx, `
		SELECT group_id, score, tier, COALESCE(detected_model, ''), checked_at
		FROM purity_checks WHERE checked_at > now() - interval '30 days'`)
	if err != nil {
		return nil, err
	}
	snap.Purity, err = pgx.CollectRows(rows, func(r pgx.CollectableRow) (Purity, error) {
		var p Purity
		err := r.Scan(&p.GroupID, &p.Score, &p.Tier, &p.DetectedModel, &p.CheckedAt)
		return p, err
	})
	if err != nil {
		return nil, err
	}

	rows, err = s.pool.Query(ctx, `
		SELECT group_id, model_id, input_cny, output_cny, recorded_at
		FROM price_history WHERE recorded_at > now() - interval '30 days'
		ORDER BY recorded_at`)
	if err != nil {
		return nil, err
	}
	snap.Prices, err = pgx.CollectRows(rows, func(r pgx.CollectableRow) (PricePoint, error) {
		var p PricePoint
		err := r.Scan(&p.GroupID, &p.ModelID, &p.InputCNY, &p.OutputCNY, &p.RecordedAt)
		return p, err
	})
	if err != nil {
		return nil, err
	}

	return snap, nil
}

// InsertProbe appends one probe result; used by the standalone prober.
func (s *Store) InsertProbe(ctx context.Context, groupID int64, status string, latencyMS int, at time.Time) error {
	_, err := s.pool.Exec(ctx, `
		INSERT INTO probes (group_id, status, latency_ms, probed_at)
		VALUES ($1, $2, $3, $4)`, groupID, status, latencyMS, at)
	return err
}

// GroupByStationService resolves a group id from station slug + service.
func (s *Store) GroupByStationService(ctx context.Context, slug, service string) (int64, error) {
	var id int64
	err := s.pool.QueryRow(ctx, `
		SELECT g.id FROM station_groups g
		JOIN stations st ON st.id = g.station_id
		WHERE st.slug = $1 AND g.service = $2
		LIMIT 1`, slug, service).Scan(&id)
	return id, err
}

func (s *Store) StationIDBySlug(ctx context.Context, slug string) (int64, error) {
	var id int64
	err := s.pool.QueryRow(ctx, `SELECT id FROM stations WHERE slug = $1`, slug).Scan(&id)
	return id, err
}

func (s *Store) ListReviews(ctx context.Context, stationID int64) ([]Review, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT id, author, rating, body, verified, created_at
		FROM reviews WHERE station_id = $1 ORDER BY created_at DESC LIMIT 50`, stationID)
	if err != nil {
		return nil, err
	}
	return pgx.CollectRows(rows, func(r pgx.CollectableRow) (Review, error) {
		var rv Review
		err := r.Scan(&rv.ID, &rv.Author, &rv.Rating, &rv.Body, &rv.Verified, &rv.CreatedAt)
		return rv, err
	})
}

func (s *Store) InsertReview(ctx context.Context, stationID int64, author string, rating int, body string) (Review, error) {
	var rv Review
	err := s.pool.QueryRow(ctx, `
		INSERT INTO reviews (station_id, author, rating, body, verified)
		VALUES ($1, $2, $3, $4, false)
		RETURNING id, author, rating, body, verified, created_at`,
		stationID, author, rating, body).
		Scan(&rv.ID, &rv.Author, &rv.Rating, &rv.Body, &rv.Verified, &rv.CreatedAt)
	return rv, err
}

func (s *Store) InsertAudit(ctx context.Context, a AuditReport) error {
	payload := a.Payload
	if len(payload) == 0 {
		payload = []byte(`{}`)
	}
	_, err := s.pool.Exec(ctx, `
		INSERT INTO audit_reports (id, target_url, model, passed, payload, created_at)
		VALUES ($1, $2, $3, $4, $5, $6)`, a.ID, a.TargetURL, a.Model, a.Passed, payload, a.CreatedAt)
	return err
}

func (s *Store) CreateClaimEdit(ctx context.Context, token, tagline, refundPolicy string, invoice bool, minTopup float64) (int64, error) {
	var id int64
	err := s.pool.QueryRow(ctx, `
		INSERT INTO claim_edits (station_id, tagline, refund_policy, invoice, min_topup_cny)
		SELECT station_id, $2, $3, $4, $5 FROM claims WHERE token = $1 AND verified
		RETURNING id`, token, tagline, refundPolicy, invoice, minTopup).Scan(&id)
	return id, err
}

func (s *Store) ListClaimEdits(ctx context.Context, includeDecided bool) ([]ClaimEdit, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT e.id, s.slug, s.name, e.tagline, e.refund_policy, e.invoice, e.min_topup_cny, e.status, e.review_note, e.created_at
		FROM claim_edits e JOIN stations s ON s.id = e.station_id
		WHERE ($1 OR e.status = 'pending') ORDER BY e.id DESC LIMIT 100`, includeDecided)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []ClaimEdit
	for rows.Next() {
		var e ClaimEdit
		if err := rows.Scan(&e.ID, &e.StationSlug, &e.StationName, &e.Tagline, &e.RefundPolicy, &e.Invoice, &e.MinTopup, &e.Status, &e.ReviewNote, &e.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, e)
	}
	return out, rows.Err()
}

func (s *Store) ApproveClaimEdit(ctx context.Context, id int64, note string) error {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	var stationID int64
	err = tx.QueryRow(ctx, `
		UPDATE claim_edits SET status = 'approved', review_note = $2 WHERE id = $1 AND status = 'pending'
		RETURNING station_id`, id, note).Scan(&stationID)
	if err != nil {
		return err
	}
	_, err = tx.Exec(ctx, `
		UPDATE stations SET tagline = e.tagline, refund_policy = e.refund_policy,
			invoice = e.invoice, min_topup_cny = e.min_topup_cny
		FROM claim_edits e WHERE e.id = $1 AND stations.id = e.station_id`, id)
	if err != nil {
		return err
	}
	return tx.Commit(ctx)
}

func (s *Store) RejectClaimEdit(ctx context.Context, id int64, note string) error {
	tag, err := s.pool.Exec(ctx, `UPDATE claim_edits SET status = 'rejected', review_note = $2 WHERE id = $1 AND status = 'pending'`, id, note)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return errors.New("claim edit not found")
	}
	return nil
}

func (s *Store) ClaimEditEmail(ctx context.Context, id int64) (string, error) {
	var email string
	err := s.pool.QueryRow(ctx, `
		SELECT c.email FROM claim_edits e
		JOIN claims c ON c.station_id = e.station_id AND c.verified
		WHERE e.id = $1 LIMIT 1`, id).Scan(&email)
	return email, err
}

func (s *Store) ClaimEditsByToken(ctx context.Context, token string, limit int) ([]ClaimEdit, error) {
	if limit <= 0 {
		limit = 20
	}
	rows, err := s.pool.Query(ctx, `
		SELECT e.id, s.slug, s.name, e.tagline, e.refund_policy, e.invoice, e.min_topup_cny, e.status, e.review_note, e.created_at
		FROM claim_edits e
		JOIN stations s ON s.id = e.station_id
		JOIN claims c ON c.station_id = e.station_id AND c.token = $1 AND c.verified
		ORDER BY e.id DESC LIMIT $2`, token, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []ClaimEdit
	for rows.Next() {
		var e ClaimEdit
		if err := rows.Scan(&e.ID, &e.StationSlug, &e.StationName, &e.Tagline, &e.RefundPolicy, &e.Invoice, &e.MinTopup, &e.Status, &e.ReviewNote, &e.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, e)
	}
	return out, rows.Err()
}

func (s *Store) CreateSubscription(ctx context.Context, sub Subscription) error {
	_, err := s.pool.Exec(ctx, `
		INSERT INTO subscriptions (id, channel, url, email, secret, kinds, stations, providers, enabled, created_at, last_sent_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)`,
		sub.ID, sub.Channel, sub.URL, sub.Email, sub.Secret, sub.Kinds, sub.Stations, sub.Providers,
		sub.Enabled, sub.CreatedAt, sub.LastSentAt)
	return err
}

func (s *Store) ListSubscriptions(ctx context.Context) ([]Subscription, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT id, channel, url, email, secret, kinds, stations, providers, enabled, attempts, last_error,
		       next_attempt_at, created_at, last_sent_at
		FROM subscriptions ORDER BY created_at`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []Subscription
	for rows.Next() {
		var sub Subscription
		if err := rows.Scan(&sub.ID, &sub.Channel, &sub.URL, &sub.Email, &sub.Secret,
			&sub.Kinds, &sub.Stations, &sub.Providers, &sub.Enabled, &sub.Attempts, &sub.LastError,
			&sub.NextAttemptAt, &sub.CreatedAt, &sub.LastSentAt); err != nil {
			return nil, err
		}
		out = append(out, sub)
	}
	return out, rows.Err()
}

func (s *Store) GetSubscription(ctx context.Context, id string) (Subscription, error) {
	var sub Subscription
	err := s.pool.QueryRow(ctx, `
		SELECT id, channel, url, email, secret, kinds, stations, providers, enabled, attempts, last_error,
		       next_attempt_at, created_at, last_sent_at
		FROM subscriptions WHERE id = $1`, id).
		Scan(&sub.ID, &sub.Channel, &sub.URL, &sub.Email, &sub.Secret,
			&sub.Kinds, &sub.Stations, &sub.Providers, &sub.Enabled, &sub.Attempts, &sub.LastError,
			&sub.NextAttemptAt, &sub.CreatedAt, &sub.LastSentAt)
	return sub, err
}

func (s *Store) UpdateSubscription(ctx context.Context, sub Subscription) error {
	_, err := s.pool.Exec(ctx, `
		UPDATE subscriptions SET kinds = $2, stations = $3, providers = $4, enabled = $5 WHERE id = $1`,
		sub.ID, sub.Kinds, sub.Stations, sub.Providers, sub.Enabled)
	return err
}

func (s *Store) SetDeliveryState(ctx context.Context, id string, attempts int, lastErr string, next *time.Time, lastSent *time.Time) error {
	_, err := s.pool.Exec(ctx, `
		UPDATE subscriptions
		SET attempts = $2, last_error = $3, next_attempt_at = $4,
		    last_sent_at = COALESCE($5, last_sent_at)
		WHERE id = $1`, id, attempts, lastErr, next, lastSent)
	return err
}

func (s *Store) TouchSubscription(ctx context.Context, id string, at time.Time) error {
	_, err := s.pool.Exec(ctx, `UPDATE subscriptions SET last_sent_at = $2 WHERE id = $1`, id, at)
	return err
}

func (s *Store) DeleteSubscription(ctx context.Context, id string) error {
	_, err := s.pool.Exec(ctx, `DELETE FROM subscriptions WHERE id = $1`, id)
	return err
}

func (s *Store) RecordOutbound(ctx context.Context, slug string) error {
	_, err := s.pool.Exec(ctx, `INSERT INTO outbound_clicks (station_slug) VALUES ($1)`, slug)
	return err
}

func (s *Store) OutboundCount(ctx context.Context, slug string) (int, error) {
	var n int
	err := s.pool.QueryRow(ctx, `
		SELECT count(*) FROM outbound_clicks
		WHERE station_slug = $1 AND created_at > now() - interval '30 days'`, slug).Scan(&n)
	return n, err
}

func (s *Store) CreateUser(ctx context.Context, email, name, passwordHash, role string) (User, error) {
	var u User
	err := s.pool.QueryRow(ctx, `
		INSERT INTO users (email, name, password_hash, role)
		VALUES ($1, $2, $3, $4)
		RETURNING id, email, name, role, email_verified, created_at`,
		email, name, passwordHash, role).
		Scan(&u.ID, &u.Email, &u.Name, &u.Role, &u.EmailVerified, &u.CreatedAt)
	return u, err
}

func (s *Store) UserByEmail(ctx context.Context, email string) (User, error) {
	var u User
	err := s.pool.QueryRow(ctx, `
		SELECT id, email, name, password_hash, role, email_verified, created_at
		FROM users WHERE email = $1`, email).
		Scan(&u.ID, &u.Email, &u.Name, &u.PasswordHash, &u.Role, &u.EmailVerified, &u.CreatedAt)
	return u, err
}

func (s *Store) CreateSession(ctx context.Context, tokenHash string, userID int64, expires time.Time) error {
	_, err := s.pool.Exec(ctx, `
		INSERT INTO sessions (token_hash, user_id, expires_at) VALUES ($1, $2, $3)`,
		tokenHash, userID, expires)
	return err
}

func (s *Store) UserBySession(ctx context.Context, tokenHash string) (User, error) {
	var u User
	err := s.pool.QueryRow(ctx, `
		SELECT u.id, u.email, u.name, u.role, u.email_verified, u.created_at
		FROM sessions s JOIN users u ON u.id = s.user_id
		WHERE s.token_hash = $1 AND s.expires_at > now()`, tokenHash).
		Scan(&u.ID, &u.Email, &u.Name, &u.Role, &u.EmailVerified, &u.CreatedAt)
	return u, err
}

func (s *Store) DeleteSession(ctx context.Context, tokenHash string) error {
	_, err := s.pool.Exec(ctx, `DELETE FROM sessions WHERE token_hash = $1`, tokenHash)
	return err
}

func (s *Store) ListSubmissions(ctx context.Context) ([]Submission, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT id, name, domain, base_url, contact, note, status, created_at
		FROM submissions ORDER BY id DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Submission
	for rows.Next() {
		var sub Submission
		if err := rows.Scan(&sub.ID, &sub.Name, &sub.Domain, &sub.BaseURL, &sub.Contact, &sub.Note, &sub.Status, &sub.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, sub)
	}
	return out, rows.Err()
}

func (s *Store) UpdateSubmissionStatus(ctx context.Context, id int64, status string) error {
	_, err := s.pool.Exec(ctx, `UPDATE submissions SET status = $2 WHERE id = $1`, id, status)
	return err
}

func (s *Store) RecordDelivery(ctx context.Context, d Delivery) (int64, error) {
	var id int64
	err := s.pool.QueryRow(ctx, `
		INSERT INTO delivery_log (subscription_id, event, status, ok, duration_ms, response, payload)
		VALUES ($1, $2, $3, $4, $5, $6, $7) RETURNING id`,
		d.SubscriptionID, d.Event, d.Status, d.OK, d.DurationMS, d.Response, d.Payload).Scan(&id)
	return id, err
}

func (s *Store) ListDeliveries(ctx context.Context, subscriptionID string, limit int) ([]Delivery, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT id, subscription_id, event, status, ok, duration_ms, response, created_at
		FROM delivery_log WHERE subscription_id = $1 ORDER BY id DESC LIMIT $2`, subscriptionID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Delivery
	for rows.Next() {
		var d Delivery
		if err := rows.Scan(&d.ID, &d.SubscriptionID, &d.Event, &d.Status, &d.OK, &d.DurationMS, &d.Response, &d.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, d)
	}
	return out, rows.Err()
}

func (s *Store) GetDelivery(ctx context.Context, id int64) (Delivery, error) {
	var d Delivery
	err := s.pool.QueryRow(ctx, `
		SELECT id, subscription_id, event, status, ok, duration_ms, response, payload, created_at
		FROM delivery_log WHERE id = $1`, id).
		Scan(&d.ID, &d.SubscriptionID, &d.Event, &d.Status, &d.OK, &d.DurationMS, &d.Response, &d.Payload, &d.CreatedAt)
	return d, err
}

func (s *Store) ListUsers(ctx context.Context) ([]User, error) {
	rows, err := s.pool.Query(ctx, `SELECT id, email, name, role, created_at FROM users ORDER BY id`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []User
	for rows.Next() {
		var u User
		if err := rows.Scan(&u.ID, &u.Email, &u.Name, &u.Role, &u.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, u)
	}
	return out, rows.Err()
}

func (s *Store) UpdateUserRole(ctx context.Context, id int64, role string) error {
	_, err := s.pool.Exec(ctx, `UPDATE users SET role = $2 WHERE id = $1`, id, role)
	return err
}

func (s *Store) SetUserVerified(ctx context.Context, id int64) error {
	_, err := s.pool.Exec(ctx, `UPDATE users SET email_verified = true WHERE id = $1`, id)
	return err
}

func (s *Store) CreateAuthToken(ctx context.Context, tokenHash string, userID int64, kind string, expires time.Time) error {
	_, err := s.pool.Exec(ctx, `
		INSERT INTO auth_tokens (token_hash, user_id, kind, expires_at) VALUES ($1, $2, $3, $4)`,
		tokenHash, userID, kind, expires)
	return err
}

func (s *Store) ConsumeAuthToken(ctx context.Context, tokenHash, kind string) (int64, error) {
	var userID int64
	err := s.pool.QueryRow(ctx, `
		DELETE FROM auth_tokens
		WHERE token_hash = $1 AND kind = $2 AND expires_at > now()
		RETURNING user_id`, tokenHash, kind).Scan(&userID)
	return userID, err
}

func (s *Store) UpdatePassword(ctx context.Context, id int64, passwordHash string) error {
	_, err := s.pool.Exec(ctx, `UPDATE users SET password_hash = $2 WHERE id = $1`, id, passwordHash)
	return err
}

func (s *Store) DeleteUserSessions(ctx context.Context, userID int64) error {
	_, err := s.pool.Exec(ctx, `DELETE FROM sessions WHERE user_id = $1`, userID)
	return err
}

func (s *Store) CreateFreeKey(ctx context.Context, k FreeKey) (int64, error) {
	var id int64
	err := s.pool.QueryRow(ctx, `
		INSERT INTO free_keys (provider, base_url, api_key, models, note, submitter, status, last_checked_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8) RETURNING id`,
		k.Provider, k.BaseURL, k.Key, k.Models, k.Note, k.Submitter, k.Status, k.LastCheckedAt).Scan(&id)
	return id, err
}

func (s *Store) ListFreeKeys(ctx context.Context, includeInvalid bool, q string) ([]FreeKey, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT id, provider, base_url, api_key, models, note, submitter, status, claims, created_at, last_checked_at
		FROM free_keys
		WHERE ($1 OR status <> 'invalid')
		  AND ($2 = '' OR provider ILIKE '%'||$2||'%' OR base_url ILIKE '%'||$2||'%' OR models ILIKE '%'||$2||'%' OR submitter ILIKE '%'||$2||'%')
		ORDER BY created_at DESC LIMIT 200`, includeInvalid, strings.TrimSpace(q))
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []FreeKey
	for rows.Next() {
		var k FreeKey
		if err := rows.Scan(&k.ID, &k.Provider, &k.BaseURL, &k.Key, &k.Models, &k.Note, &k.Submitter, &k.Status, &k.Claims, &k.CreatedAt, &k.LastCheckedAt); err != nil {
			return nil, err
		}
		out = append(out, k)
	}
	return out, rows.Err()
}

func (s *Store) GetFreeKey(ctx context.Context, id int64) (FreeKey, error) {
	var k FreeKey
	err := s.pool.QueryRow(ctx, `
		SELECT id, provider, base_url, api_key, models, note, submitter, status, claims, created_at, last_checked_at
		FROM free_keys WHERE id = $1`, id).
		Scan(&k.ID, &k.Provider, &k.BaseURL, &k.Key, &k.Models, &k.Note, &k.Submitter, &k.Status, &k.Claims, &k.CreatedAt, &k.LastCheckedAt)
	return k, err
}

func (s *Store) IncrementFreeKeyClaims(ctx context.Context, id int64) error {
	_, err := s.pool.Exec(ctx, `UPDATE free_keys SET claims = claims + 1 WHERE id = $1`, id)
	return err
}

func (s *Store) DeleteFreeKey(ctx context.Context, id int64) error {
	_, err := s.pool.Exec(ctx, `DELETE FROM free_keys WHERE id = $1`, id)
	return err
}

func (s *Store) RecordStatus(ctx context.Context, st StatusSnapshot) error {
	_, err := s.pool.Exec(ctx, `INSERT INTO component_status (component, status, detail, checked_at) VALUES ($1, $2, $3, $4)`,
		st.Component, st.Status, st.Detail, st.CheckedAt)
	return err
}

func (s *Store) StatusHistory(ctx context.Context, since time.Time) ([]StatusSnapshot, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT component, status, detail, checked_at FROM component_status
		WHERE checked_at > $1 ORDER BY checked_at`, since)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []StatusSnapshot
	for rows.Next() {
		var st StatusSnapshot
		if err := rows.Scan(&st.Component, &st.Status, &st.Detail, &st.CheckedAt); err != nil {
			return nil, err
		}
		out = append(out, st)
	}
	return out, rows.Err()
}

func (s *Store) CreateIncident(ctx context.Context, i Incident) (int64, error) {
	var id int64
	err := s.pool.QueryRow(ctx, `
		INSERT INTO incidents (title, status, component, body, started_at, resolved_at)
		VALUES ($1, $2, $3, $4, $5, $6) RETURNING id`,
		i.Title, i.Status, i.Component, i.Body, i.StartedAt, i.ResolvedAt).Scan(&id)
	return id, err
}

func (s *Store) ListIncidents(ctx context.Context, includeResolved bool) ([]Incident, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT id, title, status, component, body, started_at, resolved_at, created_at
		FROM incidents
		WHERE ($1 OR status <> 'resolved')
		ORDER BY started_at DESC LIMIT 100`, includeResolved)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Incident
	for rows.Next() {
		var i Incident
		if err := rows.Scan(&i.ID, &i.Title, &i.Status, &i.Component, &i.Body, &i.StartedAt, &i.ResolvedAt, &i.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, i)
	}
	return out, rows.Err()
}

func (s *Store) UpdateIncident(ctx context.Context, id int64, status, body string, resolvedAt *time.Time) error {
	_, err := s.pool.Exec(ctx, `UPDATE incidents SET status = $2, body = $3, resolved_at = $4 WHERE id = $1`, id, status, body, resolvedAt)
	return err
}

func (s *Store) DeleteIncident(ctx context.Context, id int64) error {
	_, err := s.pool.Exec(ctx, `DELETE FROM incidents WHERE id = $1`, id)
	return err
}

func (s *Store) RecordProviderStatus(ctx context.Context, p ProviderSnapshot) error {
	_, err := s.pool.Exec(ctx, `INSERT INTO provider_status (provider, status, detail, checked_at) VALUES ($1, $2, $3, $4)`,
		p.Provider, p.Status, p.Detail, p.CheckedAt)
	return err
}

func (s *Store) ProviderStatusHistory(ctx context.Context, since time.Time) ([]ProviderSnapshot, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT provider, status, detail, checked_at FROM provider_status
		WHERE checked_at > $1 ORDER BY checked_at`, since)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []ProviderSnapshot
	for rows.Next() {
		var p ProviderSnapshot
		if err := rows.Scan(&p.Provider, &p.Status, &p.Detail, &p.CheckedAt); err != nil {
			return nil, err
		}
		out = append(out, p)
	}
	return out, rows.Err()
}

func (s *Store) RecordReachability(ctx context.Context, rows []Reachability) error {
	for _, r := range rows {
		_, err := s.pool.Exec(ctx, `
			INSERT INTO station_reachability (station_slug, status, latency_ms, checked_at)
			VALUES ($1, $2, $3, $4)
			ON CONFLICT (station_slug) DO UPDATE SET status = $2, latency_ms = $3, checked_at = $4`,
			r.Slug, r.Status, r.LatencyMS, r.CheckedAt)
		if err != nil {
			return err
		}
		if _, err := s.pool.Exec(ctx, `INSERT INTO station_reachability_history (station_slug, status, latency_ms, checked_at) VALUES ($1, $2, $3, $4)`,
			r.Slug, r.Status, r.LatencyMS, r.CheckedAt); err != nil {
			return err
		}
	}
	return nil
}

func (s *Store) ReachabilityHistory(ctx context.Context, slug string, since time.Time) ([]Reachability, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT station_slug, status, latency_ms, checked_at FROM station_reachability_history
		WHERE station_slug = $1 AND checked_at > $2 ORDER BY checked_at`, slug, since)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Reachability
	for rows.Next() {
		var r Reachability
		if err := rows.Scan(&r.Slug, &r.Status, &r.LatencyMS, &r.CheckedAt); err != nil {
			return nil, err
		}
		out = append(out, r)
	}
	return out, rows.Err()
}

func (s *Store) ListReachability(ctx context.Context) (map[string]Reachability, error) {
	rows, err := s.pool.Query(ctx, `SELECT station_slug, status, latency_ms, checked_at FROM station_reachability`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := map[string]Reachability{}
	for rows.Next() {
		var r Reachability
		if err := rows.Scan(&r.Slug, &r.Status, &r.LatencyMS, &r.CheckedAt); err != nil {
			return nil, err
		}
		out[r.Slug] = r
	}
	return out, rows.Err()
}

func (s *Store) PruneProviderStatus(ctx context.Context, before time.Time) error {
	_, err := s.pool.Exec(ctx, `DELETE FROM provider_status WHERE checked_at < $1`, before)
	return err
}

func (s *Store) PruneComponentStatus(ctx context.Context, before time.Time) error {
	_, err := s.pool.Exec(ctx, `DELETE FROM component_status WHERE checked_at < $1`, before)
	return err
}

func (s *Store) RecordAdminAction(ctx context.Context, a AdminAction) error {
	_, err := s.pool.Exec(ctx, `
		INSERT INTO admin_audit_log (actor_id, actor_email, action, target, detail, ip)
		VALUES ($1, $2, $3, $4, $5, $6)`,
		a.ActorID, a.ActorEmail, a.Action, a.Target, a.Detail, a.IP)
	return err
}

func (s *Store) ListAdminActions(ctx context.Context, limit int) ([]AdminAction, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT id, actor_id, actor_email, action, target, detail, ip, created_at
		FROM admin_audit_log ORDER BY id DESC LIMIT $1`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []AdminAction
	for rows.Next() {
		var a AdminAction
		if err := rows.Scan(&a.ID, &a.ActorID, &a.ActorEmail, &a.Action, &a.Target, &a.Detail, &a.IP, &a.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, a)
	}
	return out, rows.Err()
}

func (s *Store) ListAllReviews(ctx context.Context) ([]AdminReview, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT r.id, s.slug, s.name, r.author, r.rating, r.body, r.created_at
		FROM reviews r JOIN stations s ON s.id = r.station_id
		ORDER BY r.id DESC LIMIT 200`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []AdminReview
	for rows.Next() {
		var r AdminReview
		if err := rows.Scan(&r.ID, &r.StationSlug, &r.StationName, &r.Author, &r.Rating, &r.Body, &r.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, r)
	}
	return out, rows.Err()
}

func (s *Store) DeleteReview(ctx context.Context, id int64) error {
	_, err := s.pool.Exec(ctx, `DELETE FROM reviews WHERE id = $1`, id)
	return err
}

func (s *Store) GetAudit(ctx context.Context, id string) (AuditReport, error) {	var a AuditReport
	err := s.pool.QueryRow(ctx, `
		SELECT id, target_url, model, passed, payload, created_at
		FROM audit_reports WHERE id = $1`, id).
		Scan(&a.ID, &a.TargetURL, &a.Model, &a.Passed, &a.Payload, &a.CreatedAt)
	return a, err
}

func (s *Store) CreateSubmission(ctx context.Context, sub Submission) (Submission, error) {
	err := s.pool.QueryRow(ctx, `
		INSERT INTO submissions (name, domain, base_url, contact, note)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id, name, domain, base_url, contact, note, status, created_at`,
		sub.Name, sub.Domain, sub.BaseURL, sub.Contact, sub.Note).
		Scan(&sub.ID, &sub.Name, &sub.Domain, &sub.BaseURL, &sub.Contact, &sub.Note, &sub.Status, &sub.CreatedAt)
	return sub, err
}

func (s *Store) CreateClaim(ctx context.Context, stationID int64, email, token string) (Claim, error) {
	var c Claim
	err := s.pool.QueryRow(ctx, `
		INSERT INTO claims (station_id, email, token)
		VALUES ($1, $2, $3)
		RETURNING id, station_id, email, token, verified, created_at`,
		stationID, email, token).
		Scan(&c.ID, &c.StationID, &c.Email, &c.Token, &c.Verified, &c.CreatedAt)
	return c, err
}

// VerifyClaim marks the claim as verified and flips the station's claimed flag.
func (s *Store) VerifyClaim(ctx context.Context, token string) (Claim, string, error) {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return Claim{}, "", err
	}
	defer tx.Rollback(ctx)

	var c Claim
	err = tx.QueryRow(ctx, `
		UPDATE claims SET verified = TRUE WHERE token = $1
		RETURNING id, station_id, email, token, verified, created_at`, token).
		Scan(&c.ID, &c.StationID, &c.Email, &c.Token, &c.Verified, &c.CreatedAt)
	if err != nil {
		return Claim{}, "", err
	}
	if _, err := tx.Exec(ctx, `UPDATE stations SET claimed = TRUE WHERE id = $1`, c.StationID); err != nil {
		return Claim{}, "", err
	}
	var slug string
	if err := tx.QueryRow(ctx, `SELECT slug FROM stations WHERE id = $1`, c.StationID).Scan(&slug); err != nil {
		return Claim{}, "", err
	}
	if err := tx.Commit(ctx); err != nil {
		return Claim{}, "", err
	}
	return c, slug, nil
}

func (s *Store) ClaimDomain(ctx context.Context, token string) (string, error) {
	var domain string
	err := s.pool.QueryRow(ctx, `
		SELECT s.domain FROM claims c JOIN stations s ON s.id = c.station_id
		WHERE c.token = $1`, token).Scan(&domain)
	return domain, err
}

func (s *Store) ClaimedStation(ctx context.Context, token string) (ClaimedStation, error) {
	var cs ClaimedStation
	err := s.pool.QueryRow(ctx, `
		SELECT s.slug, s.name, s.tagline, s.refund_policy, s.invoice, s.min_topup_cny,
		       COALESCE((SELECT e.status FROM claim_edits e WHERE e.station_id = s.id ORDER BY e.id DESC LIMIT 1), ''),
		       COALESCE((SELECT e.review_note FROM claim_edits e WHERE e.station_id = s.id ORDER BY e.id DESC LIMIT 1), '')
		FROM claims c JOIN stations s ON s.id = c.station_id
		WHERE c.token = $1 AND c.verified`, token).
		Scan(&cs.Slug, &cs.Name, &cs.Tagline, &cs.RefundPolicy, &cs.Invoice, &cs.MinTopup, &cs.ReviewStatus, &cs.ReviewNote)
	return cs, err
}

func (s *Store) UpdateClaimedStation(ctx context.Context, token, tagline, refundPolicy string, invoice bool, minTopup float64) error {
	tag, err := s.pool.Exec(ctx, `
		UPDATE stations SET tagline = $2, refund_policy = $3, invoice = $4, min_topup_cny = $5
		WHERE id = (SELECT station_id FROM claims WHERE token = $1 AND verified)`,
		token, tagline, refundPolicy, invoice, minTopup)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return errors.New("claim token not verified")
	}
	return nil
}

func (s *Store) Counts(ctx context.Context) (map[string]int, time.Time, error) {
	out := map[string]int{}
	var lastProbe *time.Time
	for key, table := range map[string]string{
		"stations": "stations", "groups": "station_groups", "models": "models", "probes": "probes",
	} {
		var n int
		if err := s.pool.QueryRow(ctx, `SELECT count(*) FROM `+table).Scan(&n); err != nil {
			return nil, time.Time{}, err
		}
		out[key] = n
	}
	if err := s.pool.QueryRow(ctx, `SELECT max(probed_at) FROM probes`).Scan(&lastProbe); err != nil {
		return nil, time.Time{}, err
	}
	t := time.Now().UTC()
	if lastProbe != nil {
		t = *lastProbe
	}
	return out, t, nil
}

func (s *Store) JSONPayload(v any) ([]byte, error) { return json.Marshal(v) }

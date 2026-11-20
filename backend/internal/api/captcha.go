package api

import (
	"bytes"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"image"
	"image/color"
	"image/draw"
	"image/png"
	"math/big"
	"net/http"
	"strconv"
	"strings"
	"time"
)

// Slider captcha for write operations that cost resources or publish secrets
// (audit probes, station submissions, account e-mails, free API keys).
//
// The server draws a random background with a square gap and sends only the
// background and the extracted piece: the gap position never leaves the
// server, so completing the puzzle requires aligning the piece visually. The
// slider is a speed bump against scripts, not a proof of humanity; the older
// per-IP rate limits stay in place.

const (
	captchaWidth     = 260
	captchaHeight    = 100
	captchaPiece     = 34
	captchaTTL       = 3 * time.Minute
	captchaTolerance = 6
	captchaTokenTTL  = 15 * time.Minute
)

type captchaItem struct {
	target  int
	expires time.Time
}

type captchaChallenge struct {
	ID         string `json:"id"`
	Background string `json:"background"`
	Piece      string `json:"piece"`
	Width      int    `json:"width"`
	Height     int    `json:"height"`
	PieceSize  int    `json:"pieceSize"`
	PieceY     int    `json:"pieceY"`
	MaxX       int    `json:"maxX"`
}

func (s *Server) initCaptcha() {
	key := make([]byte, 32)
	if _, err := rand.Read(key); err != nil {
		key = []byte("relayradar-captcha-fallback-key")
	}
	s.captchaKey = key
	s.captcha = map[string]captchaItem{}
}

func (s *Server) handleCaptchaIssue(w http.ResponseWriter, r *http.Request) {
	if !s.cfg.CaptchaEnabled {
		writeJSON(w, http.StatusOK, map[string]any{"disabled": true})
		return
	}
	maxX := captchaWidth - captchaPiece
	target, err := randInt(10, maxX-10)
	if err != nil {
		s.fail(w, http.StatusInternalServerError, err)
		return
	}
	pieceY, err := randInt(8, captchaHeight-captchaPiece-8)
	if err != nil {
		s.fail(w, http.StatusInternalServerError, err)
		return
	}

	bg, piece, err := drawCaptchaImage(target, pieceY)
	if err != nil {
		s.fail(w, http.StatusInternalServerError, err)
		return
	}

	id := randomID()
	s.captchaMu.Lock()
	// Drop expired challenges while we are here.
	now := time.Now()
	for k, v := range s.captcha {
		if now.After(v.expires) {
			delete(s.captcha, k)
		}
	}
	s.captcha[id] = captchaItem{target: target, expires: now.Add(captchaTTL)}
	s.captchaMu.Unlock()

	writeJSON(w, http.StatusOK, captchaChallenge{
		ID: id, Background: bg, Piece: piece,
		Width: captchaWidth, Height: captchaHeight, PieceSize: captchaPiece,
		PieceY: pieceY, MaxX: maxX,
	})
}

func (s *Server) handleCaptchaVerify(w http.ResponseWriter, r *http.Request) {
	if !s.cfg.CaptchaEnabled {
		writeJSON(w, http.StatusOK, map[string]any{"token": "disabled"})
		return
	}
	var body struct {
		X int `json:"x"`
	}
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<12)).Decode(&body); err != nil {
		writeError(w, http.StatusBadRequest, "invalid body")
		return
	}
	id := r.PathValue("id")

	s.captchaMu.Lock()
	item, ok := s.captcha[id]
	if ok {
		delete(s.captcha, id)
	}
	s.captchaMu.Unlock()
	if !ok || time.Now().After(item.expires) {
		writeError(w, http.StatusBadRequest, "验证码已过期，请重试")
		return
	}
	if diff := body.X - item.target; diff > captchaTolerance || diff < -captchaTolerance {
		writeError(w, http.StatusBadRequest, "拼图位置不对，请重试")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"token": s.issueCaptchaToken()})
}

// requireCaptcha returns false (and answers the request) when the caller has no
// valid slider token. Handlers call it before doing any work.
func (s *Server) requireCaptcha(w http.ResponseWriter, r *http.Request) bool {
	if !s.cfg.CaptchaEnabled {
		return true
	}
	if s.validCaptchaToken(r.Header.Get("X-Captcha-Token")) {
		return true
	}
	w.Header().Set("X-Captcha-Required", "1")
	writeError(w, http.StatusPreconditionRequired, "captcha_required")
	return false
}

func (s *Server) issueCaptchaToken() string {
	exp := time.Now().Add(captchaTokenTTL).Unix()
	payload := strconv.FormatInt(exp, 10)
	mac := hmac.New(sha256.New, s.captchaKey)
	mac.Write([]byte(payload))
	sum := mac.Sum(nil)
	return payload + "." + base64.RawURLEncoding.EncodeToString(sum)
}

func (s *Server) validCaptchaToken(token string) bool {
	if token == "" {
		return false
	}
	parts := strings.SplitN(token, ".", 2)
	if len(parts) != 2 {
		return false
	}
	exp, err := strconv.ParseInt(parts[0], 10, 64)
	if err != nil || time.Now().Unix() > exp {
		return false
	}
	mac := hmac.New(sha256.New, s.captchaKey)
	mac.Write([]byte(parts[0]))
	want := base64.RawURLEncoding.EncodeToString(mac.Sum(nil))
	return hmac.Equal([]byte(want), []byte(parts[1]))
}

func randomID() string {
	b := make([]byte, 12)
	if _, err := rand.Read(b); err != nil {
		return strconv.FormatInt(time.Now().UnixNano(), 36)
	}
	return base64.RawURLEncoding.EncodeToString(b)
}

func randInt(min, max int) (int, error) {
	if max <= min {
		return min, nil
	}
	n, err := rand.Int(rand.Reader, big.NewInt(int64(max-min)))
	if err != nil {
		return 0, err
	}
	return min + int(n.Int64()), nil
}

// drawCaptchaImage returns the background (with a gap) and the piece that was
// cut out of it, both PNG data URLs.
func drawCaptchaImage(target, pieceY int) (string, string, error) {
	seed, err := rand.Int(rand.Reader, big.NewInt(1<<31))
	if err != nil {
		return "", "", err
	}
	rng := newRNG(seed.Int64())

	bg := image.NewRGBA(image.Rect(0, 0, captchaWidth, captchaHeight))
	// Soft two-tone background with a few translucent blocks, so the piece has
	// visible features to align.
	draw.Draw(bg, bg.Bounds(), &image.Uniform{color.RGBA{18, 26, 34, 255}}, image.Point{}, draw.Src)
	for i := 0; i < 26; i++ {
		x := rng.intn(captchaWidth)
		y := rng.intn(captchaHeight)
		w := 12 + rng.intn(70)
		h := 8 + rng.intn(34)
		c := color.RGBA{uint8(20 + rng.intn(60)), uint8(80 + rng.intn(100)), uint8(90 + rng.intn(120)), 120}
		draw.Draw(bg, image.Rect(x, y, x+w, y+h), &image.Uniform{c}, image.Point{}, draw.Over)
	}
	for i := 0; i < 6; i++ {
		c := color.RGBA{uint8(200 + rng.intn(55)), uint8(150 + rng.intn(60)), 60, 90}
		x := rng.intn(captchaWidth - 20)
		draw.Draw(bg, image.Rect(x, rng.intn(captchaHeight-3), x+18+rng.intn(30), rng.intn(captchaHeight-3)+3), &image.Uniform{c}, image.Point{}, draw.Over)
	}

	// Extract the piece from the background, then punch the gap.
	piRect := image.Rect(target, pieceY, target+captchaPiece, pieceY+captchaPiece)
	piece := image.NewRGBA(image.Rect(0, 0, captchaPiece, captchaPiece))
	draw.Draw(piece, piece.Bounds(), bg, piRect.Min, draw.Src)

	gap := color.RGBA{8, 12, 16, 235}
	draw.Draw(bg, piRect, &image.Uniform{gap}, image.Point{}, draw.Src)
	// 1px highlight so the hole reads as a hole.
	draw.Draw(bg, image.Rect(piRect.Min.X, piRect.Min.Y, piRect.Max.X, piRect.Min.Y+1), &image.Uniform{color.RGBA{90, 110, 120, 180}}, image.Point{}, draw.Over)

	bgPng, err := encodePNG(bg)
	if err != nil {
		return "", "", err
	}
	piecePng, err := encodePNG(piece)
	if err != nil {
		return "", "", err
	}
	return bgPng, piecePng, nil
}

func encodePNG(img image.Image) (string, error) {
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		return "", err
	}
	return "data:image/png;base64," + base64.StdEncoding.EncodeToString(buf.Bytes()), nil
}

// rng is a tiny deterministic generator so one challenge looks coherent.
type rng struct{ state uint64 }

func newRNG(seed int64) *rng {
	return &rng{state: uint64(seed) | 1}
}

func (r *rng) next() uint64 {
	r.state ^= r.state << 13
	r.state ^= r.state >> 7
	r.state ^= r.state << 17
	return r.state
}

func (r *rng) intn(n int) int {
	if n <= 0 {
		return 0
	}
	return int(r.next() % uint64(n))
}


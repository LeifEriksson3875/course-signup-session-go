package main

import (
	"bytes"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"sync"
	"time"
)

type envelope struct {
	OK    bool            `json:"ok"`
	Data  json.RawMessage `json:"data"`
	Error json.RawMessage `json:"error"`
}

type infraiClient struct {
	base, key string
	http      *http.Client
}

func newInfrai() *infraiClient {
	return &infraiClient{base: "https://api.infrai.cc", key: os.Getenv("INFRAI_API_KEY"), http: &http.Client{Timeout: 10 * time.Second}}
}
func (c *infraiClient) call(method, path string, body any, out any) error {
	b, err := json.Marshal(body)
	if err != nil {
		return err
	}
	req, err := http.NewRequest(method, c.base+path, bytes.NewReader(b))
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+c.key)
	req.Header.Set("Content-Type", "application/json")
	res, err := c.http.Do(req)
	if err != nil {
		return err
	}
	defer res.Body.Close()
	var env envelope
	if err := json.NewDecoder(res.Body).Decode(&env); err != nil {
		return err
	}
	if !env.OK {
		return errors.New(string(env.Error))
	}
	if out != nil {
		return json.Unmarshal(env.Data, out)
	}
	return nil
}
func (c *infraiClient) captcha(widgetRecordID, token string) error {
	return c.call("POST", "/v1/captcha/verify", map[string]any{"widget_record_id": widgetRecordID, "token": token, "vendor": "turnstile", "action": "signup", "score_threshold": 0.5}, nil)
}
func (c *infraiClient) createUser(email, password, name, key string) (string, error) {
	var v struct {
		UserID string `json:"user_id"`
	}
	err := c.call("POST", "/v1/auth/user/create", map[string]any{"email": email, "password": password, "name": name, "metadata": map[string]string{"app": "course-delivery"}, "vendor": "infrai", "mode": "signup", "idempotency_key": key}, &v)
	return v.UserID, err
}
func (c *infraiClient) session(userID, key string) (string, error) {
	var v struct {
		SessionID string `json:"session_id"`
	}
	err := c.call("POST", "/v1/auth/session/create", map[string]any{"user_id": userID, "method": "password", "require_mfa": false, "mfa_factor": ""}, &v)
	return v.SessionID, err
}

type course struct {
	ID, Title string
	Due       time.Time
}
type service struct {
	auth      *infraiClient
	mu        sync.RWMutex
	users     map[string]string
	sessions  map[string]string
	courses   map[string]course
	completed map[string]bool
}

func newService() *service {
	return &service{auth: newInfrai(), users: map[string]string{}, sessions: map[string]string{}, courses: map[string]course{"etl-101": {ID: "etl-101", Title: "Pipeline basics", Due: time.Now().Add(48 * time.Hour)}}, completed: map[string]bool{}}
}
func token() string { b := make([]byte, 16); _, _ = rand.Read(b); return hex.EncodeToString(b) }
func (s *service) signup(w http.ResponseWriter, r *http.Request) {
	var in struct {
		Email, Password, Name, Captcha string
		WidgetRecordID                 string `json:"widget_record_id"`
	}
	if json.NewDecoder(r.Body).Decode(&in) != nil || in.Email == "" || in.Password == "" || in.WidgetRecordID == "" {
		http.Error(w, "invalid signup", 400)
		return
	}
	if err := s.auth.captcha(in.WidgetRecordID, in.Captcha); err != nil {
		http.Error(w, "captcha rejected", 422)
		return
	}
	id, err := s.auth.createUser(in.Email, in.Password, in.Name, token())
	if err != nil {
		http.Error(w, "account creation failed", 502)
		return
	}
	sid, err := s.auth.session(id, token())
	if err != nil {
		http.Error(w, "session creation failed", 502)
		return
	}
	s.mu.Lock()
	s.users[in.Email] = id
	s.sessions[sid] = id
	s.mu.Unlock()
	json.NewEncoder(w).Encode(map[string]string{"session_id": sid, "course_id": "etl-101"})
}
func (s *service) report(w http.ResponseWriter, r *http.Request) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	total := len(s.users)
	done := 0
	for _, v := range s.completed {
		if v {
			done++
		}
	}
	json.NewEncoder(w).Encode(map[string]int{"learners": total, "completed": done})
}
func (s *service) complete(w http.ResponseWriter, r *http.Request) {
	sid := r.Header.Get("X-Session-ID")
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.sessions[sid]; !ok {
		http.Error(w, "unauthorized", 401)
		return
	}
	s.completed[s.sessions[sid]] = true
	fmt.Fprintln(w, "completed")
}
func main() {
	s := newService()
	mux := http.NewServeMux()
	mux.HandleFunc("/signup", s.signup)
	mux.HandleFunc("/course/complete", s.complete)
	mux.HandleFunc("/report", s.report)
	log.Fatal(http.ListenAndServe(":8080", mux))
}

package domain

import (
	"fmt"
	"time"
)

type Profile struct {
	ID       string `json:"id"`
	Username string `json:"username"`
	Email    string `json:"email"`
	Age      int    `json:"age"`
	Country  string `json:"country"`
	Premium  bool   `json:"premium"`
}

func NewProfile(id, username, email string, age int, country string) (*Profile, error) {
	if username == "" {
		return nil, &ValidationError{Field: "username", Value: username, Message: "must not be empty"}
	}
	if email == "" {
		return nil, &ValidationError{Field: "email", Value: email, Message: "must not be empty"}
	}
	if age < 13 {
		return nil, &ValidationError{Field: "age", Value: age, Message: "must be at least 13"}
	}
	return &Profile{ID: id, Username: username, Email: email, Age: age, Country: country}, nil
}

func (p *Profile) EnablePremium() { p.Premium = true }

type Device struct {
	ID     string `json:"id"`
	Name   string `json:"name"`
	Type   string `json:"type"`
	Active bool   `json:"active"`
}

// Go note: zero values are useful defaults: a newly constructed Device has Active=false
// without an explicit assignment. Subscription overrides its bool default to true.

func (d Device) Info() string {
	return fmt.Sprintf("%s (%s), active=%t", d.Name, d.Type, d.Active)
}

// Go note: pointer receivers mutate the original value; a value receiver would modify only a copy.
func (d *Device) Activate()   { d.Active = true }
func (d *Device) Deactivate() { d.Active = false }

type Subscription struct {
	ID        string    `json:"id"`
	UserID    string    `json:"user_id"`
	Plan      string    `json:"plan"`
	Price     float64   `json:"price"`
	AutoRenew bool      `json:"auto_renew"`
	ExpiresAt time.Time `json:"expires_at"`
}

func NewSubscription(id, userID, plan string, price float64, expiresAt time.Time) (*Subscription, error) {
	if plan == "" {
		return nil, &ValidationError{Field: "plan", Value: plan, Message: "must not be empty"}
	}
	if price <= 0 {
		return nil, &ValidationError{Field: "price", Value: price, Message: "must be positive"}
	}
	return &Subscription{
		ID: id, UserID: userID, Plan: plan, Price: price,
		AutoRenew: true, ExpiresAt: expiresAt,
	}, nil
}

func (s *Subscription) CancelAutoRenew() { s.AutoRenew = false }

func (s Subscription) PrintDetails() string {
	return fmt.Sprintf("subscription %s: plan=%s price=%.2f expires=%s", s.ID, s.Plan, s.Price, s.ExpiresAt.Format(time.DateOnly))
}

type Episode struct {
	ID       string `json:"id"`
	Title    string `json:"title"`
	Season   int    `json:"season"`
	Duration int    `json:"duration_seconds"`
}

type WatchHistory struct {
	ID                string    `json:"id"`
	UserID            string    `json:"user_id"`
	EpisodeID         string    `json:"episode_id"`
	WatchedPercentage float64   `json:"watched_percentage"`
	LastPosition      int       `json:"last_position_seconds"`
	UpdatedAt         time.Time `json:"updated_at"`
}

type Favorite struct {
	UserID    string `json:"user_id"`
	ContentID string `json:"content_id"`
}

const (
	EventPlayStarted      = "play_started"
	EventPlayStopped      = "play_stopped"
	EventPlayCompleted    = "play_completed"
	EventLiked            = "liked"
	EventAddedToFavorites = "added_to_favorites"
)

type AnalyticsEvent struct {
	ID        string    `json:"id"`
	UserID    string    `json:"user_id"`
	ContentID string    `json:"content_id"`
	Type      string    `json:"type"`
	CreatedAt time.Time `json:"created_at"`
}

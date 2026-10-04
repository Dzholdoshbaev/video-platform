package httpapi

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"streamingplatform/internal/domain"
	appmiddleware "streamingplatform/internal/http/middleware"
	"streamingplatform/internal/service"
	"streamingplatform/internal/worker"
)

type ProfileApplication interface {
	Create(ctx context.Context, username, email string, age int, country string) (*domain.Profile, error)
	Get(ctx context.Context, id string) (*domain.Profile, error)
}

type SubscriptionApplication interface {
	ActivateSubscription(ctx context.Context, userID, plan string, price float64, expirationDate string) (*domain.Subscription, error)
	Get(ctx context.Context, id string) (*domain.Subscription, error)
}

type FavoriteApplication interface {
	Add(ctx context.Context, userID, contentID string) (*domain.Favorite, error)
	List(ctx context.Context, userID string) ([]domain.Favorite, error)
}

type WatchHistoryApplication interface {
	Save(ctx context.Context, history *domain.WatchHistory) (*domain.WatchHistory, error)
	List(ctx context.Context, userID string) ([]domain.WatchHistory, error)
}

type ModerationApplication interface {
	CheckContentStatus(contentID, status string) error
}

type AnalyticsApplication interface {
	Submit(ctx context.Context, event domain.AnalyticsEvent) error
}

type TranscodingApplication interface {
	TranscodeBatch(ctx context.Context, jobs []worker.TranscodingJob) ([]worker.TranscodingResult, error)
}

type TasteApplication interface {
	Load(ctx context.Context, userID string) ([]string, error)
}

type Dependencies struct {
	Profiles      ProfileApplication
	Subscriptions SubscriptionApplication
	Favorites     FavoriteApplication
	WatchHistory  WatchHistoryApplication
	Moderation    ModerationApplication
	Analytics     AnalyticsApplication
	Transcoding   TranscodingApplication
	Taste         TasteApplication
}

type API struct {
	dependencies Dependencies
	logger       *slog.Logger
	newID        service.IDGenerator
}

func NewAPI(dependencies Dependencies, logger *slog.Logger) *API {
	return &API{dependencies: dependencies, logger: logger, newID: service.NewID}
}

func (a *API) Handler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("POST /users", a.createUser)
	mux.HandleFunc("GET /users/{id}", a.getUser)
	mux.HandleFunc("POST /subscriptions/activate", a.activateSubscription)
	mux.HandleFunc("GET /subscriptions/{id}", a.getSubscription)
	mux.HandleFunc("POST /favorites", a.addFavorite)
	mux.HandleFunc("GET /users/{id}/favorites", a.listFavorites)
	mux.HandleFunc("POST /watch-history", a.saveWatchHistory)
	mux.HandleFunc("GET /users/{id}/watch-history", a.listWatchHistory)
	mux.HandleFunc("POST /moderation/check", a.checkModeration)
	mux.HandleFunc("POST /analytics/events", a.submitAnalytics)
	mux.HandleFunc("POST /transcoding/batch", a.transcodeBatch)
	mux.HandleFunc("GET /recommendations/{userID}", a.recommendations)
	mux.HandleFunc("GET /health", func(w http.ResponseWriter, _ *http.Request) {
		writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
	})

	// Request ID is outermost so both logging and recovery can read it. Logging wraps
	// recovery so a recovered panic is recorded with status 500.
	return appmiddleware.RequestIDMiddleware(
		appmiddleware.LoggingMiddleware(a.logger,
			appmiddleware.RecoveryMiddleware(a.logger, mux),
		),
	)
}

func (a *API) createUser(w http.ResponseWriter, r *http.Request) {
	var request struct {
		Username string `json:"username"`
		Email    string `json:"email"`
		Age      int    `json:"age"`
		Country  string `json:"country"`
	}
	if err := decodeJSON(w, r, &request); err != nil {
		a.writeError(w, r, err)
		return
	}
	profile, err := a.dependencies.Profiles.Create(r.Context(), request.Username, request.Email, request.Age, request.Country)
	if err != nil {
		a.writeError(w, r, err)
		return
	}
	writeJSON(w, http.StatusCreated, profile)
}

func (a *API) getUser(w http.ResponseWriter, r *http.Request) {
	profile, err := a.dependencies.Profiles.Get(r.Context(), r.PathValue("id"))
	if err != nil {
		a.writeError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, profile)
}

func (a *API) activateSubscription(w http.ResponseWriter, r *http.Request) {
	var request struct {
		UserID         string  `json:"user_id"`
		Plan           string  `json:"plan"`
		Price          float64 `json:"price"`
		ExpirationDate string  `json:"expiration_date"`
	}
	if err := decodeJSON(w, r, &request); err != nil {
		a.writeError(w, r, err)
		return
	}
	subscription, err := a.dependencies.Subscriptions.ActivateSubscription(
		r.Context(), request.UserID, request.Plan, request.Price, request.ExpirationDate,
	)
	if err != nil {
		a.writeError(w, r, err)
		return
	}
	writeJSON(w, http.StatusCreated, subscription)
}

func (a *API) getSubscription(w http.ResponseWriter, r *http.Request) {
	subscription, err := a.dependencies.Subscriptions.Get(r.Context(), r.PathValue("id"))
	if err != nil {
		a.writeError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, subscription)
}

func (a *API) addFavorite(w http.ResponseWriter, r *http.Request) {
	var request domain.Favorite
	if err := decodeJSON(w, r, &request); err != nil {
		a.writeError(w, r, err)
		return
	}
	favorite, err := a.dependencies.Favorites.Add(r.Context(), request.UserID, request.ContentID)
	if err != nil {
		a.writeError(w, r, err)
		return
	}
	writeJSON(w, http.StatusCreated, favorite)
}

func (a *API) listFavorites(w http.ResponseWriter, r *http.Request) {
	favorites, err := a.dependencies.Favorites.List(r.Context(), r.PathValue("id"))
	if err != nil {
		a.writeError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, favorites)
}

func (a *API) saveWatchHistory(w http.ResponseWriter, r *http.Request) {
	var request domain.WatchHistory
	if err := decodeJSON(w, r, &request); err != nil {
		a.writeError(w, r, err)
		return
	}
	history, err := a.dependencies.WatchHistory.Save(r.Context(), &request)
	if err != nil {
		a.writeError(w, r, err)
		return
	}
	writeJSON(w, http.StatusCreated, history)
}

func (a *API) listWatchHistory(w http.ResponseWriter, r *http.Request) {
	history, err := a.dependencies.WatchHistory.List(r.Context(), r.PathValue("id"))
	if err != nil {
		a.writeError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, history)
}

func (a *API) checkModeration(w http.ResponseWriter, r *http.Request) {
	var request struct {
		ContentID string `json:"content_id"`
		Status    string `json:"status"`
	}
	if err := decodeJSON(w, r, &request); err != nil {
		a.writeError(w, r, err)
		return
	}
	if err := a.dependencies.Moderation.CheckContentStatus(request.ContentID, request.Status); err != nil {
		a.writeError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "approved"})
}

func (a *API) submitAnalytics(w http.ResponseWriter, r *http.Request) {
	var event domain.AnalyticsEvent
	if err := decodeJSON(w, r, &event); err != nil {
		a.writeError(w, r, err)
		return
	}
	if event.UserID == "" || event.ContentID == "" || !validEventType(event.Type) {
		a.writeError(w, r, &domain.ValidationError{Field: "event", Value: event, Message: "user_id, content_id and a valid type are required"})
		return
	}
	event.ID = a.newID()
	event.CreatedAt = time.Now().UTC()
	if err := a.dependencies.Analytics.Submit(r.Context(), event); err != nil {
		a.writeError(w, r, fmt.Errorf("queue analytics event: %w", err))
		return
	}
	writeJSON(w, http.StatusAccepted, event)
}

func validEventType(eventType string) bool {
	switch eventType {
	case domain.EventPlayStarted, domain.EventPlayStopped, domain.EventPlayCompleted, domain.EventLiked, domain.EventAddedToFavorites:
		return true
	default:
		return false
	}
}

func (a *API) transcodeBatch(w http.ResponseWriter, r *http.Request) {
	var request struct {
		Jobs []worker.TranscodingJob `json:"jobs"`
	}
	if err := decodeJSON(w, r, &request); err != nil {
		a.writeError(w, r, err)
		return
	}
	results, err := a.dependencies.Transcoding.TranscodeBatch(r.Context(), request.Jobs)
	if err != nil {
		a.writeError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"results": results})
}

func (a *API) recommendations(w http.ResponseWriter, r *http.Request) {
	// The endpoint owns its deadline; the same derived context reaches the slow source.
	ctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
	defer cancel()
	recommendations, err := a.dependencies.Taste.Load(ctx, r.PathValue("userID"))
	if err != nil {
		a.writeError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"recommendations": recommendations})
}

func (a *API) writeError(w http.ResponseWriter, r *http.Request, err error) {
	response := errorResponse{Error: "internal_server_error", Message: "internal server error"}
	status := http.StatusInternalServerError

	var validationErr *domain.ValidationError
	var moderationErr *domain.ModerationError
	switch {
	case errors.Is(err, domain.ErrUserNotFound), errors.Is(err, domain.ErrEpisodeNotFound), errors.Is(err, domain.ErrSubscriptionNotFound):
		status = http.StatusNotFound
		response = errorResponse{Error: "not_found", Message: err.Error()}
	case errors.As(err, &validationErr):
		status = http.StatusBadRequest
		response = errorResponse{Error: "validation_error", Message: validationErr.Message, Field: validationErr.Field}
	case errors.Is(err, domain.ErrConflict):
		status = http.StatusConflict
		response = errorResponse{Error: "conflict", Message: err.Error()}
	case errors.As(err, &moderationErr):
		status = http.StatusUnprocessableEntity
		response = errorResponse{Error: "moderation_rejected", Message: moderationErr.Error()}
	case errors.Is(err, context.DeadlineExceeded):
		status = http.StatusGatewayTimeout
		response = errorResponse{Error: "timeout", Message: "operation timed out"}
	default:
		a.logger.Error("request failed", "request_id", appmiddleware.RequestIDFromContext(r.Context()), "error", err)
	}
	writeJSON(w, status, response)
}

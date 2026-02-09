package api

import (
	"encoding/json"
	"fmt"
	"html/template"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/Kerhoff/TodoboT/internal/models"
	"github.com/Kerhoff/TodoboT/internal/repository"
	"github.com/Kerhoff/TodoboT/internal/service"
	"github.com/sirupsen/logrus"
)

// Server provides the HTTP API and serves the web UI.
type Server struct {
	svc    *service.Service
	logger *logrus.Logger
	mux    *http.ServeMux
}

// NewServer creates a Server, registers all routes, and returns it.
func NewServer(svc *service.Service, logger *logrus.Logger) *Server {
	s := &Server{svc: svc, logger: logger, mux: http.NewServeMux()}
	s.routes()
	return s
}

// Handler returns the http.Handler that can be passed to http.Server.
func (s *Server) Handler() http.Handler {
	return s.mux
}

// ---------------------------------------------------------------------------
// Routes
// ---------------------------------------------------------------------------

func (s *Server) routes() {
	// API – Todos
	s.mux.HandleFunc("GET /api/todos", s.handleGetTodos)
	s.mux.HandleFunc("POST /api/todos", s.handleCreateTodo)
	s.mux.HandleFunc("PUT /api/todos/{id}/done", s.handleCompleteTodo)
	s.mux.HandleFunc("DELETE /api/todos/{id}", s.handleDeleteTodo)

	// API – Calendar events
	s.mux.HandleFunc("GET /api/events", s.handleGetEvents)
	s.mux.HandleFunc("POST /api/events", s.handleCreateEvent)
	s.mux.HandleFunc("DELETE /api/events/{id}", s.handleDeleteEvent)

	// API – Buying list
	s.mux.HandleFunc("GET /api/buying", s.handleGetBuyingItems)
	s.mux.HandleFunc("POST /api/buying", s.handleAddBuyingItem)
	s.mux.HandleFunc("PUT /api/buying/{id}/bought", s.handleMarkBought)
	s.mux.HandleFunc("DELETE /api/buying/{id}", s.handleDeleteBuyingItem)

	// API – Wish list
	s.mux.HandleFunc("GET /api/wishes", s.handleGetWishes)
	s.mux.HandleFunc("POST /api/wishes", s.handleAddWish)
	s.mux.HandleFunc("PUT /api/wishes/{id}/reserve", s.handleReserveWish)
	s.mux.HandleFunc("DELETE /api/wishes/{id}", s.handleDeleteWish)

	// API – Reminders
	s.mux.HandleFunc("GET /api/reminders", s.handleGetReminders)
	s.mux.HandleFunc("POST /api/reminders", s.handleCreateReminder)
	s.mux.HandleFunc("DELETE /api/reminders/{id}", s.handleDeleteReminder)

	// Static files & web UI
	s.mux.Handle("GET /static/", http.StripPrefix("/static/", http.FileServer(http.Dir("web/static"))))
	s.mux.HandleFunc("GET /", s.handleIndex)
}

// ---------------------------------------------------------------------------
// JSON helpers
// ---------------------------------------------------------------------------

func (s *Server) respondJSON(w http.ResponseWriter, status int, data any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	if data != nil {
		if err := json.NewEncoder(w).Encode(data); err != nil {
			s.logger.WithError(err).Error("failed to encode JSON response")
		}
	}
}

func (s *Server) respondError(w http.ResponseWriter, status int, message string) {
	s.respondJSON(w, status, map[string]string{"error": message})
}

// decodeJSON reads the request body into dst and returns an error message on
// failure.  The caller should return immediately when ok == false.
func (s *Server) decodeJSON(r *http.Request, dst any) (ok bool, errMsg string) {
	if r.Body == nil {
		return false, "request body is empty"
	}
	if err := json.NewDecoder(r.Body).Decode(dst); err != nil {
		return false, fmt.Sprintf("invalid JSON: %v", err)
	}
	return true, ""
}

// pathID extracts the {id} path value and converts it to int64.
func pathID(r *http.Request) (int64, error) {
	raw := r.PathValue("id")
	if raw == "" {
		return 0, fmt.Errorf("missing id in path")
	}
	return strconv.ParseInt(raw, 10, 64)
}

// requireChatID reads the chat_id query parameter.  It writes an error
// response and returns 0 when the parameter is absent or invalid.
func (s *Server) requireChatID(w http.ResponseWriter, r *http.Request) (int64, bool) {
	raw := r.URL.Query().Get("chat_id")
	if raw == "" {
		s.respondError(w, http.StatusBadRequest, "chat_id query parameter is required")
		return 0, false
	}
	id, err := strconv.ParseInt(raw, 10, 64)
	if err != nil {
		s.respondError(w, http.StatusBadRequest, "chat_id must be an integer")
		return 0, false
	}
	return id, true
}

// ---------------------------------------------------------------------------
// Index (web UI)
// ---------------------------------------------------------------------------

func (s *Server) handleIndex(w http.ResponseWriter, r *http.Request) {
	// Only serve the index page for the root path; return 404 for unknown
	// paths so the API is not accidentally shadowed.
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}

	tmpl, err := template.ParseFiles("web/templates/index.html")
	if err != nil {
		s.logger.WithError(err).Error("failed to parse index template")
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := tmpl.Execute(w, nil); err != nil {
		s.logger.WithError(err).Error("failed to execute index template")
	}
}

// ---------------------------------------------------------------------------
// Todos
// ---------------------------------------------------------------------------

type createTodoRequest struct {
	Title        string `json:"title"`
	Description  string `json:"description"`
	Priority     string `json:"priority"`
	Deadline     string `json:"deadline"` // RFC 3339
	CreatedByID  int64  `json:"created_by_id"`
	AssignedToID *int64 `json:"assigned_to_id"`
	ChatID       int64  `json:"chat_id"`
}

func (s *Server) handleGetTodos(w http.ResponseWriter, r *http.Request) {
	chatID, ok := s.requireChatID(w, r)
	if !ok {
		return
	}

	q := r.URL.Query()
	var filters repository.TodoFilters

	if status := q.Get("status"); status != "" {
		st := models.TodoStatus(status)
		filters.Status = &st
	}
	if priority := q.Get("priority"); priority != "" {
		pr := models.TodoPriority(priority)
		filters.Priority = &pr
	}
	if limit := q.Get("limit"); limit != "" {
		if v, err := strconv.Atoi(limit); err == nil {
			filters.Limit = v
		}
	}
	if offset := q.Get("offset"); offset != "" {
		if v, err := strconv.Atoi(offset); err == nil {
			filters.Offset = v
		}
	}

	todos, err := s.svc.Todos.GetByChatID(r.Context(), chatID, filters)
	if err != nil {
		s.logger.WithError(err).Error("failed to get todos")
		s.respondError(w, http.StatusInternalServerError, "failed to get todos")
		return
	}

	s.respondJSON(w, http.StatusOK, todos)
}

func (s *Server) handleCreateTodo(w http.ResponseWriter, r *http.Request) {
	var req createTodoRequest
	if ok, msg := s.decodeJSON(r, &req); !ok {
		s.respondError(w, http.StatusBadRequest, msg)
		return
	}

	if strings.TrimSpace(req.Title) == "" {
		s.respondError(w, http.StatusBadRequest, "title is required")
		return
	}
	if req.ChatID == 0 {
		s.respondError(w, http.StatusBadRequest, "chat_id is required")
		return
	}
	if req.CreatedByID == 0 {
		s.respondError(w, http.StatusBadRequest, "created_by_id is required")
		return
	}

	priority := models.TodoPriorityMedium
	if req.Priority != "" {
		priority = models.TodoPriority(req.Priority)
	}

	todo := &models.Todo{
		Title:        strings.TrimSpace(req.Title),
		Description:  strings.TrimSpace(req.Description),
		Status:       models.TodoStatusPending,
		Priority:     priority,
		CreatedByID:  req.CreatedByID,
		AssignedToID: req.AssignedToID,
		ChatID:       req.ChatID,
	}

	if req.Deadline != "" {
		t, err := time.Parse(time.RFC3339, req.Deadline)
		if err != nil {
			s.respondError(w, http.StatusBadRequest, "deadline must be RFC 3339 format")
			return
		}
		todo.Deadline = &t
	}

	created, err := s.svc.Todos.Create(r.Context(), todo)
	if err != nil {
		s.logger.WithError(err).Error("failed to create todo")
		s.respondError(w, http.StatusInternalServerError, "failed to create todo")
		return
	}

	s.respondJSON(w, http.StatusCreated, created)
}

func (s *Server) handleCompleteTodo(w http.ResponseWriter, r *http.Request) {
	id, err := pathID(r)
	if err != nil {
		s.respondError(w, http.StatusBadRequest, "invalid todo id")
		return
	}

	todo, err := s.svc.Todos.GetByID(r.Context(), id)
	if err != nil {
		s.logger.WithError(err).Error("failed to get todo")
		s.respondError(w, http.StatusNotFound, "todo not found")
		return
	}

	todo.Status = models.TodoStatusCompleted
	now := time.Now()
	todo.UpdatedAt = now

	updated, err := s.svc.Todos.Update(r.Context(), todo)
	if err != nil {
		s.logger.WithError(err).Error("failed to complete todo")
		s.respondError(w, http.StatusInternalServerError, "failed to complete todo")
		return
	}

	s.respondJSON(w, http.StatusOK, updated)
}

func (s *Server) handleDeleteTodo(w http.ResponseWriter, r *http.Request) {
	id, err := pathID(r)
	if err != nil {
		s.respondError(w, http.StatusBadRequest, "invalid todo id")
		return
	}

	if err := s.svc.Todos.Delete(r.Context(), id); err != nil {
		s.logger.WithError(err).Error("failed to delete todo")
		s.respondError(w, http.StatusInternalServerError, "failed to delete todo")
		return
	}

	s.respondJSON(w, http.StatusNoContent, nil)
}

// ---------------------------------------------------------------------------
// Calendar Events
// ---------------------------------------------------------------------------

type createEventRequest struct {
	Title       string `json:"title"`
	Description string `json:"description"`
	StartTime   string `json:"start_time"` // RFC 3339
	EndTime     string `json:"end_time"`   // RFC 3339, optional
	AllDay      bool   `json:"all_day"`
	Recurring   string `json:"recurring"`
	Location    string `json:"location"`
	CreatedByID int64  `json:"created_by_id"`
	ChatID      int64  `json:"chat_id"`
	FamilyID    int64  `json:"family_id"`
}

func (s *Server) handleGetEvents(w http.ResponseWriter, r *http.Request) {
	chatID, ok := s.requireChatID(w, r)
	if !ok {
		return
	}

	q := r.URL.Query()
	var filters repository.CalendarFilters

	if from := q.Get("from"); from != "" {
		filters.From = &from
	}
	if to := q.Get("to"); to != "" {
		filters.To = &to
	}
	if limit := q.Get("limit"); limit != "" {
		if v, err := strconv.Atoi(limit); err == nil {
			filters.Limit = v
		}
	}

	events, err := s.svc.Calendar.GetByChatID(r.Context(), chatID, filters)
	if err != nil {
		s.logger.WithError(err).Error("failed to get events")
		s.respondError(w, http.StatusInternalServerError, "failed to get events")
		return
	}

	s.respondJSON(w, http.StatusOK, events)
}

func (s *Server) handleCreateEvent(w http.ResponseWriter, r *http.Request) {
	var req createEventRequest
	if ok, msg := s.decodeJSON(r, &req); !ok {
		s.respondError(w, http.StatusBadRequest, msg)
		return
	}

	if strings.TrimSpace(req.Title) == "" {
		s.respondError(w, http.StatusBadRequest, "title is required")
		return
	}
	if req.ChatID == 0 {
		s.respondError(w, http.StatusBadRequest, "chat_id is required")
		return
	}
	if req.CreatedByID == 0 {
		s.respondError(w, http.StatusBadRequest, "created_by_id is required")
		return
	}
	if req.StartTime == "" {
		s.respondError(w, http.StatusBadRequest, "start_time is required")
		return
	}

	startTime, err := time.Parse(time.RFC3339, req.StartTime)
	if err != nil {
		s.respondError(w, http.StatusBadRequest, "start_time must be RFC 3339 format")
		return
	}

	event := &models.CalendarEvent{
		FamilyID:    req.FamilyID,
		ChatID:      req.ChatID,
		Title:       strings.TrimSpace(req.Title),
		Description: strings.TrimSpace(req.Description),
		StartTime:   startTime,
		AllDay:      req.AllDay,
		Recurring:   req.Recurring,
		Location:    strings.TrimSpace(req.Location),
		CreatedByID: req.CreatedByID,
	}

	if req.EndTime != "" {
		t, err := time.Parse(time.RFC3339, req.EndTime)
		if err != nil {
			s.respondError(w, http.StatusBadRequest, "end_time must be RFC 3339 format")
			return
		}
		event.EndTime = &t
	}

	created, err := s.svc.Calendar.Create(r.Context(), event)
	if err != nil {
		s.logger.WithError(err).Error("failed to create event")
		s.respondError(w, http.StatusInternalServerError, "failed to create event")
		return
	}

	s.respondJSON(w, http.StatusCreated, created)
}

func (s *Server) handleDeleteEvent(w http.ResponseWriter, r *http.Request) {
	id, err := pathID(r)
	if err != nil {
		s.respondError(w, http.StatusBadRequest, "invalid event id")
		return
	}

	if err := s.svc.Calendar.Delete(r.Context(), id); err != nil {
		s.logger.WithError(err).Error("failed to delete event")
		s.respondError(w, http.StatusInternalServerError, "failed to delete event")
		return
	}

	s.respondJSON(w, http.StatusNoContent, nil)
}

// ---------------------------------------------------------------------------
// Buying List
// ---------------------------------------------------------------------------

type addBuyingItemRequest struct {
	Name      string `json:"name"`
	Quantity  string `json:"quantity"`
	AddedByID int64  `json:"added_by_id"`
	ChatID    int64  `json:"chat_id"`
}

type markBoughtRequest struct {
	BoughtByID int64 `json:"bought_by_id"`
}

func (s *Server) handleGetBuyingItems(w http.ResponseWriter, r *http.Request) {
	chatID, ok := s.requireChatID(w, r)
	if !ok {
		return
	}

	onlyUnbought := r.URL.Query().Get("only_unbought") == "true"

	list, err := s.svc.Buying.GetListByChatID(r.Context(), chatID)
	if err != nil {
		// If no list exists yet return an empty array rather than a 500.
		s.logger.WithField("chat_id", chatID).Debug("no buying list found, returning empty")
		s.respondJSON(w, http.StatusOK, []*models.BuyingItem{})
		return
	}

	items, err := s.svc.Buying.GetItems(r.Context(), list.ID, onlyUnbought)
	if err != nil {
		s.logger.WithError(err).Error("failed to get buying items")
		s.respondError(w, http.StatusInternalServerError, "failed to get buying items")
		return
	}

	s.respondJSON(w, http.StatusOK, items)
}

func (s *Server) handleAddBuyingItem(w http.ResponseWriter, r *http.Request) {
	var req addBuyingItemRequest
	if ok, msg := s.decodeJSON(r, &req); !ok {
		s.respondError(w, http.StatusBadRequest, msg)
		return
	}

	if strings.TrimSpace(req.Name) == "" {
		s.respondError(w, http.StatusBadRequest, "name is required")
		return
	}
	if req.ChatID == 0 {
		s.respondError(w, http.StatusBadRequest, "chat_id is required")
		return
	}
	if req.AddedByID == 0 {
		s.respondError(w, http.StatusBadRequest, "added_by_id is required")
		return
	}

	// Ensure there is a buying list for this chat.  If one does not exist we
	// create it on the fly.
	list, err := s.svc.Buying.GetListByChatID(r.Context(), req.ChatID)
	if err != nil {
		// No list yet -- create one.
		list, err = s.svc.Buying.CreateList(r.Context(), &models.BuyingList{
			ChatID:      req.ChatID,
			Name:        "Shopping List",
			CreatedByID: req.AddedByID,
		})
		if err != nil {
			s.logger.WithError(err).Error("failed to create buying list")
			s.respondError(w, http.StatusInternalServerError, "failed to create buying list")
			return
		}
	}

	item := &models.BuyingItem{
		BuyingListID: list.ID,
		Name:         strings.TrimSpace(req.Name),
		Quantity:     strings.TrimSpace(req.Quantity),
		AddedByID:    req.AddedByID,
	}

	created, err := s.svc.Buying.AddItem(r.Context(), item)
	if err != nil {
		s.logger.WithError(err).Error("failed to add buying item")
		s.respondError(w, http.StatusInternalServerError, "failed to add buying item")
		return
	}

	s.respondJSON(w, http.StatusCreated, created)
}

func (s *Server) handleMarkBought(w http.ResponseWriter, r *http.Request) {
	id, err := pathID(r)
	if err != nil {
		s.respondError(w, http.StatusBadRequest, "invalid buying item id")
		return
	}

	var req markBoughtRequest
	if ok, msg := s.decodeJSON(r, &req); !ok {
		s.respondError(w, http.StatusBadRequest, msg)
		return
	}
	if req.BoughtByID == 0 {
		s.respondError(w, http.StatusBadRequest, "bought_by_id is required")
		return
	}

	if err := s.svc.Buying.MarkBought(r.Context(), id, req.BoughtByID); err != nil {
		s.logger.WithError(err).Error("failed to mark item as bought")
		s.respondError(w, http.StatusInternalServerError, "failed to mark item as bought")
		return
	}

	s.respondJSON(w, http.StatusOK, map[string]string{"status": "bought"})
}

func (s *Server) handleDeleteBuyingItem(w http.ResponseWriter, r *http.Request) {
	id, err := pathID(r)
	if err != nil {
		s.respondError(w, http.StatusBadRequest, "invalid buying item id")
		return
	}

	if err := s.svc.Buying.DeleteItem(r.Context(), id); err != nil {
		s.logger.WithError(err).Error("failed to delete buying item")
		s.respondError(w, http.StatusInternalServerError, "failed to delete buying item")
		return
	}

	s.respondJSON(w, http.StatusNoContent, nil)
}

// ---------------------------------------------------------------------------
// Wish List
// ---------------------------------------------------------------------------

type addWishRequest struct {
	Name     string `json:"name"`
	URL      string `json:"url"`
	Price    string `json:"price"`
	Notes    string `json:"notes"`
	UserID   int64  `json:"user_id"`
	FamilyID int64  `json:"family_id"`
}

type reserveWishRequest struct {
	ReservedByID int64 `json:"reserved_by_id"`
}

func (s *Server) handleGetWishes(w http.ResponseWriter, r *http.Request) {
	chatID, ok := s.requireChatID(w, r)
	if !ok {
		return
	}

	// The wish list repository is keyed by family_id, so we resolve the
	// family from the chat_id first.
	family, err := s.svc.Families.GetByChatID(r.Context(), chatID)
	if err != nil {
		s.logger.WithField("chat_id", chatID).Debug("no family found for chat, returning empty wish lists")
		s.respondJSON(w, http.StatusOK, []*models.WishList{})
		return
	}

	lists, err := s.svc.WishList.GetListsByFamily(r.Context(), family.ID)
	if err != nil {
		s.logger.WithError(err).Error("failed to get wish lists")
		s.respondError(w, http.StatusInternalServerError, "failed to get wish lists")
		return
	}

	// Populate items for each wish list so the client receives a complete
	// snapshot in a single request.
	for _, list := range lists {
		items, err := s.svc.WishList.GetItems(r.Context(), list.ID)
		if err != nil {
			s.logger.WithError(err).WithField("wish_list_id", list.ID).Error("failed to get wish items")
			continue
		}
		list.Items = make([]models.WishItem, len(items))
		for i, item := range items {
			list.Items[i] = *item
		}
	}

	s.respondJSON(w, http.StatusOK, lists)
}

func (s *Server) handleAddWish(w http.ResponseWriter, r *http.Request) {
	var req addWishRequest
	if ok, msg := s.decodeJSON(r, &req); !ok {
		s.respondError(w, http.StatusBadRequest, msg)
		return
	}

	if strings.TrimSpace(req.Name) == "" {
		s.respondError(w, http.StatusBadRequest, "name is required")
		return
	}
	if req.UserID == 0 {
		s.respondError(w, http.StatusBadRequest, "user_id is required")
		return
	}
	if req.FamilyID == 0 {
		s.respondError(w, http.StatusBadRequest, "family_id is required")
		return
	}

	// Ensure the user has a wish list within the family.  Create one if it
	// does not exist.
	list, err := s.svc.WishList.GetListByUser(r.Context(), req.UserID, req.FamilyID)
	if err != nil {
		// No list yet -- create one.
		list, err = s.svc.WishList.CreateList(r.Context(), &models.WishList{
			FamilyID: req.FamilyID,
			UserID:   req.UserID,
			Name:     "My Wishes",
		})
		if err != nil {
			s.logger.WithError(err).Error("failed to create wish list")
			s.respondError(w, http.StatusInternalServerError, "failed to create wish list")
			return
		}
	}

	item := &models.WishItem{
		WishListID: list.ID,
		Name:       strings.TrimSpace(req.Name),
		URL:        strings.TrimSpace(req.URL),
		Price:      strings.TrimSpace(req.Price),
		Notes:      strings.TrimSpace(req.Notes),
	}

	created, err := s.svc.WishList.AddItem(r.Context(), item)
	if err != nil {
		s.logger.WithError(err).Error("failed to add wish item")
		s.respondError(w, http.StatusInternalServerError, "failed to add wish item")
		return
	}

	s.respondJSON(w, http.StatusCreated, created)
}

func (s *Server) handleReserveWish(w http.ResponseWriter, r *http.Request) {
	id, err := pathID(r)
	if err != nil {
		s.respondError(w, http.StatusBadRequest, "invalid wish item id")
		return
	}

	var req reserveWishRequest
	if ok, msg := s.decodeJSON(r, &req); !ok {
		s.respondError(w, http.StatusBadRequest, msg)
		return
	}
	if req.ReservedByID == 0 {
		s.respondError(w, http.StatusBadRequest, "reserved_by_id is required")
		return
	}

	if err := s.svc.WishList.ReserveItem(r.Context(), id, req.ReservedByID); err != nil {
		s.logger.WithError(err).Error("failed to reserve wish item")
		s.respondError(w, http.StatusInternalServerError, "failed to reserve wish item")
		return
	}

	s.respondJSON(w, http.StatusOK, map[string]string{"status": "reserved"})
}

func (s *Server) handleDeleteWish(w http.ResponseWriter, r *http.Request) {
	id, err := pathID(r)
	if err != nil {
		s.respondError(w, http.StatusBadRequest, "invalid wish item id")
		return
	}

	if err := s.svc.WishList.DeleteItem(r.Context(), id); err != nil {
		s.logger.WithError(err).Error("failed to delete wish item")
		s.respondError(w, http.StatusInternalServerError, "failed to delete wish item")
		return
	}

	s.respondJSON(w, http.StatusNoContent, nil)
}

// ---------------------------------------------------------------------------
// Reminders
// ---------------------------------------------------------------------------

type createReminderRequest struct {
	Text     string `json:"text"`
	RemindAt string `json:"remind_at"` // RFC 3339
	Repeat   string `json:"repeat"`    // none, daily, weekly, monthly
	UserID   int64  `json:"user_id"`
	ChatID   int64  `json:"chat_id"`
	FamilyID int64  `json:"family_id"`
}

func (s *Server) handleGetReminders(w http.ResponseWriter, r *http.Request) {
	chatID, ok := s.requireChatID(w, r)
	if !ok {
		return
	}

	reminders, err := s.svc.Reminders.GetByChatID(r.Context(), chatID)
	if err != nil {
		s.logger.WithError(err).Error("failed to get reminders")
		s.respondError(w, http.StatusInternalServerError, "failed to get reminders")
		return
	}

	s.respondJSON(w, http.StatusOK, reminders)
}

func (s *Server) handleCreateReminder(w http.ResponseWriter, r *http.Request) {
	var req createReminderRequest
	if ok, msg := s.decodeJSON(r, &req); !ok {
		s.respondError(w, http.StatusBadRequest, msg)
		return
	}

	if strings.TrimSpace(req.Text) == "" {
		s.respondError(w, http.StatusBadRequest, "text is required")
		return
	}
	if req.ChatID == 0 {
		s.respondError(w, http.StatusBadRequest, "chat_id is required")
		return
	}
	if req.UserID == 0 {
		s.respondError(w, http.StatusBadRequest, "user_id is required")
		return
	}
	if req.RemindAt == "" {
		s.respondError(w, http.StatusBadRequest, "remind_at is required")
		return
	}

	remindAt, err := time.Parse(time.RFC3339, req.RemindAt)
	if err != nil {
		s.respondError(w, http.StatusBadRequest, "remind_at must be RFC 3339 format")
		return
	}

	repeat := models.ReminderRepeatNone
	if req.Repeat != "" {
		repeat = models.ReminderRepeat(req.Repeat)
	}

	reminder := &models.Reminder{
		FamilyID: req.FamilyID,
		ChatID:   req.ChatID,
		UserID:   req.UserID,
		Text:     strings.TrimSpace(req.Text),
		RemindAt: remindAt,
		Repeat:   repeat,
		Active:   true,
	}

	created, err := s.svc.Reminders.Create(r.Context(), reminder)
	if err != nil {
		s.logger.WithError(err).Error("failed to create reminder")
		s.respondError(w, http.StatusInternalServerError, "failed to create reminder")
		return
	}

	s.respondJSON(w, http.StatusCreated, created)
}

func (s *Server) handleDeleteReminder(w http.ResponseWriter, r *http.Request) {
	id, err := pathID(r)
	if err != nil {
		s.respondError(w, http.StatusBadRequest, "invalid reminder id")
		return
	}

	if err := s.svc.Reminders.Delete(r.Context(), id); err != nil {
		s.logger.WithError(err).Error("failed to delete reminder")
		s.respondError(w, http.StatusInternalServerError, "failed to delete reminder")
		return
	}

	s.respondJSON(w, http.StatusNoContent, nil)
}

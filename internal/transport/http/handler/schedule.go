package handler

import (
	"net/http"
	"sort"
	"strings"

	"booking-service/internal/domain"
	authmw "booking-service/internal/transport/http/middleware"
	"booking-service/internal/transport/http/response"
	"booking-service/internal/usecase"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

type ScheduleHandler struct {
	uc usecase.ScheduleUsecase
}

func NewScheduleHandler(uc usecase.ScheduleUsecase) *ScheduleHandler {
	return &ScheduleHandler{uc: uc}
}

type createScheduleRequest struct {
	DaysOfWeek []int  `json:"daysOfWeek"`
	StartTime  string `json:"startTime"`
	EndTime    string `json:"endTime"`
}

// scheduleDTO matches api.yaml Schedule (create response).
type scheduleDTO struct {
	RoomID     string `json:"roomId"`
	DaysOfWeek []int  `json:"daysOfWeek"`
	StartTime  string `json:"startTime"`
	EndTime    string `json:"endTime"`
}

type createScheduleResponse struct {
	Schedule scheduleDTO `json:"schedule"`
}

func (h *ScheduleHandler) CreateSchedule(w http.ResponseWriter, r *http.Request) {
	user, ok := authmw.UserFromContext(r.Context())
	if !ok {
		response.WriteError(w, domain.NewDomainError(domain.ErrorUnauthorized, "missing auth user"))
		return
	}
	if h.uc == nil {
		response.WriteError(w, domain.NewDomainError(domain.ErrorInternalError, "schedule usecase is not configured"))
		return
	}

	roomID, err := uuid.Parse(chi.URLParam(r, "roomId"))
	if err != nil {
		response.WriteError(w, domain.WrapDomainError(domain.ErrorInvalidRequest, "invalid roomId", err))
		return
	}

	var req createScheduleRequest
	if err := response.DecodeJSON(w, r, &req); err != nil {
		response.WriteError(w, err)
		return
	}

	view, err := h.uc.CreateSchedule(r.Context(), user, roomID, usecase.CreateScheduleInput{
		DaysOfWeek: req.DaysOfWeek,
		StartTime:  domain.TimeOfDay(req.StartTime),
		EndTime:    domain.TimeOfDay(req.EndTime),
	})
	if err != nil {
		response.WriteError(w, err)
		return
	}

	days := make([]int, 0, len(view.Rules))
	for _, rule := range view.Rules {
		days = append(days, rule.DayOfWeek)
	}
	sort.Ints(days)

	startAPI := scheduleTimeToAPI(string(view.Rules[0].StartTime))
	endAPI := scheduleTimeToAPI(string(view.Rules[0].EndTime))

	resp := createScheduleResponse{
		Schedule: scheduleDTO{
			RoomID:     view.RoomID.String(),
			DaysOfWeek: days,
			StartTime:  startAPI,
			EndTime:    endAPI,
		},
	}

	response.WriteJSON(w, http.StatusCreated, resp)
}

// scheduleTimeToAPI returns HH:MM as in api.yaml pattern (from TIME / TimeOfDay strings).
func scheduleTimeToAPI(s string) string {
	s = strings.TrimSpace(s)
	if len(s) >= 8 && strings.Count(s, ":") == 2 {
		return s[:5]
	}
	return s
}

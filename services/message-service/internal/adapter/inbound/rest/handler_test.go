package rest

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/handziurdmytro/messenger/message-service/internal/domain/message"
	"github.com/handziurdmytro/messenger/message-service/internal/usecase"
	contract "github.com/handziurdmytro/messenger/pkg/contracts/message"
)

var errInternal = errors.New("scylla unavailable")

type route struct {
	chatID    message.ChatID
	messageID message.ID
	userID    string
}

func newRoute() route {
	return route{
		chatID:    message.ChatID(uuid.NewString()),
		messageID: message.ID(uuid.NewString()),
		userID:    uuid.NewString(),
	}
}

func (r route) request(method, body string) *http.Request {
	req := httptest.NewRequest(method, "/", strings.NewReader(body))
	req.SetPathValue("chat_id", string(r.chatID))
	req.SetPathValue("message_id", string(r.messageID))
	req.Header.Set("X-User-ID", r.userID)
	return req
}

func TestHandler_EditMessage(t *testing.T) {
	rt := newRoute()
	edited := message.Message{
		ID: rt.messageID, ChatID: rt.chatID, FromUser: rt.userID, Type: message.TypeText,
		Payload: json.RawMessage(`{"text":"edited"}`), CreatedAt: time.Now(),
	}
	input := usecase.EditMessageInput{ChatID: rt.chatID, MessageID: rt.messageID, UserID: rt.userID, Payload: json.RawMessage(`{"text":"edited"}`)}

	tests := []struct {
		name       string
		body       string
		ucErr      error
		callUC     bool
		wantStatus int
	}{
		{name: "success", body: `{"payload":{"text":"edited"}}`, callUC: true, wantStatus: http.StatusOK},
		{name: "not author", body: `{"payload":{"text":"edited"}}`, callUC: true, ucErr: usecase.ErrForbidden, wantStatus: http.StatusForbidden},
		{name: "not found", body: `{"payload":{"text":"edited"}}`, callUC: true, ucErr: usecase.ErrMessageNotFound, wantStatus: http.StatusNotFound},
		{name: "invalid payload", body: `{"payload":{"text":"edited"}}`, callUC: true, ucErr: contract.ErrInvalidPayload, wantStatus: http.StatusBadRequest},
		{name: "deleted", body: `{"payload":{"text":"edited"}}`, callUC: true, ucErr: message.ErrMessageDeleted, wantStatus: http.StatusConflict},
		{name: "not editable", body: `{"payload":{"text":"edited"}}`, callUC: true, ucErr: message.ErrMessageNotEditable, wantStatus: http.StatusUnprocessableEntity},
		{name: "internal", body: `{"payload":{"text":"edited"}}`, callUC: true, ucErr: errInternal, wantStatus: http.StatusInternalServerError},
		{name: "invalid body", body: `{`, wantStatus: http.StatusBadRequest},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			uc := NewMockEditMessageUseCase(t)
			if tt.callUC {
				uc.EXPECT().Execute(mock.Anything, input).Return(edited, tt.ucErr).Once()
			}

			rec := httptest.NewRecorder()
			NewHandler(Services{EditMessage: uc}).EditMessage(rec, rt.request(http.MethodPatch, tt.body))

			require.Equal(t, tt.wantStatus, rec.Code)
			if tt.wantStatus == http.StatusOK {
				require.Contains(t, rec.Body.String(), `"edited"`)
			}
		})
	}
}

func TestHandler_DeleteMessage(t *testing.T) {
	rt := newRoute()

	tests := []struct {
		err        error
		wantStatus int
	}{
		{nil, http.StatusNoContent},
		{usecase.ErrForbidden, http.StatusForbidden},
		{message.ErrMessageNotDeletable, http.StatusUnprocessableEntity},
		{message.ErrMessageDeleted, http.StatusConflict},
	}

	for _, tt := range tests {
		t.Run(http.StatusText(tt.wantStatus), func(t *testing.T) {
			uc := NewMockDeleteMessageUseCase(t)
			uc.EXPECT().Execute(mock.Anything, rt.chatID, rt.messageID, rt.userID).Return(tt.err).Once()

			rec := httptest.NewRecorder()
			NewHandler(Services{DeleteMessage: uc}).DeleteMessage(rec, rt.request(http.MethodDelete, ""))
			require.Equal(t, tt.wantStatus, rec.Code)
		})
	}
}

func TestHandler_ReadReceipts(t *testing.T) {
	rt := newRoute()
	readAt := time.Date(2026, 11, 10, 12, 0, 0, 0, time.UTC)
	marker := usecase.ReadMarker{UserID: rt.userID, MessageID: rt.messageID, ReadAt: readAt}
	markerJSON := `{"user_id":"` + rt.userID + `","message_id":"` + string(rt.messageID) + `","read_at":"2026-11-10T12:00:00Z"}`

	t.Run("mark read", func(t *testing.T) {
		uc := NewMockMarkReadUseCase(t)
		uc.EXPECT().Execute(mock.Anything, rt.chatID, rt.messageID, rt.userID).Return(marker, nil).Once()

		rec := httptest.NewRecorder()
		NewHandler(Services{MarkRead: uc}).MarkRead(rec, rt.request(http.MethodPost, `{"message_id":"`+string(rt.messageID)+`"}`))
		require.Equal(t, http.StatusOK, rec.Code)
		require.JSONEq(t, markerJSON, rec.Body.String())
	})

	t.Run("mark read errors", func(t *testing.T) {
		uc := NewMockMarkReadUseCase(t)
		uc.EXPECT().Execute(mock.Anything, rt.chatID, rt.messageID, rt.userID).Return(usecase.ReadMarker{}, usecase.ErrMessageNotFound).Once()

		h := NewHandler(Services{MarkRead: uc})
		rec := httptest.NewRecorder()
		h.MarkRead(rec, rt.request(http.MethodPost, `{"message_id":"`+string(rt.messageID)+`"}`))
		require.Equal(t, http.StatusNotFound, rec.Code)

		rec = httptest.NewRecorder()
		h.MarkRead(rec, rt.request(http.MethodPost, `{`))
		require.Equal(t, http.StatusBadRequest, rec.Code)
	})

	t.Run("list markers", func(t *testing.T) {
		uc := NewMockReadMarkersUseCase(t)
		uc.EXPECT().Execute(mock.Anything, rt.chatID).Return([]usecase.ReadMarker{marker}, nil).Once()

		rec := httptest.NewRecorder()
		NewHandler(Services{ReadMarkers: uc}).ListReadMarkers(rec, rt.request(http.MethodGet, ""))
		require.Equal(t, http.StatusOK, rec.Code)
		require.JSONEq(t, `[`+markerJSON+`]`, rec.Body.String())
	})

	t.Run("empty markers list", func(t *testing.T) {
		uc := NewMockReadMarkersUseCase(t)
		uc.EXPECT().Execute(mock.Anything, rt.chatID).Return(nil, nil).Once()

		rec := httptest.NewRecorder()
		NewHandler(Services{ReadMarkers: uc}).ListReadMarkers(rec, rt.request(http.MethodGet, ""))
		require.JSONEq(t, `[]`, rec.Body.String())
	})

	t.Run("unread count", func(t *testing.T) {
		uc := NewMockUnreadCountUseCase(t)
		uc.EXPECT().Execute(mock.Anything, rt.chatID, rt.userID).Return(usecase.UnreadCount{Count: 100, Capped: true}, nil).Once()

		rec := httptest.NewRecorder()
		NewHandler(Services{UnreadCount: uc}).UnreadCount(rec, rt.request(http.MethodGet, ""))
		require.Equal(t, http.StatusOK, rec.Code)
		require.JSONEq(t, `{"count":100,"capped":true}`, rec.Body.String())
	})

	t.Run("unread count error", func(t *testing.T) {
		uc := NewMockUnreadCountUseCase(t)
		uc.EXPECT().Execute(mock.Anything, rt.chatID, rt.userID).Return(usecase.UnreadCount{}, usecase.ErrInvalidChatID).Once()

		rec := httptest.NewRecorder()
		NewHandler(Services{UnreadCount: uc}).UnreadCount(rec, rt.request(http.MethodGet, ""))
		require.Equal(t, http.StatusBadRequest, rec.Code)
	})
}

func TestHandler_GetMessageNotFound(t *testing.T) {
	rt := newRoute()
	uc := NewMockGetMessageUseCase(t)
	uc.EXPECT().Execute(mock.Anything, rt.chatID, rt.messageID).Return(message.Message{}, usecase.ErrMessageNotFound).Once()

	rec := httptest.NewRecorder()
	NewHandler(Services{GetMessage: uc}).GetMessage(rec, rt.request(http.MethodGet, ""))
	require.Equal(t, http.StatusNotFound, rec.Code)
}

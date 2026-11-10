package rest

import (
	"bytes"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"net/textproto"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/handziurdmytro/messenger/message-service/internal/domain/message"
	"github.com/handziurdmytro/messenger/message-service/internal/usecase"
)

func TestHandler_GetHistory(t *testing.T) {
	rt := newRoute()
	before := time.Date(2026, 11, 10, 12, 0, 0, 0, time.UTC)
	beforeID := message.ID(uuid.NewString())

	t.Run("passes query", func(t *testing.T) {
		uc := NewMockGetHistoryUseCase(t)
		uc.EXPECT().Execute(mock.Anything, usecase.HistoryQuery{ChatID: rt.chatID, Before: before, BeforeID: beforeID, Limit: 20}).Return([]message.Message{}, nil).Once()

		req := rt.request(http.MethodGet, "")
		req.URL.RawQuery = "before=" + before.Format(time.RFC3339Nano) + "&before_id=" + string(beforeID) + "&limit=20"
		rec := httptest.NewRecorder()
		NewHandler(Services{GetHistory: uc}).GetHistory(rec, req)
		require.Equal(t, http.StatusOK, rec.Code)
	})

	for _, query := range []string{"before=yesterday", "limit=ten"} {
		t.Run("rejects "+query, func(t *testing.T) {
			req := rt.request(http.MethodGet, "")
			req.URL.RawQuery = query
			rec := httptest.NewRecorder()
			NewHandler(Services{}).GetHistory(rec, req)
			require.Equal(t, http.StatusBadRequest, rec.Code)
		})
	}

	t.Run("maps use case errors", func(t *testing.T) {
		uc := NewMockGetHistoryUseCase(t)
		uc.EXPECT().Execute(mock.Anything, mock.Anything).Return(nil, usecase.ErrInvalidHistoryLimit).Once()

		rec := httptest.NewRecorder()
		NewHandler(Services{GetHistory: uc}).GetHistory(rec, rt.request(http.MethodGet, ""))
		require.Equal(t, http.StatusBadRequest, rec.Code)
	})
}

func multipartRequest(t *testing.T, contentType string, content []byte) *http.Request {
	t.Helper()

	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	header := textproto.MIMEHeader{}
	header.Set("Content-Disposition", `form-data; name="file"; filename="a.bin"`)
	header.Set("Content-Type", contentType)
	part, err := writer.CreatePart(header)
	require.NoError(t, err)
	_, err = part.Write(content)
	require.NoError(t, err)
	require.NoError(t, writer.Close())

	req := httptest.NewRequest(http.MethodPost, "/files", &body)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	return req
}

func TestHandler_UploadFile(t *testing.T) {
	t.Run("uploads file", func(t *testing.T) {
		uc := NewMockUploadFileUseCase(t)
		uc.EXPECT().Execute(mock.Anything, mock.MatchedBy(func(in usecase.UploadFileInput) bool {
			_, err := uuid.Parse(in.Key)
			return err == nil && in.Size == 5 && in.ContentType == "text/plain"
		})).Return(nil).Once()

		rec := httptest.NewRecorder()
		NewHandler(Services{UploadFile: uc}).UploadFile(rec, multipartRequest(t, "text/plain", []byte("hello")))
		require.Equal(t, http.StatusCreated, rec.Code)
		require.Contains(t, rec.Body.String(), `"file_id"`)
	})

	t.Run("rejects unsupported MIME", func(t *testing.T) {
		uc := NewMockUploadFileUseCase(t)
		uc.EXPECT().Execute(mock.Anything, mock.Anything).Return(usecase.ErrUnsupportedMIME).Once()

		rec := httptest.NewRecorder()
		NewHandler(Services{UploadFile: uc}).UploadFile(rec, multipartRequest(t, "text/plain", []byte("hello")))
		require.Equal(t, http.StatusBadRequest, rec.Code)
	})

	t.Run("rejects too large file", func(t *testing.T) {
		uc := NewMockUploadFileUseCase(t)
		uc.EXPECT().Execute(mock.Anything, mock.Anything).Return(usecase.ErrFileTooLarge).Once()

		rec := httptest.NewRecorder()
		NewHandler(Services{UploadFile: uc}).UploadFile(rec, multipartRequest(t, "text/plain", []byte("hello")))
		require.Equal(t, http.StatusRequestEntityTooLarge, rec.Code)
	})

	t.Run("rejects missing file", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/files", strings.NewReader("x"))
		rec := httptest.NewRecorder()
		NewHandler(Services{}).UploadFile(rec, req)
		require.Equal(t, http.StatusBadRequest, rec.Code)
	})
}

func TestHandler_DownloadFile(t *testing.T) {
	id := uuid.NewString()

	t.Run("streams file", func(t *testing.T) {
		uc := NewMockDownloadFileUseCase(t)
		uc.EXPECT().Execute(mock.Anything, id).Return(strings.NewReader("data"), nil).Once()

		req := httptest.NewRequest(http.MethodGet, "/files/"+id, nil)
		req.SetPathValue("id", id)
		rec := httptest.NewRecorder()
		NewHandler(Services{DownloadFile: uc}).DownloadFile(rec, req)

		require.Equal(t, http.StatusOK, rec.Code)
		require.Equal(t, "data", rec.Body.String())
		require.Equal(t, "application/octet-stream", rec.Header().Get("Content-Type"))
	})

	t.Run("invalid ID", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/files/bad", nil)
		req.SetPathValue("id", "bad")
		rec := httptest.NewRecorder()
		NewHandler(Services{}).DownloadFile(rec, req)
		require.Equal(t, http.StatusBadRequest, rec.Code)
	})

	t.Run("storage error", func(t *testing.T) {
		uc := NewMockDownloadFileUseCase(t)
		uc.EXPECT().Execute(mock.Anything, id).Return(nil, errInternal).Once()

		req := httptest.NewRequest(http.MethodGet, "/files/"+id, nil)
		req.SetPathValue("id", id)
		rec := httptest.NewRecorder()
		NewHandler(Services{DownloadFile: uc}).DownloadFile(rec, req)
		require.Equal(t, http.StatusInternalServerError, rec.Code)
	})
}

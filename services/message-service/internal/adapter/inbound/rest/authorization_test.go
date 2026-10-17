package rest

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestAuthorizeChat(t *testing.T) {
	const userID = "02928c6b-08f1-40f8-90d6-7cd4931b9d07"
	const chatID = "3364cbd1-db14-45bf-9b78-3930502f3fb3"
	for _, status := range []int{200, 403, 404, 500} {
		t.Run(http.StatusText(status), func(t *testing.T) {
			upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.URL.Path != "/chats/"+chatID || r.Header.Get("X-User-ID") != userID {
					t.Error("incorrect authorization request")
				}
				w.WriteHeader(status)
			}))
			defer upstream.Close()
			called := false
			handler := authorizeChat(upstream.URL, func(w http.ResponseWriter, r *http.Request) { called = true; w.WriteHeader(204) })
			req := httptest.NewRequest("GET", "/", nil)
			req.SetPathValue("chat_id", chatID)
			req.Header.Set("X-User-ID", userID)
			recorder := httptest.NewRecorder()
			handler(recorder, req)
			expected := 503
			if status == 200 {
				expected = 204
			} else if status == 403 || status == 404 {
				expected = 403
			}
			if recorder.Code != expected || called != (status == 200) {
				t.Fatalf("status=%d handlerCalled=%v", recorder.Code, called)
			}
		})
	}
}

func TestAuthorizeChatRejectsMissingIdentity(t *testing.T) {
	handler := authorizeChat("http://invalid", func(http.ResponseWriter, *http.Request) { t.Fatal("unauthorized handler called") })
	recorder := httptest.NewRecorder()
	handler(recorder, httptest.NewRequest("GET", "/", nil))
	if recorder.Code != http.StatusUnauthorized {
		t.Fatal(recorder.Code)
	}
}

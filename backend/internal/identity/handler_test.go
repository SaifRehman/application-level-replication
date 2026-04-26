package identity

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func newTestRouter(t *testing.T) (http.Handler, *MemoryStore) {
	t.Helper()
	store := NewMemoryStore()
	return NewRouter(store), store
}

func doJSON(t *testing.T, h http.Handler, method, path string, body any) *httptest.ResponseRecorder {
	t.Helper()
	var buf bytes.Buffer
	if body != nil {
		require.NoError(t, json.NewEncoder(&buf).Encode(body))
	}
	req := httptest.NewRequest(method, path, &buf)
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	return rr
}

func TestHealthEndpoints(t *testing.T) {
	r, _ := newTestRouter(t)
	for _, p := range []string{"/healthz", "/readyz"} {
		rr := doJSON(t, r, http.MethodGet, p, nil)
		require.Equal(t, http.StatusOK, rr.Code, p)
	}
}

func TestCreateIdentity(t *testing.T) {
	r, store := newTestRouter(t)
	rr := doJSON(t, r, http.MethodPost, "/api/identities", map[string]string{
		"full_name": "Ada Lovelace",
		"email":     "ada@example.com",
	})
	require.Equal(t, http.StatusCreated, rr.Code, rr.Body.String())

	var got Identity
	require.NoError(t, json.NewDecoder(rr.Body).Decode(&got))
	require.NotEqual(t, uuid.Nil, got.ID)

	all, err := store.List(context.Background())
	require.NoError(t, err)
	require.Len(t, all, 1)
}

func TestCreateIdentityValidation(t *testing.T) {
	r, _ := newTestRouter(t)
	rr := doJSON(t, r, http.MethodPost, "/api/identities", map[string]string{
		"email": "ada@example.com",
	})
	require.Equal(t, http.StatusBadRequest, rr.Code)
}

func TestCreateIdentityBadJSON(t *testing.T) {
	r, _ := newTestRouter(t)
	req := httptest.NewRequest(http.MethodPost, "/api/identities", bytes.NewBufferString("{not json"))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)
	require.Equal(t, http.StatusBadRequest, rr.Code)
}

func TestListIdentities(t *testing.T) {
	r, store := newTestRouter(t)
	require.NoError(t, store.Create(context.Background(), &Identity{FullName: "A"}))
	require.NoError(t, store.Create(context.Background(), &Identity{FullName: "B"}))

	rr := doJSON(t, r, http.MethodGet, "/api/identities", nil)
	require.Equal(t, http.StatusOK, rr.Code)

	var got []Identity
	require.NoError(t, json.NewDecoder(rr.Body).Decode(&got))
	require.Len(t, got, 2)
}

func TestGetIdentity(t *testing.T) {
	r, store := newTestRouter(t)
	in := Identity{FullName: "Ada"}
	require.NoError(t, store.Create(context.Background(), &in))

	rr := doJSON(t, r, http.MethodGet, "/api/identities/"+in.ID.String(), nil)
	require.Equal(t, http.StatusOK, rr.Code)

	body, _ := io.ReadAll(rr.Body)
	var got Identity
	require.NoError(t, json.Unmarshal(body, &got))
	require.Equal(t, in.ID, got.ID)
}

func TestGetIdentityNotFound(t *testing.T) {
	r, _ := newTestRouter(t)
	rr := doJSON(t, r, http.MethodGet, "/api/identities/"+uuid.New().String(), nil)
	require.Equal(t, http.StatusNotFound, rr.Code)
}

func TestGetIdentityBadID(t *testing.T) {
	r, _ := newTestRouter(t)
	rr := doJSON(t, r, http.MethodGet, "/api/identities/not-a-uuid", nil)
	require.Equal(t, http.StatusBadRequest, rr.Code)
}

func TestUpdateIdentity(t *testing.T) {
	r, store := newTestRouter(t)
	in := Identity{FullName: "Ada"}
	require.NoError(t, store.Create(context.Background(), &in))

	rr := doJSON(t, r, http.MethodPut, "/api/identities/"+in.ID.String(), map[string]string{
		"full_name": "Ada Byron",
		"email":     "ada@example.com",
	})
	require.Equal(t, http.StatusOK, rr.Code, rr.Body.String())

	got, err := store.Get(context.Background(), in.ID)
	require.NoError(t, err)
	require.Equal(t, "Ada Byron", got.FullName)
}

func TestUpdateIdentityNotFound(t *testing.T) {
	r, _ := newTestRouter(t)
	rr := doJSON(t, r, http.MethodPut, "/api/identities/"+uuid.New().String(), map[string]string{
		"full_name": "Ada",
	})
	require.Equal(t, http.StatusNotFound, rr.Code)
}

func TestUpdateIdentityValidation(t *testing.T) {
	r, store := newTestRouter(t)
	in := Identity{FullName: "Ada"}
	require.NoError(t, store.Create(context.Background(), &in))

	rr := doJSON(t, r, http.MethodPut, "/api/identities/"+in.ID.String(), map[string]string{
		"full_name": "",
	})
	require.Equal(t, http.StatusBadRequest, rr.Code)
}

func TestDeleteIdentity(t *testing.T) {
	r, store := newTestRouter(t)
	in := Identity{FullName: "Ada"}
	require.NoError(t, store.Create(context.Background(), &in))

	rr := doJSON(t, r, http.MethodDelete, "/api/identities/"+in.ID.String(), nil)
	require.Equal(t, http.StatusNoContent, rr.Code)

	_, err := store.Get(context.Background(), in.ID)
	require.ErrorIs(t, err, ErrNotFound)
}

func TestDeleteIdentityNotFound(t *testing.T) {
	r, _ := newTestRouter(t)
	rr := doJSON(t, r, http.MethodDelete, "/api/identities/"+uuid.New().String(), nil)
	require.Equal(t, http.StatusNotFound, rr.Code)
}

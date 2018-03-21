package hub_test

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/manifoldco/heighliner/pkg/registry/hub"
)

func TestClient_Docker(t *testing.T) {
	routes := map[string]http.HandlerFunc{
		"/v2/users/login/":                   loginHandler("manifold", "secret", "token"),
		"/v2/repositories/myrepo/test/tags":  tagsHandler("token"),
		"/v2/repositories/invalid/auth/tags": tagsHandler("invalid-token"),
	}
	srv := newServer(t, routes)
	defer srv.Close()

	t.Run("with an invalid password", func(t *testing.T) {
		cfg := hub.Config{
			URL:      srv.URL,
			Username: "",
			Password: "",
		}

		cl, err := hub.New(cfg)
		if err != nil {
			t.Fatalf("Expected no error creating new client, got %s", err)
		}
		_, err = cl.Tags("myrepo/test", -1)
		if err == nil {
			t.Fatalf("Expected auth error")
		}
	})

	t.Run("with a valid login", func(t *testing.T) {
		cfg := hub.Config{
			URL:      srv.URL,
			Username: "manifold",
			Password: "secret",
		}

		cl, err := hub.New(cfg)
		if err != nil {
			t.Fatalf("Expected no error creating new client, got %s", err)
		}

		t.Run("with an invalid repo", func(t *testing.T) {
			if _, err := cl.Tags("non/existing", -1); err == nil {
				t.Fatalf("Expected auth error")
			}
		})

		t.Run("with an existing repo", func(t *testing.T) {
			tags, err := cl.Tags("myrepo/test", -1)
			if err != nil {
				t.Fatalf("Didn't expect an error, got '%s'", err)
			}
			if len(tags) != 2 {
				t.Fatalf("Expected 2 tags, got %d", len(tags))
			}
		})

		t.Run("with a repo from a different org", func(t *testing.T) {
			if _, err := cl.Tags("invalid/auth", -1); err == nil {
				t.Fatalf("Expected an error")
			}
		})
	})
}

func tagsHandler(token string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		auth, ok := r.Header["Authorization"]
		if !ok {
			http.Error(w, http.StatusText(http.StatusUnauthorized), http.StatusUnauthorized)
			return
		}

		if auth[0] != fmt.Sprintf("Bearer %s", token) {
			http.Error(w, http.StatusText(http.StatusUnauthorized), http.StatusUnauthorized)
			return
		}

		data := map[string]interface{}{
			"results": []map[string]string{
				{
					"name": "latest",
				},
				{
					"name": "v1.2.3",
				},
			},
		}
		bts, err := json.Marshal(data)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.Write(bts)
	}
}

func loginHandler(username, password, token string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		dec := json.NewDecoder(r.Body)

		data := struct {
			Username string
			Password string
		}{}
		if err := dec.Decode(&data); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		if data.Username != username || data.Password != password {
			http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
			return
		}

		rsp := map[string]string{
			"token": token,
		}
		bts, err := json.Marshal(rsp)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.Write(bts)
	}
}

func newServer(t *testing.T, routes map[string]http.HandlerFunc) *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		route, ok := routes[r.URL.Path]
		if !ok {
			http.Error(w, http.StatusText(http.StatusNotFound), http.StatusNotFound)
			return
		}
		route(w, r)
	}))
}

package server

import (
	"embed"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"net/http"
	"strings"
	"time"
)

//go:embed ui/*
var uiFiles embed.FS

type API struct{ Service *Service }

func respond(w http.ResponseWriter, code int, value any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(value)
}
func apiError(w http.ResponseWriter, code int, err error) {
	respond(w, code, map[string]string{"error": err.Error()})
}
func decodeBody(r *http.Request, v any) error {
	r.Body = http.MaxBytesReader(nil, r.Body, 8192)
	d := json.NewDecoder(r.Body)
	d.DisallowUnknownFields()
	if err := d.Decode(v); err != nil {
		return fmt.Errorf("JSON ungültig: %w", err)
	}
	var x any
	if err := d.Decode(&x); !errors.Is(err, io.EOF) {
		return errors.New("nur ein JSON-Objekt erlaubt")
	}
	return nil
}
func (api API) Handler() http.Handler {
	mux := http.NewServeMux()
	static, _ := fs.Sub(uiFiles, "ui")
	staticHandler := http.FileServer(http.FS(static))
	mux.Handle("GET /", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/", "/index.html", "/sw.js", "/pwa-update.js":
			w.Header().Set("Cache-Control", "no-cache, max-age=0, must-revalidate")
		}
		staticHandler.ServeHTTP(w, r)
	}))
	mux.HandleFunc("GET /manifest.webmanifest", func(w http.ResponseWriter, r *http.Request) {
		body, err := fs.ReadFile(static, "manifest.webmanifest")
		if err != nil {
			http.Error(w, "manifest unavailable", http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/manifest+json; charset=utf-8")
		_, _ = w.Write(body)
	})
	mux.HandleFunc("GET /api/state", func(w http.ResponseWriter, r *http.Request) { respond(w, 200, api.Service.Snapshot()) })
	mux.HandleFunc("GET /api/events", func(w http.ResponseWriter, r *http.Request) {
		f, ok := w.(http.Flusher)
		if !ok {
			apiError(w, 500, errors.New("SSE nicht unterstützt"))
			return
		}
		w.Header().Set("Content-Type", "text/event-stream")
		w.Header().Set("Cache-Control", "no-cache, no-transform")
		w.Header().Set("X-Accel-Buffering", "no")
		stream, unsubscribe := api.Service.Subscribe()
		defer unsubscribe()
		keepalive := time.NewTicker(15 * time.Second)
		defer keepalive.Stop()
		for {
			select {
			case ev := <-stream:
				b, _ := json.Marshal(ev)
				if _, err := fmt.Fprintf(w, "event: state\ndata: %s\n\n", b); err != nil {
					return
				}
				f.Flush()
			case <-keepalive.C:
				if _, err := io.WriteString(w, ": ping\n\n"); err != nil {
					return
				}
				f.Flush()
			case <-r.Context().Done():
				return
			}
		}
	})
	mux.HandleFunc("POST /api/timers", func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			Name        string `json:"name"`
			Duration    string `json:"duration"`
			TemplateID  string `json:"template_id"`
			Sound       *bool  `json:"sound"`
			Notify      *bool  `json:"notify"`
			BrowserOnly bool   `json:"browser_only"`
		}
		if err := decodeBody(r, &req); err != nil {
			apiError(w, 400, err)
			return
		}
		if req.BrowserOnly {
			apiError(w, 400, errors.New("Browser-Timer müssen lokal im Browser gestartet werden"))
			return
		}
		var timer *TimerView
		var err error
		if req.TemplateID != "" {
			timer, err = api.Service.StartTemplate(req.TemplateID)
		} else {
			var duration time.Duration
			duration, err = parseDuration(req.Duration)
			if err == nil && len([]rune(req.Name)) > 80 {
				err = errors.New("Name zu lang (maximal 80 Zeichen)")
			}
			if err == nil {
				timer = api.Service.StartConfigured(req.Name, duration, TimerOptions{Sound: enabled(req.Sound), Notify: enabled(req.Notify)})
			}
		}
		if err != nil {
			apiError(w, 400, err)
			return
		}
		respond(w, 201, timer)
	})
	mux.HandleFunc("POST /api/timers/{id}/{action}", func(w http.ResponseWriter, r *http.Request) {
		id, action := r.PathValue("id"), r.PathValue("action")
		if action != "pause" && action != "resume" && action != "stop" && action != "adjust" {
			apiError(w, 404, errors.New("unbekannte Aktion"))
			return
		}
		var delta time.Duration
		if action == "adjust" {
			var req struct {
				Delta string `json:"delta"`
			}
			if err := decodeBody(r, &req); err != nil {
				apiError(w, 400, err)
				return
			}
			raw := strings.TrimSpace(req.Delta)
			sign := 1
			if strings.HasPrefix(raw, "-") {
				sign = -1
				raw = strings.TrimPrefix(raw, "-")
			}
			raw = strings.TrimPrefix(raw, "+")
			var err error
			delta, err = parseDuration(raw)
			if err != nil {
				apiError(w, 400, err)
				return
			}
			delta *= time.Duration(sign)
		}
		v, err := api.Service.Control(id, action, delta)
		if err != nil {
			apiError(w, 400, err)
			return
		}
		respond(w, 200, v)
	})
	mux.HandleFunc("POST /api/templates", func(w http.ResponseWriter, r *http.Request) {
		var req Template
		if err := decodeBody(r, &req); err != nil {
			apiError(w, 400, err)
			return
		}
		item, err := api.Service.AddTemplateConfigured(req)
		if err != nil {
			apiError(w, 400, err)
			return
		}
		respond(w, 201, item)
	})
	mux.HandleFunc("PUT /api/templates/order", func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			IDs []string `json:"ids"`
		}
		if err := decodeBody(r, &req); err != nil {
			apiError(w, 400, err)
			return
		}
		if err := api.Service.ReorderTemplates(req.IDs); err != nil {
			apiError(w, 409, err)
			return
		}
		respond(w, 200, api.Service.Snapshot())
	})
	mux.HandleFunc("PUT /api/groups/order", func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			Groups []string `json:"groups"`
		}
		if err := decodeBody(r, &req); err != nil {
			apiError(w, 400, err)
			return
		}
		if err := api.Service.ReorderGroups(req.Groups); err != nil {
			apiError(w, 409, err)
			return
		}
		respond(w, 200, api.Service.Snapshot())
	})
	mux.HandleFunc("PUT /api/groups/{name}", func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			Name string `json:"name"`
		}
		if err := decodeBody(r, &req); err != nil {
			apiError(w, 400, err)
			return
		}
		if err := api.Service.RenameGroup(r.PathValue("name"), req.Name); err != nil {
			apiError(w, 400, err)
			return
		}
		respond(w, 200, api.Service.Snapshot())
	})
	mux.HandleFunc("PUT /api/templates/{id}", func(w http.ResponseWriter, r *http.Request) {
		var req Template
		if err := decodeBody(r, &req); err != nil {
			apiError(w, 400, err)
			return
		}
		t, err := api.Service.UpdateTemplateConfigured(r.PathValue("id"), req)
		if err != nil {
			apiError(w, 400, err)
			return
		}
		respond(w, 200, t)
	})
	mux.HandleFunc("DELETE /api/templates/{id}", func(w http.ResponseWriter, r *http.Request) {
		if err := api.Service.DeleteTemplate(r.PathValue("id")); err != nil {
			apiError(w, 404, err)
			return
		}
		w.WriteHeader(204)
	})
	return mux
}

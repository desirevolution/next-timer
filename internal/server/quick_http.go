package server

import (
	"encoding/json"
	"errors"
	"io"
	"mime"
	"net/http"
	"strings"
	"unicode/utf8"
)

func quickHTTPError(w http.ResponseWriter, status int, code, message string) {
	respond(w, status, map[string]string{"code": code, "error": message})
}

func readQuickInput(w http.ResponseWriter, r *http.Request) (string, int, string, error) {
	mediaType, params, err := mime.ParseMediaType(r.Header.Get("Content-Type"))
	if err != nil || (mediaType != "text/plain" && mediaType != "application/json") {
		return "", http.StatusUnsupportedMediaType, "UNSUPPORTED_MEDIA_TYPE", errors.New("Content-Type muss text/plain oder application/json sein")
	}
	if charset, exists := params["charset"]; exists && !strings.EqualFold(charset, "utf-8") {
		return "", http.StatusUnsupportedMediaType, "UNSUPPORTED_MEDIA_TYPE", errors.New("Nur UTF-8 wird unterstützt")
	}
	// No unbounded request bodies (including text/plain).
	const limit = 8192
	data, err := io.ReadAll(http.MaxBytesReader(w, r.Body, limit))
	if err != nil {
		var tooLarge *http.MaxBytesError
		if errors.As(err, &tooLarge) {
			return "", http.StatusRequestEntityTooLarge, "INPUT_TOO_LARGE", errors.New("Eingabe zu groß")
		}
		return "", http.StatusUnprocessableEntity, "INVALID_INPUT", errors.New("Eingabe nicht lesbar")
	}
	if !utf8.Valid(data) {
		return "", http.StatusUnprocessableEntity, "INVALID_INPUT", errors.New("Eingabe muss UTF-8 sein")
	}
	if mediaType == "text/plain" {
		return string(data), 0, "", nil
	}
	var input struct {
		Input *string `json:"input"`
	}
	decoder := json.NewDecoder(strings.NewReader(string(data)))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&input); err != nil {
		return "", http.StatusUnprocessableEntity, "INVALID_INPUT", errors.New("JSON muss ein Objekt mit String-Feld 'input' sein")
	}
	var trailing any
	if err := decoder.Decode(&trailing); !errors.Is(err, io.EOF) {
		return "", http.StatusUnprocessableEntity, "INVALID_INPUT", errors.New("Nur ein JSON-Objekt erlaubt")
	}
	if input.Input == nil {
		return "", http.StatusUnprocessableEntity, "INVALID_INPUT", errors.New("JSON muss ein String-Feld 'input' enthalten")
	}
	return *input.Input, 0, "", nil
}

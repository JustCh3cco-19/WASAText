package api

import (
	"encoding/base64"
	"encoding/json"
	"errors"
	"io"
	"net/http"
)

func decodeBase64Payload(value string) ([]byte, error) {
	decoded, err := base64.StdEncoding.DecodeString(value)
	if err != nil {
		return nil, errors.New("invalid base64 payload")
	}
	if len(decoded) > maxBinaryPayload {
		return nil, errors.New("decoded payload too large")
	}
	return decoded, nil
}

func validImagePayload(value string) bool {
	decoded, err := decodeBase64Payload(value)
	if err != nil {
		return false
	}
	return (len(decoded) >= 8 && string(decoded[:8]) == "\x89PNG\r\n\x1a\n") ||
		(len(decoded) >= 3 && decoded[0] == 0xff && decoded[1] == 0xd8 && decoded[2] == 0xff) ||
		(len(decoded) >= 6 && (string(decoded[:6]) == "GIF87a" || string(decoded[:6]) == "GIF89a"))
}

func decodeJSON(w http.ResponseWriter, r *http.Request, dst interface{}) error {
	r.Body = http.MaxBytesReader(w, r.Body, maxJSONPayload)
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(dst); err != nil {
		var tooLarge *http.MaxBytesError
		if errors.As(err, &tooLarge) {
			writeError(w, http.StatusRequestEntityTooLarge, "Payload troppo grande")
		} else {
			writeError(w, http.StatusBadRequest, "Payload JSON non valido")
		}
		return err
	}
	if err := decoder.Decode(&struct{}{}); !errors.Is(err, io.EOF) {
		writeError(w, http.StatusBadRequest, "Il body deve contenere un solo documento JSON")
		if err == nil {
			return errors.New("multiple JSON documents")
		}
		return err
	}
	return nil
}

func readBinaryBody(w http.ResponseWriter, r *http.Request) ([]byte, error) {
	r.Body = http.MaxBytesReader(w, r.Body, maxBinaryPayload)
	body, err := io.ReadAll(r.Body)
	if err != nil {
		var tooLarge *http.MaxBytesError
		if errors.As(err, &tooLarge) {
			writeError(w, http.StatusRequestEntityTooLarge, "Payload troppo grande")
		} else {
			writeError(w, http.StatusBadRequest, "Impossibile leggere il payload")
		}
		return nil, err
	}
	return body, nil
}

func parseMultipart(w http.ResponseWriter, r *http.Request) error {
	if err := r.ParseMultipartForm(maxMultipartMemory); err != nil {
		var tooLarge *http.MaxBytesError
		if errors.As(err, &tooLarge) {
			writeError(w, http.StatusRequestEntityTooLarge, "Payload troppo grande")
		} else {
			writeError(w, http.StatusBadRequest, "Invalid multipart payload")
		}
		return err
	}
	return nil
}

func writeJSON(w http.ResponseWriter, status int, payload interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if payload == nil {
		return
	}
	_ = json.NewEncoder(w).Encode(payload)
}

func writeError(w http.ResponseWriter, status int, message string) {
	if message == "" {
		message = http.StatusText(status)
	}
	writeJSON(w, status, map[string]string{
		"error": message,
	})
}

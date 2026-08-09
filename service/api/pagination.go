package api

import (
	"fmt"
	"net/http"
	"strconv"
)

type pageResponse struct {
	Limit  int `json:"limit"`
	Offset int `json:"offset"`
}

func parsePage(r *http.Request, defaultLimit, maxLimit int) (int, int, error) {
	limit, offset := defaultLimit, 0
	var err error
	if raw := r.URL.Query().Get("limit"); raw != "" {
		limit, err = strconv.Atoi(raw)
		if err != nil {
			return 0, 0, fmt.Errorf("invalid limit")
		}
	}
	if raw := r.URL.Query().Get("offset"); raw != "" {
		offset, err = strconv.Atoi(raw)
		if err != nil {
			return 0, 0, fmt.Errorf("invalid offset")
		}
	}
	if limit < 1 || limit > maxLimit || offset < 0 {
		return 0, 0, fmt.Errorf("invalid pagination")
	}
	return limit, offset, nil
}

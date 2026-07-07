package main

import (
	"encoding/json"
	"net/http"
	"time"
)

func getJson(url string, target interface{}) error {
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return err
	}

	req.Header.Add("Content-Type", "application/json")
	req.Header.Add("X-Golbat-Secret", config.Golbat.ApiPassword)

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)

	if err != nil {
		return err
	}
	defer resp.Body.Close()

	// Golbat signals a missing record with a 404. Older versions returned an
	// empty body; newer versions return a JSON error body such as
	// {"title":"Not Found","status":404,"detail":"pokemon not found"}. In both
	// cases we must not decode into target, so callers see a nil record and
	// return a 404 to the client instead of treating the error as a result.
	if resp.StatusCode == http.StatusNotFound {
		return nil
	}
	return json.NewDecoder(resp.Body).Decode(target)
}

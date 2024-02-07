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
	return json.NewDecoder(resp.Body).Decode(target)
}

package main

import (
	"encoding/json"
	"fmt"
	"net/http"
)

func getLatestReleaseVersion(repo string) (string, error) {
	rlsURL := fmt.Sprintf("https://api.github.com/repos/%s/releases/latest", repo)
	resp, err := http.Get(rlsURL)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	jsonResp := struct {
		TagName string `json:"tag_name"`
	}{}

	decoder := json.NewDecoder(resp.Body)
	if err := decoder.Decode(&jsonResp); err != nil {
		return "", err
	}

	return jsonResp.TagName, nil
}

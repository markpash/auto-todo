package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
)

func getLatestReleaseVersion(ctx context.Context, client http.Client, repo string) (string, error) {
	rlsURL := fmt.Sprintf("https://api.github.com/repos/%s/releases/latest", repo)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, rlsURL, nil)
	if err != nil {
		return "", err
	}

	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("github api responded with status code: %d", resp.StatusCode)
	}

	jsonResp := struct {
		TagName string `json:"tag_name"`
	}{}

	decoder := json.NewDecoder(resp.Body)
	if err := decoder.Decode(&jsonResp); err != nil {
		return "", err
	}

	return jsonResp.TagName, nil
}

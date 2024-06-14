package main

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/joho/godotenv"
)

type Repo struct {
	Name    string `json:"name"`
	HTMLUrl string `json:"html_url"`
}

func getRepos(username, token string) ([]Repo, error) {
	url := fmt.Sprintf("https://api.github.com/users/%s/repos", username)
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}
	req.SetBasicAuth(username, token)

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to fetch repositories: %s", resp.Status)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var repos []Repo
	if err := json.Unmarshal(body, &repos); err != nil {
		return nil, err
	}

	return repos, nil
}

func runGitCommand(args ...string) error {
	cmd := exec.Command("git", args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

func syncRepo(repo Repo, targetDir string) error {
	repoPath := filepath.Join(targetDir, repo.Name)
	if _, err := os.Stat(repoPath); os.IsNotExist(err) {
		fmt.Printf("Cloning repository %s...\n", repo.Name)
		return runGitCommand("clone", repo.HTMLUrl, repoPath)
	} else {
		fmt.Printf("Pulling updates for repository %s...\n", repo.Name)
		cmd := exec.Command("git", "-C", repoPath, "pull")
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		return cmd.Run()
	}
}

func main() {
	err := godotenv.Load()
	if err != nil {
		println("Failed to load .env file: %v\n", err)
	}

	ghUser := os.Getenv("GITHUB_USERNAME")
	ghToken := os.Getenv("GITHUB_TOKEN")
	repoDir := os.Getenv("REPO_DIR")

	if ghUser == "" || ghToken == "" || repoDir == "" {
		log.Fatalf("GitHub username and token must be provided\n")
	}

	repos, err := getRepos(ghUser, ghToken)
	if err != nil {
		log.Fatalf("Failed to fetch repositories: %v\n", err)
	}

	for _, repo := range repos {
		if err := syncRepo(repo, repoDir); err != nil {
			log.Printf("Failed to sync repository %s: %v\n", repo.Name, err)
		}
	}
}

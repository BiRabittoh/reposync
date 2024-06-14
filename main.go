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
	Name        string `json:"name"`
	HTMLUrl     string `json:"html_url"`
	Description string `json:"description"`
}

func getRepos(username, token string) ([]Repo, error) {
	var allRepos []Repo
	page := 1

	for {
		url := fmt.Sprintf("https://api.github.com/users/%s/repos?page=%d&per_page=100", username, page)
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

		if len(repos) == 0 {
			break
		}

		allRepos = append(allRepos, repos...)
		page++
	}

	return allRepos, nil
}

func runGitCommand(args ...string) error {
	cmd := exec.Command("git", args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

func writeDescription(repoPath, description string) error {
	descriptionPath := filepath.Join(repoPath, "description")
	return os.WriteFile(descriptionPath, []byte(description), 0644)
}

func syncRepo(repo Repo, targetDir string) error {
	repoPath := filepath.Join(targetDir, repo.Name+".git")
	if _, err := os.Stat(repoPath); os.IsNotExist(err) {
		fmt.Printf("Cloning repository %s as bare...\n", repo.Name)
		if err := runGitCommand("clone", "--bare", repo.HTMLUrl, repoPath); err != nil {
			return err
		}
	} else {
		fmt.Printf("Fetching updates for bare repository %s...\n", repo.Name)
		cmd := exec.Command("git", "--git-dir", repoPath, "fetch", "--all")
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		if err := cmd.Run(); err != nil {
			return err
		}
	}

	// Write the repository description
	if err := writeDescription(repoPath, repo.Description); err != nil {
		return fmt.Errorf("failed to write description for %s: %v", repo.Name, err)
	}

	return nil
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

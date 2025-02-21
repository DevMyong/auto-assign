package main

import (
	"context"
	"github.com/google/go-github/v45/github"
	"golang.org/x/oauth2"
	"log"
	"math/rand"
	"os"
	"regexp"
	"strconv"
	"strings"
)

func main() {
	ctx := context.Background()

	// Retrieve environment variables.
	token := os.Getenv("GITHUB_TOKEN")
	if token == "" {
		log.Fatal("GITHUB_TOKEN env not set")
	}
	repoFull := os.Getenv("GITHUB_REPOSITORY")
	if repoFull == "" {
		log.Fatal("GITHUB_REPOSITORY env not set")
	}
	parts := strings.Split(repoFull, "/")
	if len(parts) != 2 {
		log.Fatal("GITHUB_REPOSITORY format invalid")
	}
	owner, repo := parts[0], parts[1]

	prNumberStr := os.Getenv("PR_NUMBER")
	if prNumberStr == "" {
		log.Fatal("PR_NUMBER env not set")
	}
	prNumber, err := strconv.Atoi(prNumberStr)
	if err != nil {
		log.Fatalf("Invalid PR_NUMBER: %v", err)
	}

	// Create GitHub client.
	client := newGitHubClient(ctx, token)

	// Retrieve the pull request details.
	pr, err := getPullRequest(ctx, client, owner, repo, prNumber)
	if err != nil {
		log.Fatalf("Failed to get PR #%d: %v", prNumber, err)
	}

	// Process each feature.
	handleTitleBasedLabel(ctx, client, owner, repo, prNumber, pr)
	handleDayLabel(ctx, client, owner, repo, prNumber, pr)
	assignDefaultAssignee(ctx, client, owner, repo, prNumber, pr)
	assignDefaultReviewers(ctx, client, owner, repo, prNumber, pr)
}

// newGitHubClient creates a GitHub client using the provided token.
func newGitHubClient(ctx context.Context, token string) *github.Client {
	ts := oauth2.StaticTokenSource(&oauth2.Token{AccessToken: token})
	return github.NewClient(oauth2.NewClient(ctx, ts))
}

// getPullRequest retrieves the pull request by number.
func getPullRequest(ctx context.Context, client *github.Client, owner, repo string, prNumber int) (*github.PullRequest, error) {
	pr, _, err := client.PullRequests.Get(ctx, owner, repo, prNumber)
	return pr, err
}

// handleTitleBasedLabel adds labels based on the PR title keywords.
func handleTitleBasedLabel(ctx context.Context, client *github.Client, owner, repo string, prNumber int, pr *github.PullRequest) {
	title := pr.GetTitle()
	if !strings.Contains(strings.ToLower(title), ":") {
		log.Fatalf("PR title does not contain a colon: %s", title)
	}

	// Split the title into a prefix and description.
	parts := strings.SplitN(title, ":", 2)
	prefix := strings.ToLower(strings.TrimSpace(parts[0]))

	// if prefix has any brackets, remove them
	re := regexp.MustCompile(`[\(\[\{<].*$`)
	prefix = re.ReplaceAllString(prefix, "")

	labelMap := map[string]string{
		"feat":     "enhancement",
		"fix":      "bug",
		"docs":     "documentation",
		"style":    "style",
		"refactor": "refactor",
		"perf":     "performance",
		"test":     "test",
		"chore":    "chore",
	}
	label, ok := labelMap[prefix]
	if !ok {
		log.Fatalf("No matching label for prefix: %s", prefix)
	}

	for _, l := range pr.Labels {
		if l.GetName() == label {
			log.Printf("PR already has label: %s", label)
			return
		}
	}

	_, _, err := client.Issues.AddLabelsToIssue(ctx, owner, repo, prNumber, []string{label})
	if err != nil {
		log.Printf("Failed to add title-based labels: %v", err)
	} else {
		log.Printf("Added title-based labels: %v", label)
	}
}

// handleDayLabel calculates code change size and adds a D-n label accordingly.
func handleDayLabel(ctx context.Context, client *github.Client, owner, repo string, prNumber int, pr *github.PullRequest) {
	files, _, err := client.PullRequests.ListFiles(ctx, owner, repo, prNumber, nil)
	if err != nil {
		log.Printf("Failed to list changed files: %v", err)
		return
	}

	totalChanges := 0
	for _, file := range files {
		totalChanges += file.GetAdditions() + file.GetDeletions()
	}

	var dayLabel string
	if totalChanges < 200 {
		dayLabel = "D-3"
	} else if totalChanges < 500 {
		dayLabel = "D-5"
	} else {
		dayLabel = "D-7"
	}

	// Only add a D-n label if one doesn't already exist.
	for _, lab := range pr.Labels {
		if strings.HasPrefix(lab.GetName(), "D-") {
			log.Printf("PR already has a D-n label: %s", lab.GetName())
			return
		}
	}

	_, _, err = client.Issues.AddLabelsToIssue(ctx, owner, repo, prNumber, []string{dayLabel})
	if err != nil {
		log.Printf("Failed to add D-n label: %v", err)
	} else {
		log.Printf("Added Day label: %s", dayLabel)
	}
}

// assignDefaultAssignee sets the PR author as the default assignee if none exists.
func assignDefaultAssignee(ctx context.Context, client *github.Client, owner, repo string, prNumber int, pr *github.PullRequest) {
	if len(pr.Assignees) != 0 {
		log.Printf("PR already has assignees")
		return
	}
	author := pr.GetUser().GetLogin()
	_, _, err := client.Issues.AddAssignees(ctx, owner, repo, prNumber, []string{author})
	if err != nil {
		log.Printf("Failed to add default assignee: %v", err)
	} else {
		log.Printf("Default assignee (%s) added", author)
	}
}

// assignDefaultReviewers requests default reviewers based on repository contributors.
func assignDefaultReviewers(ctx context.Context, client *github.Client, owner, repo string, prNumber int, pr *github.PullRequest) {
	if len(pr.RequestedReviewers) != 0 {
		log.Printf("PR already has reviewers")
		return
	}

	opts := &github.ListContributorsOptions{ListOptions: github.ListOptions{PerPage: 100}}
	var contributors []string
	for {
		contributor, resp, err := client.Repositories.ListContributors(ctx, owner, repo, opts)
		if err != nil {
			log.Printf("Failed to list contributors: %v", err)
			break
		}
		for _, c := range contributor {
			if c.GetLogin() == pr.GetUser().GetLogin() {
				continue
			}
			contributors = append(contributors, c.GetLogin())
		}
		if resp.NextPage == 0 {
			break
		}
		opts.Page = resp.NextPage
	}
	if len(contributors) == 0 || len(contributors) == 1 && contributors[0] == pr.GetUser().GetLogin() {
		log.Printf("No contributors found")
		return
	}

	var reviewers []string
	if len(contributors) > 10 {
		rand.Shuffle(len(contributors), func(i, j int) {
			contributors[i], contributors[j] = contributors[j], contributors[i]
		})
		reviewers = contributors[:10]
	} else {
		reviewers = contributors
	}

	reviewersRequest := github.ReviewersRequest{
		Reviewers: reviewers,
	}
	_, _, err := client.PullRequests.RequestReviewers(ctx, owner, repo, prNumber, reviewersRequest)
	if err != nil {
		log.Printf("Failed to add default reviewers: %v", err)
	} else {
		log.Printf("Default reviewers added: %v", reviewers)
	}
}

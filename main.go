package main

import (
	"context"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"text/tabwriter"
	"time"

	"github.com/google/go-github/v64/github"
	"github.com/urfave/cli/v2"
	"golang.org/x/oauth2"
)

type Export struct {
	Commits      []Commit      `json:"commits"`
	PullRequests []PullRequest `json:"pull_requests"`
	Issues       []Issue       `json:"issues"`
	Releases     []Release     `json:"releases"`
	Watch        []Watch       `json:"watch"`
}

type Commit struct {
	Repo    string    `json:"repo"`
	SHA     string    `json:"sha"`
	Message string    `json:"message"`
	Author  string    `json:"author"`
	Date    time.Time `json:"date"`
}

type PullRequest struct {
	Repo   string    `json:"repo"`
	Number int       `json:"number"`
	Title  string    `json:"title"`
	State  string    `json:"state"`
	Author string    `json:"author"`
	Action string    `json:"action"`
	Date   time.Time `json:"date"`
}

type Issue struct {
	Repo   string    `json:"repo"`
	Number int       `json:"number"`
	Title  string    `json:"title"`
	State  string    `json:"state"`
	Author string    `json:"author"`
	Action string    `json:"action"`
	Date   time.Time `json:"date"`
}

type Release struct {
	Repo    string    `json:"repo"`
	TagName string    `json:"tag_name"`
	Name    string    `json:"name"`
	Author  string    `json:"author"`
	Action  string    `json:"action"`
	Date    time.Time `json:"date"`
}

type Watch struct {
	Repo   string    `json:"repo"`
	Author string    `json:"author"`
	Action string    `json:"action"`
	Date   time.Time `json:"date"`
}

var Version = "dev"

func main() {
	app := &cli.App{
		Name:  "github-export",
		Usage: "Export GitHub user activity",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:    "output",
				Aliases: []string{"o"},
				Value:   "github-export.json",
				Usage:   "Output file path",
			},
			&cli.StringFlag{
				Name:     "token",
				Aliases:  []string{"t"},
				Usage:    "Github API access token",
				Required: true,
				EnvVars:  []string{"GITHUB_TOKEN"},
			},
			&cli.StringFlag{
				Name:    "format",
				Aliases: []string{"f"},
				Value:   "",
				Usage:   "Output format (json, csv, txt)",
			},
			&cli.StringFlag{
				Name:    "kind",
				Aliases: []string{"k"},
				Value:   "commits",
				Usage:   "Kind of data to export (commits, pull_requests, issues, releases)",
			},
			&cli.StringFlag{
				Name:    "mode",
				Aliases: []string{"m"},
				Value:   "",
				Usage:   "Use the Github events API",
			},
		},
		Action: run,
	}
	app.Name = "github-exporter"
	app.Usage = "Export your Github activity to a file"
	app.Version = Version

	err := app.Run(os.Args)
	if err != nil {
		fmt.Println("Error:", err)
		os.Exit(1)
	}
}

func run(c *cli.Context) error {
	token := c.String("token")
	outputFile := c.String("output")
	format := c.String("format")
	kind := c.String("kind")

	ctx := context.Background()
	ts := oauth2.StaticTokenSource(
		&oauth2.Token{AccessToken: token},
	)
	tc := oauth2.NewClient(ctx, ts)
	client := github.NewClient(tc)

	var export Export
	var err error
	if c.String("mode") == "events" {
		export, err = fetchGitHubEvents(ctx, client)
		if err != nil {
			return err
		}
	} else {
		export, err = fetchGitHubData(ctx, client, kind)

		if err != nil {
			return err
		}
	}

	outputFile = generateFilePath(outputFile, kind, format)

	switch format {
	case "json":

		err = outputJSON(export, outputFile)
	case "csv":
		err = outputCSV(export, outputFile, kind)
	default:
		err = outputStdOut(export, kind)
	}

	if err != nil {
		return err
	}

	fmt.Printf("Export completed successfully. Output written to %s\n", outputFile)
	return nil
}

func fetchGitHubData(ctx context.Context, client *github.Client, kind string) (Export, error) {
	export := Export{}

	// List user's repositories
	opt := &github.RepositoryListByAuthenticatedUserOptions{
		ListOptions: github.ListOptions{PerPage: 100},
		Affiliation: "owner",
	}
	// Fetch repositories
	repos, _, err := client.Repositories.ListByAuthenticatedUser(ctx, opt)
	if err != nil {
		return export, err
	}

	// Get authenticated user
	user, _, err := client.Users.Get(ctx, "")
	if err != nil {
		return export, err
	}
	username := user.GetLogin()

	for _, repo := range repos {
		opt := &github.CommitsListOptions{
			Author:      username,
			ListOptions: github.ListOptions{PerPage: 100},
		}

		switch kind {
		case "commits":
			// Fetch commits
			commits, _, err := client.Repositories.ListCommits(ctx, *repo.Owner.Login, *repo.Name, opt)
			if err != nil {
				return export, err
			}
			for _, commit := range commits {
				export.Commits = append(export.Commits, Commit{
					Repo:    *repo.Name,
					SHA:     *commit.SHA,
					Message: *commit.Commit.Message,
					Author:  *commit.Commit.Author.Name,
					Date:    commit.Commit.Author.Date.Time,
				})
			}
		case "pull_requests":

			// Fetch pull requests
			prs, _, err := client.PullRequests.List(ctx, *repo.Owner.Login, *repo.Name, nil)
			if err != nil {
				return export, err
			}
			for _, pr := range prs {
				export.PullRequests = append(export.PullRequests, PullRequest{
					Repo:   *repo.Name,
					Number: *pr.Number,
					Title:  *pr.Title,
					State:  *pr.State,
					Author: *pr.User.Login,
					Date:   pr.CreatedAt.Time,
				})
			}
		case "issues":
			// Fetch issues
			issues, _, err := client.Issues.ListByRepo(ctx, *repo.Owner.Login, *repo.Name, nil)
			if err != nil {
				return export, err
			}
			for _, issue := range issues {
				if issue.PullRequestLinks == nil {
					export.Issues = append(export.Issues, Issue{
						Repo:   *repo.Name,
						Number: *issue.Number,
						Title:  *issue.Title,
						State:  *issue.State,
						Author: *issue.User.Login,
						Date:   issue.CreatedAt.Time,
					})
				}
			}

		case "releases":
			// Fetch releases
			releases, _, err := client.Repositories.ListReleases(ctx, *repo.Owner.Login, *repo.Name, nil)
			if err != nil {
				return export, err
			}
			for _, release := range releases {
				export.Releases = append(export.Releases, Release{
					Repo:    *repo.Name,
					TagName: *release.TagName,
					Name:    *release.Name,
					Author:  *release.Author.Login,
					Date:    release.CreatedAt.Time,
				})
			}
		default:
			return export, fmt.Errorf("unsupported kind: %s", kind)
		}
	}
	return export, nil
}

func outputJSON(export Export, outputFile string) error {
	data, err := json.MarshalIndent(export, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(outputFile, data, 0644)
}

func outputCSV(export Export, outputFile string, kind string) error {
	file, err := os.Create(outputFile)
	if err != nil {
		return err
	}
	defer file.Close()

	writer := csv.NewWriter(file)
	defer writer.Flush()

	// Write headers
	headers := []string{"Type", "Repo", "ID", "Title", "State", "Author", "Date"}
	if err := writer.Write(headers); err != nil {
		return err
	}

	switch kind {
	case "commits":
		// Write commits
		for _, commit := range export.Commits {
			row := []string{"Commit", commit.Repo, commit.SHA, commit.Message, "", commit.Author, commit.Date.String()}
			if err := writer.Write(row); err != nil {
				return err
			}
		}
	case "pull_requests":
		// Write pull requests
		for _, pr := range export.PullRequests {
			row := []string{"PullRequest", pr.Repo, fmt.Sprintf("%d", pr.Number), pr.Title, pr.State, pr.Author, pr.Date.String()}
			if err := writer.Write(row); err != nil {
				return err
			}
		}
	case "issues":

		// Write issues
		for _, issue := range export.Issues {
			row := []string{"Issue", issue.Repo, fmt.Sprintf("%d", issue.Number), issue.Title, issue.State, issue.Author, issue.Date.String()}
			if err := writer.Write(row); err != nil {
				return err
			}
		}

	case "releases":
		// Write releases
		for _, release := range export.Releases {
			row := []string{"Release", release.Repo, release.TagName, release.Name, "", release.Author, release.Date.String()}
			if err := writer.Write(row); err != nil {
				return err
			}
		}
	case "watch":
		// Write watch
		for _, watch := range export.Watch {
			row := []string{"Watch", watch.Repo, "", "", "", watch.Action, watch.Date.String()}
			if err := writer.Write(row); err != nil {
				return err
			}
		}
	}

	return nil
}

func outputStdOut(export Export, kind string) error {
	writer := tabwriter.NewWriter(os.Stdout, 0, 0, 1, ' ', tabwriter.Debug)
	defer writer.Flush()

	switch kind {

	case "commits":
		// Write commits
		fmt.Fprintln(writer, "Date\tRepo\tSHA\tAuthor\tMessage")
		for _, commit := range export.Commits {
			fmt.Fprintf(writer, "%s\t%s\t%s\t%s\t%s\n", commit.Date, commit.Repo, commit.SHA, commit.Author, commit.Message)
		}

	case "pull_requests":
		// Write pull requests
		fmt.Fprintln(writer, "Date\tRepo\tNumber\tTitle\tState\tAuthor")
		for _, pr := range export.PullRequests {
			fmt.Fprintf(writer, "%s\t%s\t%d\t%s\t%s\t%s\n", pr.Date, pr.Repo, pr.Number, pr.Title, pr.State, pr.Author)
		}
	case "issues":
		// Write issues
		fmt.Fprintln(writer, "Date\tRepo\tNumber\tTitle\tState\tAuthor")
		for _, issue := range export.Issues {
			fmt.Fprintf(writer, "%s\t%s\t%d\t%s\t%s\t%s\n",
				issue.Date, issue.Repo, issue.Number, issue.Title, issue.State, issue.Author)
		}
	case "releases":
		// Write releases
		fmt.Fprintln(writer, "Date\tRepo\tTag\tName\tAuthor")
		for _, release := range export.Releases {
			fmt.Fprintf(writer, "%s\t%s\t%s\t%s\t%s\n", release.Date, release.Repo, release.TagName, release.Name, release.Author)
		}
	case "watch":
		// Write watch
		fmt.Fprintln(writer, "Date\tRepo\tAction")
		for _, watch := range export.Watch {
			fmt.Fprintf(writer, "%s\t%s\t%s\n", watch.Date, watch.Repo, watch.Action)
		}

	}
	return nil
}

func fetchGitHubEvents(ctx context.Context, client *github.Client) (Export, error) {
	export := Export{}

	user, _, err := client.Users.Get(ctx, "")
	if err != nil {
		return export, err
	}

	opt := &github.ListOptions{PerPage: 100}
	for {
		events, resp, err := client.Activity.ListEventsPerformedByUser(ctx, *user.Login, false, opt)
		if err != nil {
			return export, err
		}

		for _, event := range events {
			if event.GetActor().GetLogin() != *user.Login {
				continue
			}

			payload, err := event.ParsePayload()
			if err != nil {
				continue
			}

			switch event.GetType() {
			case "PushEvent":
				if p, ok := payload.(*github.PushEvent); ok {
					for _, commit := range p.Commits {
						export.Commits = append(export.Commits, Commit{
							Repo:    event.GetRepo().GetName(),
							SHA:     commit.GetSHA(),
							Message: *commit.Message,
							Date:    event.GetCreatedAt().Time,
						})
					}
				}
			case "PullRequestEvent":
				if p, ok := payload.(*github.PullRequestEvent); ok {
					export.PullRequests = append(export.PullRequests, PullRequest{
						Repo:   event.GetRepo().GetName(),
						Number: p.GetPullRequest().GetNumber(),
						Title:  p.GetPullRequest().GetTitle(),
						Action: p.GetAction(),
						Date:   event.GetCreatedAt().Time,
					})
				}
			case "IssuesEvent":
				if p, ok := payload.(*github.IssuesEvent); ok {
					export.Issues = append(export.Issues, Issue{
						Repo:   event.GetRepo().GetName(),
						Number: p.GetIssue().GetNumber(),
						Title:  p.GetIssue().GetTitle(),
						Action: p.GetAction(),
						Date:   event.GetCreatedAt().Time,
					})
				}
			case "ReleaseEvent":
				if p, ok := payload.(*github.ReleaseEvent); ok {
					export.Releases = append(export.Releases, Release{
						Repo:    event.GetRepo().GetName(),
						TagName: p.GetRelease().GetTagName(),
						Name:    p.GetRelease().GetName(),
						Action:  p.GetAction(),
						Date:    event.GetCreatedAt().Time,
					})
				}
			case "WatchEvent":
				if p, ok := payload.(*github.WatchEvent); ok {
					export.Watch = append(export.Watch, Watch{
						Repo:   event.GetRepo().GetName(),
						Action: p.GetAction(),
						Date:   event.GetCreatedAt().Time,
					})
				}
			}

		}

		if resp.NextPage == 0 {
			break
		}
		opt.Page = resp.NextPage
	}

	return export, nil
}

func generateFilePath(filepath, kind, format string) string {
	var filename string
	timeNow := time.Now().Format("20060102")
	switch format {
	case "json":
		filename = fmt.Sprintf("%s-%s-export-%s.%s", "github", kind, timeNow, "json")
	case "csv":
		filename = fmt.Sprintf("%s-%s-export-%s.%s", "github", kind, timeNow, "csv")
	default:
		filename = "stdout"
	}
	outputFile := filepath[:strings.LastIndex(filepath, "/")+1] + filename
	return outputFile
}

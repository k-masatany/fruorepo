package main

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/briandowns/spinner"
	"github.com/google/go-github/github"
	"github.com/manifoldco/promptui"
	"golang.org/x/oauth2"
)

type Fruorepo struct {
	Token      string
	Client     *github.Client
	Repository *github.Repository
}

func (f *Fruorepo) SetTokenFromEnv() {
	f.Token = os.Getenv("GITHUB_TOKEN")
}

func (f *Fruorepo) SetClient() {
	ctx := context.Background()
	ts := oauth2.StaticTokenSource(
		&oauth2.Token{AccessToken: f.Token},
	)
	tc := oauth2.NewClient(ctx, ts)
	f.Client = github.NewClient(tc)
}

// SelectRepository get taraget *github.Repository
func (f *Fruorepo) SelectRepository() {
	fmt.Println("access github.com...")
	s := spinner.New(spinner.CharSets[34], 100*time.Millisecond)
	s.Start()

	f.SetTokenFromEnv()
	f.SetClient()
	// get all pages of results
	opt := &github.RepositoryListOptions{
		ListOptions: github.ListOptions{PerPage: 10},
	}
	var repositories []*github.Repository
	for {
		repos, resp, err := f.Client.Repositories.List(context.Background(), "", opt)
		if err != nil {
			MessageAndDie(err.Error())
		}
		repositories = append(repositories, repos...)
		if resp.NextPage == 0 {
			break
		}
		opt.Page = resp.NextPage
	}

	var items = []string{}
	for _, r := range repositories {
		items = append(items, r.GetFullName())
	}
	s.Stop()

	templates := &promptui.SelectTemplates{
		Label:    "{{ . }}?",
		Active:   "\U0001F449 {{ . | cyan }}",
		Inactive: "  {{ .FullName | cyan }}",
		Selected: "\U00002728 {{ . | cyan }}",
	}
	prompt := promptui.Select{
		Label:     "Select Repository",
		Items:     items,
		Templates: templates,
		Searcher: func(input string, idx int) bool {
			item := items[idx]
			name := strings.ToLower(item)

			if strings.Contains(name, input) {
				return true
			}

			return false
		},
	}

	n, _, err := prompt.Run()

	if err != nil {
		MessageAndDie(fmt.Sprintf("Prompt failed %v\n", err))
	}

	f.Repository = repositories[n]
}

// PrintRepositoryOverview
func (f *Fruorepo) PrintRepositoryOverview() {
	fmt.Println("------------------------------------------------")
	fmt.Println("       Name:", f.Repository.GetName())
	fmt.Println("        URL:", f.Repository.GetURL())
	fmt.Println("Description:", f.Repository.GetDescription())
	fmt.Println("------------------------------------------------")
}

func (f *Fruorepo) FetchLabels() ([]*github.Label, error) {
	var labels []*github.Label

	pagination := &github.ListOptions{
		PerPage: 50,
		Page:    1,
	}

	fmt.Println("Fetching labels...")
	s := spinner.New(spinner.CharSets[34], 100*time.Millisecond)
	s.Start()

	for {
		labelList, resp, err := f.Client.Issues.ListLabels(
			context.Background(),
			f.Repository.GetOwner().GetLogin(),
			f.Repository.GetName(),
			pagination,
		)

		if err != nil {
			fmt.Printf("Failed to FetchLabels.\n")
			return nil, err
		}

		labels = append(labels, labelList...)

		if resp.NextPage == 0 {
			break
		}
		pagination.Page = resp.NextPage
	}
	s.Stop()

	return labels, nil
}

func (f *Fruorepo) DeleteLabel(label *github.Label, opt *Options) error {
	if opt.DryRun {
		fmt.Printf("Deleted Label...Name:'%s', Color:'%s'(dry run)\n", *label.Name, *label.Color)
		return nil
	}

	resp, err := f.Client.Issues.DeleteLabel(
		context.Background(),
		f.Repository.GetOwner().GetLogin(),
		f.Repository.GetName(),
		*label.Name,
	)

	if err != nil {
		fmt.Printf("Failed to DeleteeLabel.\n")
		fmt.Printf("Response: %s\n", resp)
		return err
	}
	fmt.Printf("Deleted Label...Name:'%s', Color:'%s'\n", *label.Name, *label.Color)

	return nil
}

func (f *Fruorepo) CreateLabel(name, color string, opt *Options) error {
	if opt.DryRun {
		fmt.Printf("Created Label...Name:'%s', Color:'%s'(dry run)\n", name, color)
		return nil
	}

	label, resp, err := f.Client.Issues.CreateLabel(
		context.Background(),
		f.Repository.GetOwner().GetLogin(),
		f.Repository.GetName(),
		&github.Label{
			Name:  &name,
			Color: &color,
		},
	)

	if err != nil {
		fmt.Printf("Failed to CreateLabel.\n")
		fmt.Printf("Response: %s\n", resp)
		return err
	}
	fmt.Printf("Created Label...Name:'%s', Color:'%s'\n", *label.Name, *label.Color)

	return nil
}

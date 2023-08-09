package gitea

import (
	"context"
	"fmt"

	"github.com/google/go-github/v53/github"
	"github.com/oklog/ulid/v2"
	"go.flipt.io/cup/pkg/api"
	"go.flipt.io/cup/pkg/source/git"
)

type SCM struct {
	client     *github.Client
	owner      string
	repository string
	actor      string
}

func New(client *github.Client, owner, repository, actor string) *SCM {
	return &SCM{
		client:     client,
		owner:      owner,
		repository: repository,
		actor:      actor,
	}
}

func (s *SCM) Merge(ctx context.Context, id ulid.ULID) error {
	prs, _, err := s.client.PullRequests.List(ctx, s.owner, s.repository, &github.PullRequestListOptions{
		Head: fmt.Sprintf("%s:cup/proposal/%s", s.actor, id),
	})
	if err != nil {
		return fmt.Errorf("merging: %w", err)
	}

	if len(prs) < 0 {
		return fmt.Errorf("proposal %q not found", id)
	}

	res, _, err := s.client.PullRequests.Merge(ctx, s.owner, s.repository, prs[0].GetNumber(), prs[0].GetTitle(), &github.PullRequestOptions{
		MergeMethod: "merge",
	})
	if err != nil {
		return err
	}

	if !res.GetMerged() {
		return fmt.Errorf("proposal %q could not be merged: %q", id, res.GetMessage())
	}

	return nil
}

func (s *SCM) Propose(ctx context.Context, p git.Proposal) (*api.Proposal, error) {
	pr, _, err := s.client.PullRequests.Create(ctx, s.owner, s.repository, &github.NewPullRequest{
		Head:  github.String(p.Head),
		Base:  github.String(p.Base),
		Title: github.String(p.Title),
		Body:  github.String(p.Body),
	})
	if err != nil {
		return nil, err
	}

	return &api.Proposal{
		Source: "github",
		URL:    pr.GetHTMLURL(),
	}, nil
}

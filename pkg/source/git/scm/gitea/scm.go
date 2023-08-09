package gitea

import (
	"context"
	"fmt"

	"code.gitea.io/sdk/gitea"
	"github.com/oklog/ulid/v2"
	"go.flipt.io/cup/pkg/api"
	"go.flipt.io/cup/pkg/source/git"
)

type SCM struct {
	client     *gitea.Client
	owner      string
	repository string
}

func New(client *gitea.Client, owner, repository string) *SCM {
	return &SCM{
		client:     client,
		owner:      owner,
		repository: repository,
	}
}

func (s *SCM) Merge(_ context.Context, id ulid.ULID) error {
	prs, _, err := s.client.ListRepoPullRequests(s.owner, s.repository, gitea.ListPullRequestsOptions{})
	if err != nil {
		return fmt.Errorf("merging: %w", err)
	}

	var (
		pr    *gitea.PullRequest
		found bool
	)
	for _, p := range prs {
		if found = p.Head.Name == fmt.Sprintf("cup/proposal/%s", id); found {
			pr = p
			break
		}
	}

	if !found {
		return fmt.Errorf("proposal %q not found", id)
	}

	ok, _, err := s.client.MergePullRequest(s.owner, s.repository, pr.Index, gitea.MergePullRequestOption{
		Style: gitea.MergeStyleMerge,
	})
	if err != nil {
		return err
	}

	if !ok {
		return fmt.Errorf("proposal %q could not be merged", id)
	}

	return nil
}

func (s *SCM) Propose(_ context.Context, p git.Proposal) (*api.Proposal, error) {
	pr, _, err := s.client.CreatePullRequest(s.owner, s.repository, gitea.CreatePullRequestOption{
		Head:  p.Head,
		Base:  p.Base,
		Title: p.Title,
		Body:  p.Body,
	})
	if err != nil {
		return nil, err
	}

	return &api.Proposal{
		Source: "gitea",
		URL:    pr.HTMLURL,
	}, nil
}

package gitea

import (
	"context"
	"fmt"
	"io/fs"

	"code.gitea.io/sdk/gitea"
	"go.flipt.io/cup"
	"go.flipt.io/cup/internal/source/git"
)

type Source struct {
	source *git.Source

	cli        *gitea.Client
	user       string
	repository string
}

func New(source *git.Source, cli *gitea.Client, user, repository string) (*Source, error) {
	return &Source{source, cli, user, repository}, nil
}

func (s *Source) Get(ctx context.Context) (fs.FS, string, error) {
	return s.source.Get(ctx)
}

func (s *Source) Propose(ctx context.Context, req cup.ProposeRequest) (*cup.Proposal, error) {
	if len(req.Changes) <= 0 {
		return &cup.Proposal{
			Status: "nochange",
		}, nil
	}

	proposal, err := s.source.Propose(ctx, req)
	if err != nil {
		return nil, err
	}

	head := fmt.Sprintf("cup/proposal/%s", *proposal.ID)
	_, _, err = s.cli.CreatePullRequest(s.user, s.repository, gitea.CreatePullRequestOption{
		Head:  head,
		Base:  "main",
		Title: req.Changes[0].Message,
		Body:  "Some PR message here",
	})

	return proposal, err
}

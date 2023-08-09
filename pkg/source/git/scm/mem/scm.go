package mem

import (
	"context"

	"github.com/oklog/ulid/v2"
	"go.flipt.io/cup/pkg/api"
	"go.flipt.io/cup/pkg/source/git"
)

// SCM is an in-memory representation of the git.SCM interface.
// For now it simply stores proposals in a map and it primarily used
// for unit testing.
type SCM struct {
	proposals map[ulid.ULID]git.Proposal
}

// New constructs and configures a new instance of SCM.
func New() *SCM {
	return &SCM{proposals: map[ulid.ULID]git.Proposal{}}
}

// Propose stores the provided proposal in a map and returns a nil error and empty response.
func (s *SCM) Propose(_ context.Context, p git.Proposal) (*api.Result, error) {
	s.proposals[p.ID] = p

	return &api.Result{ID: p.ID}, nil
}

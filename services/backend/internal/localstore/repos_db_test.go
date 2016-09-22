package localstore

import (
	"fmt"
	"reflect"
	"sort"
	"testing"
	"time"

	"context"

	"sourcegraph.com/sourcegraph/sourcegraph/api/sourcegraph"
	"sourcegraph.com/sourcegraph/sourcegraph/pkg/store"
	"sourcegraph.com/sourcegraph/sourcegraph/services/backend/accesscontrol"
	"sourcegraph.com/sourcegraph/sourcegraph/services/ext/github"
	"sourcegraph.com/sqs/pbtypes"
)

/*
 * Helpers
 */

func sortedRepoURIs(repos []*sourcegraph.Repo) []string {
	var uris []string
	for _, repo := range repos {
		uris = append(uris, repo.URI)
	}
	sort.Strings(uris)
	return uris
}

func (s *repos) mustCreate(ctx context.Context, t *testing.T, repos ...*sourcegraph.Repo) []*sourcegraph.Repo {
	var createdRepos []*sourcegraph.Repo
	for _, repo := range repos {
		repo.DefaultBranch = "master"

		if _, err := s.Create(ctx, repo); err != nil {
			t.Fatal(err)
		}
		repo, err := s.GetByURI(ctx, repo.URI)
		if err != nil {
			t.Fatal(err)
		}
		createdRepos = append(createdRepos, repo)
	}
	return createdRepos
}

/*
 * Tests
 */

func TestRepos_Get(t *testing.T) {
	if testing.Short() {
		t.Skip()
	}

	t.Parallel()

	ctx, _, done := testContext()
	defer done()

	s := repos{}

	want := s.mustCreate(ctx, t, &sourcegraph.Repo{URI: "r"})

	repo, err := s.Get(ctx, want[0].ID)
	if err != nil {
		t.Fatal(err)
	}
	if !jsonEqual(t, repo, want[0]) {
		t.Errorf("got %v, want %v", repo, want[0])
	}
}

func TestRepos_Get_origin(t *testing.T) {
	if testing.Short() {
		t.Skip()
	}

	t.Parallel()

	ctx, _, done := testContext()
	defer done()

	s := repos{}

	wantOrigin := &sourcegraph.Origin{ID: "id", Service: sourcegraph.Origin_GitHub, APIBaseURL: "u"}
	want := s.mustCreate(ctx, t, &sourcegraph.Repo{URI: "r", Origin: wantOrigin})

	repo, err := s.Get(ctx, want[0].ID)
	if err != nil {
		t.Fatal(err)
	}
	if !jsonEqual(t, repo, want[0]) {
		t.Errorf("got %v, want %v", repo, want[0])
	}
	if !reflect.DeepEqual(repo.Origin, wantOrigin) {
		t.Errorf("got origin %v, want %v", repo.Origin, wantOrigin)
	}
}

func TestRepos_List(t *testing.T) {
	if testing.Short() {
		t.Skip()
	}

	t.Parallel()

	ctx, _, done := testContext()
	defer done()

	ctx = github.WithMockHasAuthedUser(ctx, false)

	s := repos{}

	want := s.mustCreate(ctx, t, &sourcegraph.Repo{URI: "r"})

	repos, err := s.List(ctx, nil)
	if err != nil {
		t.Fatal(err)
	}
	if !jsonEqual(t, repos, want) {
		t.Errorf("got %v, want %v", repos, want)
	}
}

func TestRepos_List_type(t *testing.T) {
	if testing.Short() {
		t.Skip()
	}

	t.Parallel()

	r1 := &sourcegraph.Repo{URI: "r1", Private: true}
	r2 := &sourcegraph.Repo{URI: "r2"}

	ctx, _, done := testContext()
	defer done()

	ctx = github.WithMockHasAuthedUser(ctx, false)

	s := repos{}

	s.mustCreate(ctx, t, r1, r2)

	getRepoURIsByType := func(typ string) []string {
		repos, err := s.List(ctx, &store.RepoListOp{Type: typ})
		if err != nil {
			t.Fatal(err)
		}
		uris := make([]string, len(repos))
		for i, repo := range repos {
			uris[i] = repo.URI
		}
		sort.Strings(uris)
		return uris
	}

	if got, want := getRepoURIsByType("private"), []string{"r1"}; !reflect.DeepEqual(got, want) {
		t.Errorf("type %s: got %v, want %v", "enabled", got, want)
	}
	if got, want := getRepoURIsByType("public"), []string{"r2"}; !reflect.DeepEqual(got, want) {
		t.Errorf("type %s: got %v, want %v", "disabled", got, want)
	}
	all := []string{"r1", "r2"}
	if got := getRepoURIsByType("all"); !reflect.DeepEqual(got, all) {
		t.Errorf("type %s: got %v, want %v", "all", got, all)
	}
	if got := getRepoURIsByType(""); !reflect.DeepEqual(got, all) {
		t.Errorf("type %s: got %v, want %v", "empty", got, all)
	}
}

// TestRepos_List_query tests the behavior of Repos.List when called with
// a query.
func TestRepos_List_query(t *testing.T) {
	if testing.Short() {
		t.Skip()
	}

	t.Parallel()

	ctx, _, done := testContext()
	defer done()

	ctx = github.WithMockHasAuthedUser(ctx, false)

	s := repos{}

	// Add some repos.
	if _, err := s.Create(ctx, &sourcegraph.Repo{URI: "abc/def", Name: "def", DefaultBranch: "master"}); err != nil {
		t.Fatal(err)
	}
	if _, err := s.Create(ctx, &sourcegraph.Repo{URI: "def/ghi", Name: "ghi", DefaultBranch: "master"}); err != nil {
		t.Fatal(err)
	}
	if _, err := s.Create(ctx, &sourcegraph.Repo{URI: "jkl/mno/pqr", Name: "pqr", DefaultBranch: "master"}); err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		query string
		want  []string
	}{
		{"de", []string{"abc/def", "def/ghi"}},
		{"def", []string{"abc/def", "def/ghi"}},
		{"ABC/DEF", []string{"abc/def"}},
		{"xyz", nil},
	}
	for _, test := range tests {
		repos, err := s.List(ctx, &store.RepoListOp{Query: test.query})
		if err != nil {
			t.Fatal(err)
		}
		if got := sortedRepoURIs(repos); !reflect.DeepEqual(got, test.want) {
			t.Errorf("%q: got repos %q, want %q", test.query, got, test.want)
		}
	}
}

// TestRepos_List_URIs tests the behavior of Repos.List when called with
// URIs.
func TestRepos_List_URIs(t *testing.T) {
	if testing.Short() {
		t.Skip()
	}

	t.Parallel()

	ctx, _, done := testContext()
	defer done()

	ctx = github.WithMockHasAuthedUser(ctx, false)

	s := repos{}

	// Add some repos.
	if _, err := s.Create(ctx, &sourcegraph.Repo{URI: "a/b", DefaultBranch: "master"}); err != nil {
		t.Fatal(err)
	}
	if _, err := s.Create(ctx, &sourcegraph.Repo{URI: "c/d", DefaultBranch: "master"}); err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		uris []string
		want []string
	}{
		{[]string{"a/b"}, []string{"a/b"}},
		{[]string{"x/y"}, nil},
		{[]string{"a/b", "c/d"}, []string{"a/b", "c/d"}},
		{[]string{"a/b", "x/y", "c/d"}, []string{"a/b", "c/d"}},
	}
	for _, test := range tests {
		repos, err := s.List(ctx, &store.RepoListOp{URIs: test.uris})
		if err != nil {
			t.Fatal(err)
		}
		if got := sortedRepoURIs(repos); !reflect.DeepEqual(got, test.want) {
			t.Errorf("%v: got repos %q, want %q", test.uris, got, test.want)
		}
	}
}

func TestRepos_List_byOwner(t *testing.T) {
	if testing.Short() {
		t.Skip()
	}

	t.Parallel()

	ctx, _, done := testContext()
	defer done()

	ctx = github.WithMockHasAuthedUser(ctx, false)
	s := repos{}
	testRepos := []*sourcegraph.Repo{{URI: "a/r", Owner: "alice"}, {URI: "b/r", Owner: "bob"}}
	s.mustCreate(ctx, t, testRepos...)

	{
		repos, err := s.List(ctx, nil)
		if err != nil {
			t.Fatal(err)
		}
		if reflect.DeepEqual(repos, testRepos) {
			t.Errorf("expected %+v, got %+v", testRepos, repos)
		}
	}

	{
		repos, err := s.List(ctx, &store.RepoListOp{Owner: "alice"})
		if err != nil {
			t.Fatal(err)
		}
		if reflect.DeepEqual(repos, testRepos[0:1]) {
			t.Errorf("expected %+v, got %+v", testRepos[0:1], repos)
		}
	}
}

func TestRepos_List_GitHub_Authenticated(t *testing.T) {
	if testing.Short() {
		t.Skip()
	}

	ctx, mock, done := testContext()
	defer done()
	ctx = accesscontrol.WithInsecureSkip(ctx, false) // use real access controls

	calledListAccessible := mock.githubRepos.MockListAccessible(ctx, []*sourcegraph.Repo{
		&sourcegraph.Repo{URI: "github.com/is/privateButAccessible", Private: true, DefaultBranch: "master", Mirror: true, Origin: &sourcegraph.Origin{ID: "123"}},
	})
	mock.githubRepos.Get_ = func(ctx context.Context, uri string) (*sourcegraph.Repo, error) {
		if uri == "github.com/is/privateButAccessible" {
			return &sourcegraph.Repo{URI: "github.com/is/privateButAccessible", Private: true, DefaultBranch: "master", Mirror: true, Origin: &sourcegraph.Origin{ID: "123"}}, nil
		} else if uri == "github.com/is/public" {
			return &sourcegraph.Repo{URI: "github.com/is/public", Private: false, DefaultBranch: "master", Mirror: true, Origin: &sourcegraph.Origin{ID: "123"}}, nil
		}
		return nil, fmt.Errorf("unauthorized")
	}
	ctx = github.WithMockHasAuthedUser(ctx, true)

	s := repos{}

	createRepos := []*sourcegraph.Repo{
		&sourcegraph.Repo{URI: "a/local", Private: false, DefaultBranch: "master"},
		&sourcegraph.Repo{URI: "a/localPrivate", DefaultBranch: "master", Private: true},
		&sourcegraph.Repo{URI: "github.com/is/public", Private: false, DefaultBranch: "master", Mirror: true, Origin: &sourcegraph.Origin{ID: "123"}},
		&sourcegraph.Repo{URI: "github.com/is/privateButAccessible", Private: true, DefaultBranch: "master", Mirror: true, Origin: &sourcegraph.Origin{ID: "123"}},
		&sourcegraph.Repo{URI: "github.com/is/inaccessibleBecausePrivate", Private: true, DefaultBranch: "master", Mirror: true, Origin: &sourcegraph.Origin{ID: "456"}},
	}
	for _, repo := range createRepos {
		if _, err := s.Create(ctx, repo); err != nil {
			t.Fatal(err)
		}
	}

	repoList, err := s.List(ctx, &store.RepoListOp{})
	if err != nil {
		t.Fatal(err)
	}

	want := []string{"a/local", "github.com/is/privateButAccessible", "github.com/is/public"}
	if got := sortedRepoURIs(repoList); !reflect.DeepEqual(got, want) {
		t.Fatalf("got repos %q, want %q", got, want)
	}
	if !*calledListAccessible {
		t.Error("!calledListAccessible")
	}
}

func TestRepos_List_GitHub_Authenticated_NoReposAccessible(t *testing.T) {
	if testing.Short() {
		t.Skip()
	}

	ctx, mock, done := testContext()
	defer done()
	ctx = accesscontrol.WithInsecureSkip(ctx, false) // use real access controls

	s := repos{}

	calledListAccessible := mock.githubRepos.MockListAccessible(ctx, nil)
	mock.githubRepos.Get_ = func(ctx context.Context, uri string) (*sourcegraph.Repo, error) {
		return nil, fmt.Errorf("unauthorized")
	}

	ctx = github.WithMockHasAuthedUser(ctx, true)

	createRepos := []*sourcegraph.Repo{
		&sourcegraph.Repo{URI: "github.com/not/accessible", DefaultBranch: "master", Mirror: true, Origin: &sourcegraph.Origin{ID: "456"}, Private: true},
	}
	for _, repo := range createRepos {
		if _, err := s.Create(ctx, repo); err != nil {
			t.Fatal(err)
		}
	}

	repoList, err := s.List(ctx, &store.RepoListOp{})
	if err != nil {
		t.Fatal(err)
	}

	if len(repoList) != 0 {
		t.Errorf("got repos %v, want empty", repoList)
	}
	if !*calledListAccessible {
		t.Error("!calledListAccessible")
	}
}

func TestRepos_List_GitHub_Unauthenticated(t *testing.T) {
	if testing.Short() {
		t.Skip()
	}

	ctx, mock, done := testContext()
	defer done()
	ctx = accesscontrol.WithInsecureSkip(ctx, false) // use real access controls

	calledListAccessible := mock.githubRepos.MockListAccessible(ctx, nil)
	ctx = github.WithMockHasAuthedUser(ctx, false)
	mock.githubRepos.Get_ = func(ctx context.Context, uri string) (*sourcegraph.Repo, error) {
		return nil, fmt.Errorf("unauthorized")
	}

	s := repos{}

	if _, err := s.Create(ctx, &sourcegraph.Repo{URI: "github.com/private", Private: true, DefaultBranch: "master", Mirror: true}); err != nil {
		t.Fatal(err)
	}

	repoList, err := s.List(ctx, &store.RepoListOp{})
	if err != nil {
		t.Fatal(err)
	}

	if got := sortedRepoURIs(repoList); len(got) != 0 {
		t.Fatal("List should not have returned any repos, got:", got)
	}

	if *calledListAccessible {
		t.Error("calledListAccessible, but wanted not called since there is no authed user")
	}
}

func TestRepos_Create(t *testing.T) {
	if testing.Short() {
		t.Skip()
	}

	t.Parallel()

	ctx, _, done := testContext()
	defer done()

	s := repos{}

	tm := time.Now().Round(time.Second)
	ts := pbtypes.NewTimestamp(tm)

	// Add a repo.
	if _, err := s.Create(ctx, &sourcegraph.Repo{URI: "a/b", CreatedAt: &ts, DefaultBranch: "master"}); err != nil {
		t.Fatal(err)
	}

	repo, err := s.GetByURI(ctx, "a/b")
	if err != nil {
		t.Fatal(err)
	}
	if repo.CreatedAt == nil {
		t.Fatal("got CreatedAt nil")
	}
	if want := ts.Time(); !repo.CreatedAt.Time().Equal(want) {
		t.Errorf("got CreatedAt %q, want %q", repo.CreatedAt.Time(), want)
	}
}

func TestRepos_Create_dupe(t *testing.T) {
	if testing.Short() {
		t.Skip()
	}

	t.Parallel()

	ctx, _, done := testContext()
	defer done()

	s := repos{}

	tm := time.Now().Round(time.Second)
	ts := pbtypes.NewTimestamp(tm)

	// Add a repo.
	if _, err := s.Create(ctx, &sourcegraph.Repo{URI: "a/b", CreatedAt: &ts, DefaultBranch: "master"}); err != nil {
		t.Fatal(err)
	}

	// Add another repo with the same name.
	if _, err := s.Create(ctx, &sourcegraph.Repo{URI: "a/b", CreatedAt: &ts, DefaultBranch: "master"}); err == nil {
		t.Fatalf("got err == nil, want an error when creating a duplicate repo")
	}
}

// TestRepos_Update_Description tests the behavior of Repos.Update to
// update a repo's description.
func TestRepos_Update_Description(t *testing.T) {
	if testing.Short() {
		t.Skip()
	}

	t.Parallel()

	ctx, _, done := testContext()
	defer done()

	s := repos{}

	// Add a repo.
	if _, err := s.Create(ctx, &sourcegraph.Repo{URI: "a/b", DefaultBranch: "master"}); err != nil {
		t.Fatal(err)
	}

	repo, err := s.GetByURI(ctx, "a/b")
	if err != nil {
		t.Fatal(err)
	}
	if want := ""; repo.Description != want {
		t.Errorf("got description %q, want %q", repo.Description, want)
	}

	if err := s.Update(ctx, store.RepoUpdate{ReposUpdateOp: &sourcegraph.ReposUpdateOp{Repo: repo.ID, Description: "d"}}); err != nil {
		t.Fatal(err)
	}

	repo, err = s.GetByURI(ctx, "a/b")
	if err != nil {
		t.Fatal(err)
	}
	if want := "d"; repo.Description != want {
		t.Errorf("got description %q, want %q", repo.Description, want)
	}
}

// TestRepos_Update_Origin tests the behavior of Repos.Update to
// update a repo's origin.
func TestRepos_Update_Origin(t *testing.T) {
	if testing.Short() {
		t.Skip()
	}

	t.Parallel()

	ctx, _, done := testContext()
	defer done()

	s := repos{}

	// Add a repo.
	if _, err := s.Create(ctx, &sourcegraph.Repo{URI: "a/b", DefaultBranch: "master"}); err != nil {
		t.Fatal(err)
	}

	repo, err := s.GetByURI(ctx, "a/b")
	if err != nil {
		t.Fatal(err)
	}
	if want := ""; repo.Description != want {
		t.Errorf("got description %q, want %q", repo.Description, want)
	}

	newOrigin := &sourcegraph.Origin{ID: "123", Service: sourcegraph.Origin_GitHub, APIBaseURL: "https://api.github.com"}
	if err := s.Update(ctx, store.RepoUpdate{ReposUpdateOp: &sourcegraph.ReposUpdateOp{Repo: repo.ID, Origin: newOrigin}}); err != nil {
		t.Fatal(err)
	}

	repo, err = s.GetByURI(ctx, "a/b")
	if err != nil {
		t.Fatal(err)
	}
	if want := newOrigin; !reflect.DeepEqual(newOrigin, repo.Origin) {
		t.Errorf("got origin %+v, want %+v", repo.Origin, want)
	}
}

func TestRepos_Update_UpdatedAt(t *testing.T) {
	if testing.Short() {
		t.Skip()
	}

	t.Parallel()

	ctx, _, done := testContext()
	defer done()

	s := repos{}

	// Add a repo.
	if _, err := s.Create(ctx, &sourcegraph.Repo{URI: "a/b", DefaultBranch: "master"}); err != nil {
		t.Fatal(err)
	}

	repo, err := s.GetByURI(ctx, "a/b")
	if err != nil {
		t.Fatal(err)
	}
	if repo.UpdatedAt != nil {
		t.Errorf("got UpdatedAt %v, want nil", repo.UpdatedAt.Time())
	}

	// Perform any update.
	newTime := time.Unix(123456, 0)
	if err := s.Update(ctx, store.RepoUpdate{ReposUpdateOp: &sourcegraph.ReposUpdateOp{Repo: repo.ID}, UpdatedAt: &newTime}); err != nil {
		t.Fatal(err)
	}

	repo, err = s.GetByURI(ctx, "a/b")
	if err != nil {
		t.Fatal(err)
	}
	if repo.UpdatedAt == nil {
		t.Fatal("got UpdatedAt nil, want non-nil")
	}
	if want := newTime; !repo.UpdatedAt.Time().Equal(want) {
		t.Errorf("got UpdatedAt %q, want %q", repo.UpdatedAt.Time(), want)
	}
}

func TestRepos_Update_PushedAt(t *testing.T) {
	if testing.Short() {
		t.Skip()
	}

	t.Parallel()

	ctx, _, done := testContext()
	defer done()

	s := repos{}

	// Add a repo.
	if _, err := s.Create(ctx, &sourcegraph.Repo{URI: "a/b", DefaultBranch: "master"}); err != nil {
		t.Fatal(err)
	}

	repo, err := s.GetByURI(ctx, "a/b")
	if err != nil {
		t.Fatal(err)
	}
	if repo.PushedAt != nil {
		t.Errorf("got PushedAt %v, want nil", repo.PushedAt.Time())
	}

	newTime := time.Unix(123456, 0)
	if err := s.Update(ctx, store.RepoUpdate{ReposUpdateOp: &sourcegraph.ReposUpdateOp{Repo: repo.ID}, PushedAt: &newTime}); err != nil {
		t.Fatal(err)
	}

	repo, err = s.GetByURI(ctx, "a/b")
	if err != nil {
		t.Fatal(err)
	}
	if repo.PushedAt == nil {
		t.Fatal("got PushedAt nil, want non-nil")
	}
	if repo.UpdatedAt != nil {
		t.Fatal("got UpdatedAt non-nil, want nil")
	}
	if want := newTime; !repo.PushedAt.Time().Equal(want) {
		t.Errorf("got PushedAt %q, want %q", repo.PushedAt.Time(), want)
	}
}

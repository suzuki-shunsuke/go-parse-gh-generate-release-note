package rnote_test

import (
	"fmt"
	"reflect"
	"testing"

	"github.com/suzuki-shunsuke/go-parse-gh-generate-release-note/rnote"
)

func Example() {
	body := `## What's Changed
* Add feature foo by @alice in https://github.com/o/r/pull/1
* Fix bar by @bob in https://github.com/o/r/pull/2

## New Contributors
* @bob made their first contribution in https://github.com/o/r/pull/2

**Full Changelog**: https://github.com/o/r/compare/v1.0.0...v1.1.0
`
	note := rnote.Parse(body)
	fmt.Println("Change Log URL:", note.FullChangelogURL)
	fmt.Println("Pull Requests:")
	for _, pr := range note.PullRequests {
		fmt.Println(pr.URL, pr.Author)
	}
	fmt.Println("New Contributors:")
	for _, c := range note.NewContributors {
		fmt.Println(c.PullRequestURL, c.Login)
	}
	// Output:
	// Change Log URL: https://github.com/o/r/compare/v1.0.0...v1.1.0
	// Pull Requests:
	// https://github.com/o/r/pull/1 alice
	// https://github.com/o/r/pull/2 bob
	// New Contributors:
	// https://github.com/o/r/pull/2 bob
}

func TestParse(t *testing.T) { //nolint:funlen
	t.Parallel()

	happyBody := `## What's Changed
* Add feature foo by @alice in https://github.com/o/r/pull/1
* Fix bar by @bob in https://github.com/o/r/pull/2

## New Contributors
* @bob made their first contribution in https://github.com/o/r/pull/2

**Full Changelog**: https://github.com/o/r/compare/v1.0.0...v1.1.0
`

	pr1 := &rnote.PullRequest{Number: 1, URL: "https://github.com/o/r/pull/1", Title: "Add feature foo", Author: "alice"}
	pr2 := &rnote.PullRequest{Number: 2, URL: "https://github.com/o/r/pull/2", Title: "Fix bar", Author: "bob"}

	tests := []struct {
		name string
		body string
		want *rnote.ReleaseNote
		// linkCheck: optional; returns (contributorIndex, expectedPRIndex) pairs to verify pointer equality.
		linkCheck [][2]int
	}{
		{
			name: "happy path",
			body: happyBody,
			want: &rnote.ReleaseNote{
				PullRequests:     []*rnote.PullRequest{pr1, pr2},
				FullChangelogURL: "https://github.com/o/r/compare/v1.0.0...v1.1.0",
				NewContributors: []*rnote.Contributor{
					{Login: "bob", PullRequestURL: "https://github.com/o/r/pull/2", PullRequest: pr2},
				},
			},
			linkCheck: [][2]int{{0, 1}},
		},
		{
			name: "empty body",
			body: "",
			want: &rnote.ReleaseNote{},
		},
		{
			name: "missing new contributors section",
			body: `## What's Changed
* Add feature foo by @alice in https://github.com/o/r/pull/1

**Full Changelog**: https://github.com/o/r/compare/v1.0.0...v1.1.0
`,
			want: &rnote.ReleaseNote{
				PullRequests: []*rnote.PullRequest{
					{Number: 1, URL: "https://github.com/o/r/pull/1", Title: "Add feature foo", Author: "alice"},
				},
				FullChangelogURL: "https://github.com/o/r/compare/v1.0.0...v1.1.0",
			},
		},
		{
			name: "missing full changelog",
			body: `## What's Changed
* Add feature foo by @alice in https://github.com/o/r/pull/1
`,
			want: &rnote.ReleaseNote{
				PullRequests: []*rnote.PullRequest{
					{Number: 1, URL: "https://github.com/o/r/pull/1", Title: "Add feature foo", Author: "alice"},
				},
			},
		},
		{
			name: "title contains 'by @' and ' in '",
			body: `## What's Changed
* Refactor login by @admin flow in module by @carol in https://github.com/o/r/pull/42
`,
			want: &rnote.ReleaseNote{
				PullRequests: []*rnote.PullRequest{
					{
						Number: 42,
						URL:    "https://github.com/o/r/pull/42",
						Title:  "Refactor login by @admin flow in module",
						Author: "carol",
					},
				},
			},
		},
		{
			name: "unknown section is ignored",
			body: `## What's Changed
* Add feature foo by @alice in https://github.com/o/r/pull/1

## Other Stuff
* not a PR line

**Full Changelog**: https://github.com/o/r/compare/v1.0.0...v1.1.0
`,
			want: &rnote.ReleaseNote{
				PullRequests: []*rnote.PullRequest{
					{Number: 1, URL: "https://github.com/o/r/pull/1", Title: "Add feature foo", Author: "alice"},
				},
				FullChangelogURL: "https://github.com/o/r/compare/v1.0.0...v1.1.0",
			},
		},
		{
			name: "contributor without matching PR",
			body: `## What's Changed
* Add feature foo by @alice in https://github.com/o/r/pull/1

## New Contributors
* @ghost made their first contribution in https://github.com/o/r/pull/999
`,
			want: &rnote.ReleaseNote{
				PullRequests: []*rnote.PullRequest{
					{Number: 1, URL: "https://github.com/o/r/pull/1", Title: "Add feature foo", Author: "alice"},
				},
				NewContributors: []*rnote.Contributor{
					{Login: "ghost", PullRequestURL: "https://github.com/o/r/pull/999", PullRequest: nil},
				},
			},
		},
		{
			name: "CRLF line endings",
			body: "## What's Changed\r\n" +
				"* Add feature foo by @alice in https://github.com/o/r/pull/1\r\n" +
				"\r\n" +
				"**Full Changelog**: https://github.com/o/r/compare/v1.0.0...v1.1.0\r\n",
			want: &rnote.ReleaseNote{
				PullRequests: []*rnote.PullRequest{
					{Number: 1, URL: "https://github.com/o/r/pull/1", Title: "Add feature foo", Author: "alice"},
				},
				FullChangelogURL: "https://github.com/o/r/compare/v1.0.0...v1.1.0",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := rnote.Parse(tt.body)
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("Parse mismatch\n got: %#v\nwant: %#v", got, tt.want)
				return
			}
			for _, pair := range tt.linkCheck {
				cIdx, prIdx := pair[0], pair[1]
				if got.NewContributors[cIdx].PullRequest != got.PullRequests[prIdx] {
					t.Errorf("NewContributors[%d].PullRequest is not the same pointer as PullRequests[%d]", cIdx, prIdx)
				}
			}
		})
	}
}

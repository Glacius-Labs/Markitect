package model

import (
	"reflect"
	"testing"
)

func TestBodyHTMLAnchorProtection(t *testing.T) {
	tests := []struct {
		name       string
		body       string
		wantLinks  []string
		wantSyntax bool
	}{
		{
			name:      "valid link in anchor and after close",
			body:      "x <a href='/x'>[[api]]</a> [[field]]",
			wantLinks: []string{"field"},
		},
		{
			name:      "orphan close does not cancel following anchor",
			body:      "x </a> <a href='/x'>[[api]]</a> [[field]]",
			wantLinks: []string{"field"},
		},
		{
			name:       "malformed link in anchor is protected",
			body:       "x <a href='/x'>[[api|]]</a> [[field]]",
			wantLinks:  []string{"field"},
			wantSyntax: false,
		},
		{
			name:       "orphan close does not expose malformed notation in following anchor",
			body:       "x </a> <a href='/x'>[[api|]]</a> [[field]]",
			wantLinks:  []string{"field"},
			wantSyntax: false,
		},
		{
			name:       "malformed link in span remains active",
			body:       "x <span>[[api|]]</span> [[field]]",
			wantLinks:  []string{"field"},
			wantSyntax: true,
		},
		{
			name:      "anchor name is case insensitive and tag prefix is not a link",
			body:      "<A data-x='1'>[[api]]</A> <abbr>[[field]]</abbr>",
			wantLinks: []string{"field"},
		},
		{
			name:      "anchor context resets at paragraph boundary",
			body:      "<a>[[api]]\n\n[[field]]",
			wantLinks: []string{"field"},
		},
		{
			name:      "display HTML looking text stays literal outside anchor",
			body:      "[[api|<a>]] [[field]]",
			wantLinks: []string{"api", "field"},
		},
		{
			name:      "native closing tag is seen inside protected notation",
			body:      "<a>[[api|</a>]] [[field]]",
			wantLinks: []string{"field"},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got := parseBodySyntax(test.body, DefaultLimits())
			links := make([]string, 0, len(got.occurrences))
			for _, occurrence := range got.occurrences {
				links = append(links, occurrence.written)
				notation := test.body[occurrence.span.StartByte:occurrence.span.EndByte]
				if len(notation) < 4 || notation[:2] != "[[" || notation[len(notation)-2:] != "]]" {
					t.Fatalf("occurrence span does not point to its original notation: %+v", occurrence)
				}
			}
			var syntax bool
			for _, problem := range got.problems {
				syntax = syntax || problem.code == "body.syntax"
			}
			if !reflect.DeepEqual(links, test.wantLinks) || syntax != test.wantSyntax {
				t.Fatalf("links/problems differ: got %q %+v; want links %q, syntax=%t", links, got.problems, test.wantLinks, test.wantSyntax)
			}
		})
	}
}

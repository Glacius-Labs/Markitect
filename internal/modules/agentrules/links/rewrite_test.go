package links

import (
	"errors"
	"strings"
	"testing"
)

func TestRewriteInlineLinksImagesAndDestinationSyntax(t *testing.T) {
	input := "ordinary [workflow](../workflows/setup-workspace.md#install?mode=quick) and ![diagram](../images/architecture%20map.png \"Architecture\")\n" +
		"angle [angle](<../tools/a%20b.md#part>) nested [paren](../tools/a_(b).md) escaped [escape](../tools/a\\(b\\).md)\n" +
		"external [site](https://example.test/a_(b)?q=1#x) and [mail](mailto:hello@example.test)\n"
	got, err := Rewrite(input, func(destination string) (string, error) {
		if strings.HasPrefix(destination, "../") {
			return "docs/markitect/engineering/" + strings.TrimPrefix(destination, "../"), nil
		}
		return destination, nil
	})
	if err != nil {
		t.Fatal(err)
	}
	want := "ordinary [workflow](docs/markitect/engineering/workflows/setup-workspace.md#install?mode=quick) and ![diagram](docs/markitect/engineering/images/architecture%20map.png \"Architecture\")\n" +
		"angle [angle](<docs/markitect/engineering/tools/a%20b.md#part>) nested [paren](docs/markitect/engineering/tools/a_%28b%29.md) escaped [escape](docs/markitect/engineering/tools/a%28b%29.md)\n" +
		"external [site](https://example.test/a_(b)?q=1#x) and [mail](mailto:hello@example.test)\n"
	if got != want {
		t.Fatalf("rewrite mismatch\nwant: %q\n got: %q", want, got)
	}
}

func TestRewriteReferenceDefinitionsAndUses(t *testing.T) {
	input := "[workflow]: ../workflows/setup-workspace.md#run \"title\"\n" +
		"[image]: <../images/diagram.svg?raw=1> 'diagram'\n" +
		"[workflow] [shortcut] [label][workflow] [collapsed][] ![image]\n" +
		"[collapsed]: ../images/collapsed.svg\n"
	got, err := Rewrite(input, func(destination string) (string, error) {
		if strings.HasPrefix(destination, "../") {
			return "central/" + strings.TrimPrefix(destination, "../"), nil
		}
		return destination, nil
	})
	if err != nil {
		t.Fatal(err)
	}
	want := "[workflow]: central/workflows/setup-workspace.md#run \"title\"\n" +
		"[image]: <central/images/diagram.svg?raw=1> 'diagram'\n" +
		"[workflow] [shortcut] [label][workflow] [collapsed][] ![image]\n" +
		"[collapsed]: central/images/collapsed.svg\n"
	if got != want {
		t.Fatalf("rewrite mismatch\nwant: %q\n got: %q", want, got)
	}
}

func TestRewriteMultilineReferenceDestination(t *testing.T) {
	input := "[workflow]:\n  ../workflows/setup-workspace.md#install\n\n[workflow]\n"
	got, err := Rewrite(input, func(destination string) (string, error) {
		return "central/" + strings.TrimPrefix(destination, "../"), nil
	})
	if err != nil {
		t.Fatal(err)
	}
	want := "[workflow]:\n  central/workflows/setup-workspace.md#install\n\n[workflow]\n"
	if got != want {
		t.Fatalf("rewrite mismatch\nwant: %q\n got: %q", want, got)
	}
}

func TestRewriteReferenceDefinitionsInsideContainers(t *testing.T) {
	input := "> [quoted]: ../quoted.md\n>\n> [quoted]\n\n- [listed]: ../listed.md\n\n  [listed]\n"
	got, err := Rewrite(input, func(destination string) (string, error) {
		return "central/" + strings.TrimPrefix(destination, "../"), nil
	})
	if err != nil {
		t.Fatal(err)
	}
	want := "> [quoted]: central/quoted.md\n>\n> [quoted]\n\n- [listed]: central/listed.md\n\n  [listed]\n"
	if got != want {
		t.Fatalf("rewrite mismatch\nwant: %q\n got: %q", want, got)
	}
}

func TestRewriteOnlyEffectiveDefinitionsThatAreUsed(t *testing.T) {
	input := "[unused]: ../unused.md\n[dup]: ../first.md\n[dup]: ../second.md\n[dup]\n"
	calls := 0
	got, err := Rewrite(input, func(destination string) (string, error) {
		calls++
		return "central/" + strings.TrimPrefix(destination, "../"), nil
	})
	if err != nil {
		t.Fatal(err)
	}
	want := "[unused]: ../unused.md\n[dup]: central/first.md\n[dup]: ../second.md\n[dup]\n"
	if got != want || calls != 1 {
		t.Fatalf("got %q with %d callback calls, want %q and one call", got, calls, want)
	}
}

func TestRewriteDecodesEscapesAndEntitiesButPreservesUnchangedBytes(t *testing.T) {
	input := `[x](a\(b\).md?x=1&amp;y=2) [keep](a\(b\).md)`
	seen := []string{}
	got, err := Rewrite(input, func(destination string) (string, error) {
		seen = append(seen, destination)
		if strings.Contains(destination, "&") {
			return "central/a%28b%29.md?x=1&y=2", nil
		}
		return destination, nil
	})
	if err != nil {
		t.Fatal(err)
	}
	want := `[x](central/a%28b%29.md?x=1&amp;y=2) [keep](a\(b\).md)`
	if got != want {
		t.Fatalf("rewrite mismatch\nwant: %q\n got: %q", want, got)
	}
	if len(seen) != 2 || seen[0] != "a(b).md?x=1&y=2" || seen[1] != "a(b).md" {
		t.Fatalf("unexpected decoded callback values: %#v", seen)
	}
}

func TestRewriteLeavesCodeAndHTMLVerbatim(t *testing.T) {
	input := "`[inline](old.md)` [real](old.md)\n" +
		"``inline `[code](old.md)` ``\n" +
		"    [indented](old.md)\n\t![tab](old.png)\n" +
		">     [blockquote indented](old.md)\n" +
		"-     [list indented](old.md)\n" +
		"> ```md\n> [blockquote fenced](old.md)\n> ```\n" +
		"- ```md\n  [list fenced](old.md)\n  ```\n" +
		"~~~md\n[ fenced ](old.md)\n~~~\n" +
		"<!-- [comment](old.md)\n[continued](old.md) -->\n" +
		"<div>\n[html block](old.md)\n</div>\n" +
		"<script>\n[script](old.md)\n</script>\n" +
		`<a href="[attribute](old.md)">tag</a>` + "\n" +
		`<https://external.example/a>` + "\n" +
		`\[escaped](old.md)` + "\n"
	got, err := Rewrite(input, func(destination string) (string, error) {
		return "new.md", nil
	})
	if err != nil {
		t.Fatal(err)
	}
	want := strings.Replace(input, "[real](old.md)", "[real](new.md)", 1)
	if got != want {
		t.Fatalf("non-link bytes changed\nwant: %q\n got: %q", want, got)
	}
}

func TestRewriteStopsContainerFencesAtSiblingOrContainerEnd(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{
			name:  "blockquote ends",
			input: "> ```md\n> [literal](old.md)\n\n[real](old.md)",
			want:  "> ```md\n> [literal](old.md)\n\n[real](new.md)",
		},
		{
			name:  "list sibling ends",
			input: "- ```md\n  [literal](old.md)\n- [real](old.md)",
			want:  "- ```md\n  [literal](old.md)\n- [real](new.md)",
		},
		{
			name:  "list continuation fence ends at sibling after blank",
			input: "- item\n  ```md\n  [literal](old.md)\n\n- [real](old.md)",
			want:  "- item\n  ```md\n  [literal](old.md)\n\n- [real](new.md)",
		},
		{
			name:  "blockquote list continuation fence ends at sibling",
			input: "> - item\n>   ```md\n>   [literal](old.md)\n>\n> - [real](old.md)",
			want:  "> - item\n>   ```md\n>   [literal](old.md)\n>\n> - [real](new.md)",
		},
		{
			name:  "four marker spaces are list content",
			input: "-    ```md\n- [real](old.md)",
			want:  "-    ```md\n- [real](new.md)",
		},
		{
			name:  "five marker spaces start indented code",
			input: "-     [literal](old.md)\n- [real](old.md)",
			want:  "-     [literal](old.md)\n- [real](new.md)",
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got, err := Rewrite(test.input, func(string) (string, error) { return "new.md", nil })
			if err != nil {
				t.Fatal(err)
			}
			if got != test.want {
				t.Fatalf("got %q, want %q", got, test.want)
			}
		})
	}
}

func TestRewriteRejectsInvalidBareDestinations(t *testing.T) {
	for _, input := range []string{
		"[not a link](a( b).md)",
		"[not a link](a(\nb).md)",
		"[not a link](a< b).md)",
		"[not a link](a> b).md)",
		"[not a link](<a<b>.md>)",
	} {
		t.Run(strings.ReplaceAll(input, "\n", "newline"), func(t *testing.T) {
			calls := 0
			got, err := Rewrite(input, func(string) (string, error) {
				calls++
				return "new.md", nil
			})
			if err != nil {
				t.Fatal(err)
			}
			if got != input || calls != 0 {
				t.Fatalf("invalid link was rewritten: %q", got)
			}
		})
	}
}

func TestRewriteAcceptsSpacesInsideAngleDestinations(t *testing.T) {
	input := `[valid](<folder name/file.md>)`
	got, err := Rewrite(input, func(destination string) (string, error) {
		if destination != "folder name/file.md" {
			t.Fatalf("callback got %q", destination)
		}
		return "folder name/central.md", nil
	})
	if err != nil {
		t.Fatal(err)
	}
	want := `[valid](<folder name/central.md>)`
	if got != want {
		t.Fatalf("got %q, want %q", got, want)
	}
}

func TestInlineCodeMasksIncompleteHTMLBeforeHTMLScanning(t *testing.T) {
	input := "`<span` [real](old.md) <b>"
	got, err := Rewrite(input, func(string) (string, error) { return "new.md", nil })
	if err != nil {
		t.Fatal(err)
	}
	want := "`<span` [real](new.md) <b>"
	if got != want {
		t.Fatalf("got %q, want %q", got, want)
	}
}

func TestRewriteSupportsOrdinaryMarkdownReferenceEscapes(t *testing.T) {
	input := "[path]: ../docs/a&amp;b.md\n[ref][path] [path]\n"
	got, err := Rewrite(input, func(destination string) (string, error) {
		if destination == "../docs/a&b.md" {
			return "central/docs/a%26b.md", nil
		}
		return destination, nil
	})
	if err != nil {
		t.Fatal(err)
	}
	want := "[path]: central/docs/a%26b.md\n[ref][path] [path]\n"
	if got != want {
		t.Fatalf("rewrite mismatch\nwant: %q\n got: %q", want, got)
	}
}

func TestRewriteNestedImageInsideLinkLabel(t *testing.T) {
	input := `[![architecture](../images/architecture.svg?raw=1)](../docs/architecture.md#overview)`
	got, err := Rewrite(input, func(destination string) (string, error) {
		return "central/" + strings.TrimPrefix(destination, "../"), nil
	})
	if err != nil {
		t.Fatal(err)
	}
	want := `[![architecture](central/images/architecture.svg?raw=1)](central/docs/architecture.md#overview)`
	if got != want {
		t.Fatalf("rewrite mismatch\nwant: %q\n got: %q", want, got)
	}
}

func TestRewriteUsedImageReferenceInsideLinkLabel(t *testing.T) {
	input := "[workspace-flow]: ../images/workspace-flow.svg\n[![flow][workspace-flow]](../docs/workspaces.md#flow)\n"
	got, err := Rewrite(input, func(destination string) (string, error) {
		return "central/" + strings.TrimPrefix(destination, "../"), nil
	})
	if err != nil {
		t.Fatal(err)
	}
	want := "[workspace-flow]: central/images/workspace-flow.svg\n[![flow][workspace-flow]](central/docs/workspaces.md#flow)\n"
	if got != want {
		t.Fatalf("rewrite mismatch\nwant: %q\n got: %q", want, got)
	}
}

func TestRewriteUsedImageReferenceInsideOuterReferenceLink(t *testing.T) {
	input := "[![img][image]][outer]\n\n[image]: ../image.png\n[outer]: ../guide.md\n"
	got, err := Rewrite(input, func(destination string) (string, error) {
		return "central/" + strings.TrimPrefix(destination, "../"), nil
	})
	if err != nil {
		t.Fatal(err)
	}
	want := "[![img][image]][outer]\n\n[image]: central/image.png\n[outer]: central/guide.md\n"
	if got != want {
		t.Fatalf("rewrite mismatch\nwant: %q\n got: %q", want, got)
	}
}

func TestRewriteEscapesEntityLookingAmpersandsInNewDestinations(t *testing.T) {
	input := `[query](old.md)`
	got, err := Rewrite(input, func(string) (string, error) {
		return `new.md?literal=&amp;copy;`, nil
	})
	if err != nil {
		t.Fatal(err)
	}
	want := `[query](new.md?literal=&amp;amp;copy;)`
	if got != want {
		t.Fatalf("rewrite mismatch\nwant: %q\n got: %q", want, got)
	}
	seen := ""
	_, err = Rewrite(got, func(destination string) (string, error) {
		seen = destination
		return destination, nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if seen != `new.md?literal=&amp;copy;` {
		t.Fatalf("rewritten destination changed after entity decoding: %q", seen)
	}
}

func TestRewriteReturnsCallbackError(t *testing.T) {
	want := errors.New("no mapping")
	_, err := Rewrite("[link](source.md)", func(string) (string, error) { return "", want })
	if !errors.Is(err, want) {
		t.Fatalf("got error %v, want %v", err, want)
	}
}

func TestRewriteNilCallback(t *testing.T) {
	if _, err := Rewrite("plain", nil); err == nil {
		t.Fatal("expected error for nil callback")
	}
}

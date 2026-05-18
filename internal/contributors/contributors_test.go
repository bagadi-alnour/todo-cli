package contributors

import "testing"

func TestParseShortlogLine(t *testing.T) {
	c, ok := parseShortlogLine("  42\tAlice Example <alice@example.com>")
	if !ok {
		t.Fatal("expected parse ok")
	}
	if c.Name != "Alice Example" || c.Email != "alice@example.com" || c.Commits != 42 {
		t.Fatalf("unexpected contributor: %+v", c)
	}
}

func TestNormalizeEmail(t *testing.T) {
	if got := NormalizeEmail("  Alice@Example.COM "); got != "alice@example.com" {
		t.Fatalf("got %q", got)
	}
}

func TestIsBot(t *testing.T) {
	if !IsBot("dependabot[bot]", "dependabot@users.noreply.github.com") {
		t.Fatal("expected bot")
	}
	if IsBot("Alice", "alice@example.com") {
		t.Fatal("expected human")
	}
}

func TestParseAuthor(t *testing.T) {
	name, email, ok := parseAuthor("Bob <bob@test.com>")
	if !ok || name != "Bob" || email != "bob@test.com" {
		t.Fatalf("got %q %q %v", name, email, ok)
	}
}

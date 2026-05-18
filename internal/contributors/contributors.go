package contributors

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/bagadi-alnour/todo-cli/internal/git"
	"github.com/bagadi-alnour/todo-cli/internal/storage"
)

const contributorsFile = "contributors.json"

// Contributor is a git author who has committed to the repository.
type Contributor struct {
	Name    string `json:"name"`
	Email   string `json:"email"`
	Commits int    `json:"commits,omitempty"`
}

// File is the cached contributor list stored under .todos/.
type File struct {
	Version      int           `json:"version"`
	UpdatedAt    time.Time     `json:"updatedAt"`
	Contributors []Contributor `json:"contributors"`
}

// CachePath returns the path to contributors.json for a project.
func CachePath(projectRoot string) string {
	return filepath.Join(projectRoot, storage.TodosDir, contributorsFile)
}

// Load reads the contributor cache, returning an empty file if missing.
func Load(projectRoot string) (*File, error) {
	path := CachePath(projectRoot)
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return &File{Version: 1, Contributors: []Contributor{}}, nil
		}
		return nil, fmt.Errorf("read contributors cache: %w", err)
	}
	var f File
	if err := json.Unmarshal(data, &f); err != nil {
		return nil, fmt.Errorf("parse contributors cache: %w", err)
	}
	if f.Contributors == nil {
		f.Contributors = []Contributor{}
	}
	return &f, nil
}

// Save writes the contributor cache.
func Save(projectRoot string, f *File) error {
	if f == nil {
		f = &File{Version: 1, Contributors: []Contributor{}}
	}
	f.Version = 1
	f.UpdatedAt = time.Now()
	data, err := json.MarshalIndent(f, "", "  ")
	if err != nil {
		return err
	}
	path := CachePath(projectRoot)
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return err
	}
	return os.WriteFile(path, data, 0644)
}

// RefreshFromGit rebuilds the contributor list from git shortlog.
func RefreshFromGit(projectRoot string) (*File, error) {
	if !git.IsGitRepo() {
		return nil, fmt.Errorf("not a git repository")
	}
	list, err := listFromGit(projectRoot)
	if err != nil {
		return nil, err
	}
	f := &File{Version: 1, UpdatedAt: time.Now(), Contributors: list}
	if err := Save(projectRoot, f); err != nil {
		return nil, err
	}
	return f, nil
}

// EnsureLoaded returns cached contributors, refreshing from git when the cache is empty.
func EnsureLoaded(projectRoot string) (*File, error) {
	f, err := Load(projectRoot)
	if err != nil {
		return nil, err
	}
	if len(f.Contributors) == 0 && git.IsGitRepo() {
		return RefreshFromGit(projectRoot)
	}
	return f, nil
}

func listFromGit(projectRoot string) ([]Contributor, error) {
	cmd := exec.Command("git", "-C", projectRoot, "shortlog", "-sne", "HEAD")
	out, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("git shortlog: %w", err)
	}
	byEmail := map[string]*Contributor{}
	for _, line := range strings.Split(string(out), "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		c, ok := parseShortlogLine(line)
		if !ok || IsBot(c.Name, c.Email) {
			continue
		}
		key := NormalizeEmail(c.Email)
		if existing, ok := byEmail[key]; ok {
			existing.Commits += c.Commits
			if existing.Name == "" && c.Name != "" {
				existing.Name = c.Name
			}
		} else {
			copy := c
			byEmail[key] = &copy
		}
	}
	list := make([]Contributor, 0, len(byEmail))
	for _, c := range byEmail {
		list = append(list, *c)
	}
	sort.Slice(list, func(i, j int) bool {
		if list[i].Commits != list[j].Commits {
			return list[i].Commits > list[j].Commits
		}
		return strings.ToLower(list[i].Name) < strings.ToLower(list[j].Name)
	})
	return list, nil
}

// parseShortlogLine parses "  123\tName <email>".
func parseShortlogLine(line string) (Contributor, bool) {
	tab := strings.IndexByte(line, '\t')
	if tab < 0 {
		return Contributor{}, false
	}
	countPart := strings.TrimSpace(line[:tab])
	author := strings.TrimSpace(line[tab+1:])
	var commits int
	fmt.Sscanf(countPart, "%d", &commits)
	name, email, ok := parseAuthor(author)
	if !ok {
		return Contributor{}, false
	}
	return Contributor{Name: name, Email: NormalizeEmail(email), Commits: commits}, true
}

func parseAuthor(author string) (name, email string, ok bool) {
	author = strings.TrimSpace(author)
	if author == "" {
		return "", "", false
	}
	if lt := strings.LastIndex(author, "<"); lt >= 0 && strings.HasSuffix(author, ">") {
		email = strings.TrimSpace(author[lt+1 : len(author)-1])
		name = strings.TrimSpace(author[:lt])
		if email == "" {
			return "", "", false
		}
		if name == "" {
			name = email
		}
		return name, email, true
	}
	if strings.Contains(author, "@") {
		return author, author, true
	}
	return author, "", false
}

// NormalizeEmail lowercases and trims an email address.
func NormalizeEmail(email string) string {
	return strings.ToLower(strings.TrimSpace(email))
}

// IsBot reports whether a contributor looks like an automated account.
func IsBot(name, email string) bool {
	lowerName := strings.ToLower(name)
	lowerEmail := strings.ToLower(email)
	if strings.Contains(lowerName, "[bot]") || strings.HasSuffix(lowerName, "bot") {
		return true
	}
	if strings.Contains(lowerEmail, "noreply") || strings.Contains(lowerEmail, "dependabot") {
		return true
	}
	if strings.HasSuffix(lowerEmail, "[bot]") {
		return true
	}
	return false
}

// Resolve finds a contributor email by query (name, email prefix, or "me").
func Resolve(projectRoot, query string) (email string, display string, err error) {
	query = strings.TrimSpace(query)
	if query == "" {
		return "", "", fmt.Errorf("assignee cannot be empty")
	}

	f, err := EnsureLoaded(projectRoot)
	if err != nil {
		return "", "", err
	}

	if strings.EqualFold(query, "me") {
		myEmail, err := git.GetUserEmail()
		if err != nil {
			return "", "", fmt.Errorf("could not read git user.email: %w", err)
		}
		myEmail = NormalizeEmail(myEmail)
		if myEmail == "" {
			return "", "", fmt.Errorf("git user.email is not set")
		}
		for _, c := range f.Contributors {
			if NormalizeEmail(c.Email) == myEmail {
				return myEmail, DisplayName(c), nil
			}
		}
		return myEmail, myEmail, nil
	}

	q := strings.ToLower(query)
	var matches []Contributor
	for _, c := range f.Contributors {
		emailLower := NormalizeEmail(c.Email)
		nameLower := strings.ToLower(c.Name)
		local := strings.Split(emailLower, "@")[0]
		if emailLower == q || strings.HasPrefix(emailLower, q) ||
			nameLower == q || strings.HasPrefix(nameLower, q) ||
			strings.HasPrefix(local, q) {
			matches = append(matches, c)
		}
	}
	if len(matches) == 0 {
		return "", "", fmt.Errorf("no contributor matching %q (run: todo contributors --refresh)", query)
	}
	if len(matches) > 1 {
		names := make([]string, 0, len(matches))
		for _, m := range matches {
			names = append(names, DisplayName(m))
		}
		return "", "", fmt.Errorf("ambiguous assignee %q: %s", query, strings.Join(names, ", "))
	}
	c := matches[0]
	return NormalizeEmail(c.Email), DisplayName(c), nil
}

// DisplayName returns a human-readable label for a contributor.
func DisplayName(c Contributor) string {
	if c.Name != "" && !strings.EqualFold(c.Name, c.Email) {
		return c.Name
	}
	return c.Email
}

// LookupName finds the display name for an email in the cache.
func LookupName(projectRoot, email string) string {
	if email == "" {
		return ""
	}
	f, err := Load(projectRoot)
	if err != nil {
		return email
	}
	email = NormalizeEmail(email)
	for _, c := range f.Contributors {
		if NormalizeEmail(c.Email) == email {
			return DisplayName(c)
		}
	}
	return email
}

// SuggestFromBlame returns top contributors for the given paths via git blame.
func SuggestFromBlame(projectRoot string, paths []string) ([]Contributor, error) {
	if !git.IsGitRepo() || len(paths) == 0 {
		return nil, nil
	}
	counts := map[string]int{}
	names := map[string]string{}
	for _, p := range paths {
		p = strings.TrimSpace(p)
		if p == "" {
			continue
		}
		abs := p
		if !filepath.IsAbs(p) {
			abs = filepath.Join(projectRoot, p)
		}
		if info, err := os.Stat(abs); err != nil || info.IsDir() {
			continue
		}
		rel, err := filepath.Rel(projectRoot, abs)
		if err != nil {
			rel = p
		}
		rel = filepath.ToSlash(rel)
		cmd := exec.Command("git", "-C", projectRoot, "blame", "--line-porcelain", "-w", "HEAD", "--", rel)
		out, err := cmd.Output()
		if err != nil {
			continue
		}
		var currentName, currentEmail string
		for _, line := range strings.Split(string(out), "\n") {
			if strings.HasPrefix(line, "author-mail ") {
				currentEmail = strings.Trim(strings.TrimPrefix(line, "author-mail "), "<>")
				currentEmail = NormalizeEmail(currentEmail)
			}
			if strings.HasPrefix(line, "author ") {
				currentName = strings.TrimPrefix(line, "author ")
			}
			if line == "\t" || strings.HasPrefix(line, "\t") {
				if currentEmail != "" && !IsBot(currentName, currentEmail) {
					counts[currentEmail]++
					if names[currentEmail] == "" && currentName != "" {
						names[currentEmail] = currentName
					}
				}
				currentName, currentEmail = "", ""
			}
		}
	}
	if len(counts) == 0 {
		return nil, nil
	}
	type scored struct {
		c Contributor
		n int
	}
	var list []scored
	for email, n := range counts {
		list = append(list, scored{Contributor{Name: names[email], Email: email, Commits: n}, n})
	}
	sort.Slice(list, func(i, j int) bool {
		return list[i].n > list[j].n
	})
	out := make([]Contributor, 0, len(list))
	for _, s := range list {
		out = append(out, s.c)
	}
	return out, nil
}

// MatchEmails returns contributor emails matching a filter query (for list --assignee).
func MatchEmails(projectRoot, query string) ([]string, error) {
	query = strings.TrimSpace(query)
	if query == "" {
		return nil, nil
	}
	email, _, err := Resolve(projectRoot, query)
	if err != nil {
		return nil, err
	}
	return []string{email}, nil
}

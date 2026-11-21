package board

import (
	"fmt"
	"os"
	"regexp"
	"strings"
)

type accessPolicy struct {
	Version    int         `json:"version"`
	Principals []principal `json:"principals"`
	Topics     []string    `json:"topics"`
	Grants     []grant     `json:"grants"`
}

type principal struct {
	ID   string `json:"id"`
	Role string `json:"role,omitempty"`
}

type grant struct {
	Scope       string   `json:"scope"`
	Principal   string   `json:"principal"`
	Permissions []string `json:"permissions"`
}

var slug = regexp.MustCompile(`^[a-z0-9][a-z0-9_-]{0,63}$`)
var reserved = regexp.MustCompile(`^(con|prn|aux|nul|com[0-9]|lpt[0-9])$`)

func validSlug(value string) bool {
	return slug.MatchString(value) && !reserved.MatchString(value)
}

func principalParts(id string) (string, string, bool) {
	kind, name, _ := strings.Cut(id, ":")
	return kind, name, (kind == "human" || kind == "agent" || kind == "skill") && validSlug(name)
}

func loadAccess(index *os.Root) (accessPolicy, error) {
	policy, err := metadata[accessPolicy](index, "access", accessLimit)
	if err != nil {
		return policy, fmt.Errorf("invalid access policy: %w", err)
	}

	roles := map[string]string{}
	hasAdmin := false
	for i, entry := range policy.Principals {
		kind, _, valid := principalParts(entry.ID)
		if !valid || roles[entry.ID] != "" || (entry.Role == "admin" && kind != "human") {
			return policy, fmt.Errorf("invalid or duplicate principal %q", entry.ID)
		}
		if entry.Role == "" {
			entry.Role = "member"
		}
		hasAdmin = hasAdmin || entry.Role == "admin"
		roles[entry.ID] = entry.Role
		policy.Principals[i] = entry
	}
	if !hasAdmin {
		return policy, fmt.Errorf("access policy needs a human administrator")
	}

	for _, topic := range policy.Topics {
		if !validSlug(topic) {
			return policy, fmt.Errorf("invalid topic %q", topic)
		}
	}
	seen := make(map[[2]string]bool)
	for _, g := range policy.Grants {
		key := [2]string{g.Scope, g.Principal}
		if !policy.hasScope(g.Scope) || roles[g.Principal] == "" || seen[key] {
			return policy, fmt.Errorf("invalid or duplicate grant for %q in %q", g.Principal, g.Scope)
		}
		seen[key] = true
		if !strings.HasPrefix(g.Scope, "topic:") {
			for _, permission := range g.Permissions {
				if permission != "read" && permission != "status" {
					return policy, fmt.Errorf("invalid private scope permission %q", permission)
				}
			}
		}
	}
	return policy, nil
}

func (p accessPolicy) role(id string) string {
	for _, entry := range p.Principals {
		if entry.ID == id {
			return entry.Role
		}
	}
	return ""
}

func (p accessPolicy) hasScope(scope string) bool {
	kind, name, _ := strings.Cut(scope, ":")
	if kind == "topic" {
		for _, topic := range p.Topics {
			if topic == name {
				return true
			}
		}
		return false
	}
	return (kind == "agent" || kind == "skill") && p.role(scope) != ""
}

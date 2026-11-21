package board

import (
	"cmp"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"slices"
	"strings"
)

// Access holds the current policy for one operation under the board lock.
type Access struct {
	root, index *os.Root
	policy      accessPolicy
	actor       string
	edit        bool
}

// WithAccess keeps authorization and the operation under the same board lock.
func WithAccess(dir, actor string, edit bool, operation func(*Access) error) error {
	if _, _, valid := principalParts(actor); !valid {
		return errors.New("a registered --as <principal> is required")
	}
	worktree, _, err := workspace(dir)
	if err != nil {
		return err
	}
	checkout, err := os.OpenRoot(worktree)
	if err != nil {
		return err
	}
	defer checkout.Close()
	root, err := openDirectory(checkout, boardName)
	if err != nil {
		return err
	}
	defer root.Close()
	index, err := openDirectory(root, "index")
	if err != nil {
		return err
	}
	defer index.Close()
	lock, err := openRegular(index, "board.lock", os.O_RDWR)
	if err != nil {
		return err
	}
	defer lock.Close()
	acquire := lockShared
	if edit {
		acquire = lockExclusive
	}
	if err := acquire(lock); err != nil {
		return err
	}

	policy, err := loadAccess(index)
	if err != nil {
		return err
	}
	role := policy.role(actor)
	if role == "" {
		return fmt.Errorf("unregistered principal %q", actor)
	}
	if edit && role != "admin" {
		return errors.New("operation requires a board administrator")
	}
	return operation(&Access{root: root, index: index, policy: policy, actor: actor, edit: edit})
}

type ScopePermissions struct {
	Scope       string   `json:"scope"`
	Permissions []string `json:"permissions"`
}

type Identity struct {
	Principal string             `json:"principal"`
	Role      string             `json:"role"`
	Scopes    []ScopePermissions `json:"scopes"`
}

var permissionOrder = []string{"read", "write", "delete", "status"}

func (p accessPolicy) permissions(actor, scope string) []string {
	role := p.role(actor)
	if role == "" || !p.hasScope(scope) {
		return nil
	}
	if role == "admin" {
		return slices.Clone(permissionOrder)
	}
	allowed := map[string]bool{}
	if actor == scope {
		allowed["read"], allowed["write"] = true, true
	}
	for _, grant := range p.Grants {
		if grant.Scope == scope && grant.Principal == actor {
			for _, permission := range grant.Permissions {
				allowed[permission] = true
			}
			allowed["read"] = true
			break
		}
	}
	result := []string{}
	for _, permission := range permissionOrder {
		if allowed[permission] {
			result = append(result, permission)
		}
	}
	return result
}

func (a *Access) Whoami() Identity {
	identity := Identity{Principal: a.actor, Role: a.policy.role(a.actor), Scopes: []ScopePermissions{}}
	scopes := make([]string, 0, len(a.policy.Topics)+len(a.policy.Principals))
	for _, topic := range a.policy.Topics {
		scopes = append(scopes, "topic:"+topic)
	}
	for _, entry := range a.policy.Principals {
		if !strings.HasPrefix(entry.ID, "human:") {
			scopes = append(scopes, entry.ID)
		}
	}
	slices.Sort(scopes)
	for _, scope := range scopes {
		if permissions := a.policy.permissions(a.actor, scope); len(permissions) > 0 {
			identity.Scopes = append(identity.Scopes, ScopePermissions{Scope: scope, Permissions: permissions})
		}
	}
	return identity
}

func (a *Access) Principals() ([]principal, error) {
	if a.policy.role(a.actor) != "admin" {
		return nil, errors.New("operation requires a board administrator")
	}
	list := slices.Clone(a.policy.Principals)
	for i := range list {
		list[i].Role = list[i].effectiveRole()
	}
	slices.SortFunc(list, func(a, b principal) int { return cmp.Compare(a.ID, b.ID) })
	return list, nil
}

func (a *Access) Grants(scope string) ([]grant, error) {
	if a.policy.role(a.actor) != "admin" && (scope != a.actor || !privateScope(scope)) {
		return nil, errors.New("select your private scope or use a board administrator")
	}
	if scope != "" && !a.policy.hasScope(scope) {
		return nil, fmt.Errorf("unknown scope %q", scope)
	}
	list := []grant{}
	for _, g := range a.policy.Grants {
		if scope == "" || scope == g.Scope {
			g.Permissions = orderedPermissions(g.Permissions)
			list = append(list, g)
		}
	}
	slices.SortFunc(list, func(a, b grant) int {
		if a.Scope != b.Scope {
			return cmp.Compare(a.Scope, b.Scope)
		}
		return cmp.Compare(a.Principal, b.Principal)
	})
	return list, nil
}

func privateScope(scope string) bool {
	return strings.HasPrefix(scope, "agent:") || strings.HasPrefix(scope, "skill:")
}

func orderedPermissions(values []string) []string {
	ordered := make([]string, 0, len(values))
	for _, p := range permissionOrder {
		if slices.Contains(values, p) {
			ordered = append(ordered, p)
		}
	}
	return ordered
}

func (a *Access) save(next accessPolicy, change func() (func() error, error)) error {
	if !a.edit || a.policy.role(a.actor) != "admin" {
		return errors.New("operation requires a board administrator and an exclusive board lock")
	}
	data, err := json.MarshalIndent(next, "", "  ")
	if err != nil {
		return err
	}
	data = append(data, '\n')
	if _, err := decodeMetadata[accessPolicy]("access", data, accessLimit); err != nil {
		return err
	}
	if err := next.validate(); err != nil {
		return err
	}

	var rollback func() error
	if change != nil {
		rollback, err = change()
		if err != nil {
			return err
		}
	}
	if err := replaceFile(a.index, "access.json", data, 0600); err != nil {
		if rollback != nil {
			return errors.Join(err, rollback())
		}
		return err
	}
	a.policy = next
	return nil
}

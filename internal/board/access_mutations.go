package board

import (
	"errors"
	"fmt"
	"io"
	"os"
	"slices"
)

func (a *Access) AddPrincipal(id, role string) error {
	if err := checkRole(role); err != nil {
		return err
	}
	kind, name, valid := principalParts(id)
	if !valid {
		return fmt.Errorf("invalid principal %q", id)
	}
	if a.policy.role(id) != "" {
		return fmt.Errorf("principal %q already exists", id)
	}
	next := a.policy
	next.Principals = append(slices.Clone(next.Principals), principal{ID: id, Role: role})
	if kind == "human" {
		return a.save(next, nil)
	}
	return a.makeScope(next, kind+"s", name)
}

func (a *Access) SetRole(id, role string) (bool, error) {
	if err := checkRole(role); err != nil {
		return false, err
	}
	for i, entry := range a.policy.Principals {
		if entry.ID != id {
			continue
		}
		if entry.effectiveRole() == role {
			return false, nil
		}
		next := a.policy
		next.Principals = slices.Clone(next.Principals)
		next.Principals[i].Role = role
		err := a.save(next, nil)
		return err == nil, err
	}
	return false, fmt.Errorf("unknown principal %q", id)
}

func (a *Access) CreateTopic(name string) error {
	if !validSlug(name) {
		return fmt.Errorf("invalid topic %q", name)
	}
	if slices.Contains(a.policy.Topics, name) {
		return fmt.Errorf("topic %q already exists", name)
	}
	next := a.policy
	next.Topics = append(slices.Clone(next.Topics), name)
	return a.makeScope(next, "topics", name)
}

func checkRole(role string) error {
	if role != "member" && role != "admin" {
		return fmt.Errorf("invalid role %q", role)
	}
	return nil
}

func (a *Access) makeScope(next accessPolicy, container, name string) error {
	parent, err := openDirectory(a.root, container)
	if err != nil {
		return err
	}
	defer parent.Close()
	return a.save(next, func() (func() error, error) {
		if err := parent.Mkdir(name, 0700); err != nil {
			if os.IsExist(err) {
				return nil, checkOrphanScope(parent, name)
			}
			return nil, err
		}
		return func() error { return removeEmptyDirectory(parent, name) }, nil
	})
}

// An interrupted create can leave an empty directory without a policy entry.
func checkOrphanScope(parent *os.Root, name string) error {
	scope, err := openDirectory(parent, name)
	if err != nil {
		return err
	}
	defer scope.Close()
	entries, err := scope.Open(".")
	if err != nil {
		return err
	}
	defer entries.Close()
	if _, err := entries.ReadDir(1); !errors.Is(err, io.EOF) {
		if err != nil {
			return err
		}
		return fmt.Errorf("unregistered scope folder %q is not empty", name)
	}
	return nil
}

func (a *Access) DeleteTopic(name string) error {
	i := slices.Index(a.policy.Topics, name)
	if i < 0 {
		return fmt.Errorf("unknown topic %q", name)
	}
	scope := "topic:" + name
	cat, err := loadCatalog(a.index, a.policy)
	if err != nil {
		return err
	}
	for _, doc := range cat.Documents {
		if doc.Scope == scope {
			return fmt.Errorf("%s still has index entries", scope)
		}
	}

	topics, err := openDirectory(a.root, "topics")
	if err != nil {
		return err
	}
	defer topics.Close()
	next := a.policy
	next.Topics = slices.Delete(slices.Clone(next.Topics), i, i+1)
	next.Grants = slices.DeleteFunc(slices.Clone(next.Grants), func(g grant) bool { return g.Scope == scope })
	return a.save(next, func() (func() error, error) {
		if err := removeEmptyDirectory(topics, name); err != nil {
			if os.IsNotExist(err) {
				return nil, nil
			}
			return nil, fmt.Errorf("remove empty %s folder: %w", scope, err)
		}
		return func() error { return topics.Mkdir(name, 0700) }, nil
	})
}

func (a *Access) ChangeGrant(scope, subject, permission string, add bool) (bool, error) {
	if !a.policy.hasScope(scope) {
		return false, fmt.Errorf("unknown scope %q", scope)
	}
	if a.policy.role(subject) == "" {
		return false, fmt.Errorf("unknown principal %q", subject)
	}
	if !slices.Contains(permissionOrder, permission) || (privateScope(scope) && permission != "read" && permission != "status") {
		return false, fmt.Errorf("permission %q is invalid for %s", permission, scope)
	}
	next := a.policy
	next.Grants = slices.Clone(next.Grants)
	i := slices.IndexFunc(next.Grants, func(g grant) bool { return g.Scope == scope && g.Principal == subject })
	if i < 0 {
		if !add {
			return false, nil
		}
		next.Grants = append(next.Grants, grant{Scope: scope, Principal: subject, Permissions: []string{permission}})
	} else {
		values := next.Grants[i].Permissions
		j := slices.Index(values, permission)
		switch {
		case add && j < 0:
			next.Grants[i].Permissions = orderedPermissions(append(slices.Clone(values), permission))
		case !add && j >= 0 && len(values) == 1:
			next.Grants = slices.Delete(next.Grants, i, i+1)
		case !add && j >= 0:
			next.Grants[i].Permissions = slices.Delete(slices.Clone(values), j, j+1)
		default:
			return false, nil
		}
	}
	err := a.save(next, nil)
	return err == nil, err
}

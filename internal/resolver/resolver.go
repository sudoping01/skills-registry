package resolver

import (
	"fmt"

	"github.com/skillhub/skillhub/internal/format"
	"github.com/skillhub/skillhub/internal/registry"
)

// ResolvedSkill is a skill that has been resolved and is ready to install.
type ResolvedSkill struct {
	User    string
	Name    string
	Version string
}

// Resolve returns the full ordered install list for a skill (deps first, root last),
// with circular dependency detection.
func Resolve(user, name, version string, client *registry.Client) ([]ResolvedSkill, error) {
	visited := map[string]bool{}
	var ordered []ResolvedSkill
	if err := resolve(user, name, version, client, visited, &ordered); err != nil {
		return nil, err
	}
	return ordered, nil
}

func resolve(
	user, name, version string,
	client *registry.Client,
	visited map[string]bool,
	ordered *[]ResolvedSkill,
) error {
	key := user + "/" + name
	if visited[key] {
		return fmt.Errorf("circular dependency detected: %s", key)
	}
	visited[key] = true
	defer func() { visited[key] = false }()

	meta, err := client.Info(user, name)
	if err != nil {
		return fmt.Errorf("dependency %s/%s not found in registry: %w", user, name, err)
	}

	resolvedVersion := version
	if resolvedVersion == "" {
		resolvedVersion = meta.Version
	}

	for _, rawDep := range meta.Dependencies {
		dep, err := format.ParseDependency(rawDep)
		if err != nil {
			return fmt.Errorf("invalid dependency in %s/%s: %w", user, name, err)
		}
		if err := resolve(dep.User, dep.Name, dep.Version, client, visited, ordered); err != nil {
			return err
		}
	}

	*ordered = append(*ordered, ResolvedSkill{
		User:    user,
		Name:    name,
		Version: resolvedVersion,
	})
	return nil
}

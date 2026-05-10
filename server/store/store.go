package store

import "github.com/skillhub/skillhub/internal/registry"

// Stats holds aggregate counts across the whole registry.
type Stats struct {
	TotalSkills    int
	TotalCreators  int
	TotalDownloads int
}

// Store is the interface for skill storage backends.
type Store interface {
	Save(meta registry.SkillMeta, archiveData []byte) error
	GetMeta(user, name string) (*registry.SkillMeta, error)
	GetMetaVersion(user, name, version string) (*registry.SkillMeta, error)
	GetArchive(user, name, version string) ([]byte, error)
	Search(query string) ([]registry.SkillMeta, error)
	Stats() Stats
	IncrementDownloads(user, name string)
	CreateUserToken(username string) (string, error)
	ValidateUserToken(username, token string) bool
	GetSkillMdBody(user, name, version string) (string, error)
}

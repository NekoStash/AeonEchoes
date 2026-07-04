package skills

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"aeonechoes/server/internal/domain"
	"aeonechoes/server/internal/repository"
)

const (
	defaultSourceID   = "default_skills_directory"
	metadataChecksum  = "checksum"
	metadataSourceTyp = "source_type"
	metadataRelPath   = "relative_path"
)

// Store is the persistence surface required by the skills service.
type Store interface {
	CreateSkillSource(source domain.SkillSource) (domain.SkillSource, error)
	GetSkillSource(id string) (domain.SkillSource, error)
	CreateSkill(skill domain.Skill) (domain.Skill, error)
	UpdateSkill(id string, skill domain.Skill) (domain.Skill, error)
	DeleteSkill(id string) error
	ListSkills(filter repository.SkillFilter) ([]domain.Skill, error)
}

// Service manages skills sourced from inline content and skill directories.
type Service struct {
	store      Store
	defaultDir string
}

// ScanResult summarizes one directory synchronization pass.
type ScanResult struct {
	SourceID  string    `json:"source_id"`
	Path      string    `json:"path"`
	Created   int       `json:"created"`
	Updated   int       `json:"updated"`
	Deleted   int       `json:"deleted"`
	Unchanged int       `json:"unchanged"`
	Errors    []string  `json:"errors,omitempty"`
	ScannedAt time.Time `json:"scanned_at"`
}

type discoveredSkill struct {
	Name         string
	Description  string
	Content      string
	Version      string
	Metadata     map[string]string
	RelativePath string
}

// NewService builds a skills service backed by store.
func NewService(store Store, defaultDir string) *Service {
	return &Service{store: store, defaultDir: strings.TrimSpace(defaultDir)}
}

// EnsureDefaultSource returns or creates the process-level default directory source.
func (s *Service) EnsureDefaultSource(ctx context.Context) (domain.SkillSource, error) {
	if err := ctx.Err(); err != nil {
		return domain.SkillSource{}, err
	}
	if s.store == nil {
		return domain.SkillSource{}, fmt.Errorf("skills store must not be nil")
	}
	if s.defaultDir == "" {
		return domain.SkillSource{}, fmt.Errorf("default skills directory must not be empty")
	}

	source, err := s.store.GetSkillSource(defaultSourceID)
	if err == nil {
		return source, nil
	}

	created, createErr := s.store.CreateSkillSource(domain.SkillSource{
		ID:      defaultSourceID,
		Name:    "Default skills directory",
		Type:    domain.SkillSourceDirectory,
		Path:    s.defaultDir,
		Enabled: true,
	})
	if createErr != nil {
		return domain.SkillSource{}, fmt.Errorf("ensure default skill source: %w", createErr)
	}
	return created, nil
}

// ScanDefault synchronizes skills from the default directory source.
func (s *Service) ScanDefault(ctx context.Context) (ScanResult, error) {
	source, err := s.EnsureDefaultSource(ctx)
	if err != nil {
		return ScanResult{}, err
	}
	return s.scanDirectorySource(ctx, source)
}

// ScanSource synchronizes skills from a persisted directory source.
func (s *Service) ScanSource(ctx context.Context, sourceID string) (ScanResult, error) {
	if err := ctx.Err(); err != nil {
		return ScanResult{}, err
	}
	if s.store == nil {
		return ScanResult{}, fmt.Errorf("skills store must not be nil")
	}
	sourceID = strings.TrimSpace(sourceID)
	if sourceID == "" {
		return ScanResult{}, fmt.Errorf("skill source id must not be empty")
	}
	source, err := s.store.GetSkillSource(sourceID)
	if err != nil {
		return ScanResult{}, fmt.Errorf("get skill source %q: %w", sourceID, err)
	}
	if source.Type != domain.SkillSourceDirectory {
		return ScanResult{}, fmt.Errorf("skill source %q type %q is not directory", source.ID, source.Type)
	}
	return s.scanDirectorySource(ctx, source)
}

// CreateInline creates one inline_text source and its normalized skill.
func (s *Service) CreateInline(ctx context.Context, name, description, content string, enabled bool, metadata map[string]string) (domain.Skill, error) {
	if err := ctx.Err(); err != nil {
		return domain.Skill{}, err
	}
	if s.store == nil {
		return domain.Skill{}, fmt.Errorf("skills store must not be nil")
	}
	name = strings.TrimSpace(name)
	if name == "" {
		return domain.Skill{}, fmt.Errorf("inline skill name must not be empty")
	}
	if strings.TrimSpace(content) == "" {
		return domain.Skill{}, fmt.Errorf("inline skill content must not be empty")
	}

	source, err := s.store.CreateSkillSource(domain.SkillSource{
		Name:       name,
		Type:       domain.SkillSourceInlineText,
		InlineText: content,
		Enabled:    enabled,
		Metadata:   cloneMetadata(metadata),
	})
	if err != nil {
		return domain.Skill{}, fmt.Errorf("create inline skill source: %w", err)
	}

	skill, err := s.store.CreateSkill(domain.Skill{
		SourceID:    source.ID,
		Name:        name,
		Description: strings.TrimSpace(description),
		Content:     content,
		Enabled:     enabled,
		Metadata:    cloneMetadata(metadata),
	})
	if err != nil {
		return domain.Skill{}, fmt.Errorf("create inline skill: %w", err)
	}
	return skill, nil
}

func (s *Service) scanDirectorySource(ctx context.Context, source domain.SkillSource) (ScanResult, error) {
	if err := ctx.Err(); err != nil {
		return ScanResult{}, err
	}
	if source.Type != domain.SkillSourceDirectory {
		return ScanResult{}, fmt.Errorf("skill source %q type %q is not directory", source.ID, source.Type)
	}
	root := strings.TrimSpace(source.Path)
	if root == "" {
		return ScanResult{}, fmt.Errorf("directory skill source %q path must not be empty", source.ID)
	}

	result := ScanResult{SourceID: source.ID, Path: root, ScannedAt: time.Now().UTC()}
	discovered, err := scanDirectory(ctx, root)
	if err != nil {
		return result, err
	}

	existing, err := s.store.ListSkills(repository.SkillFilter{SourceID: source.ID})
	if err != nil {
		return result, fmt.Errorf("list skills for source %q: %w", source.ID, err)
	}

	byRelativePath := make(map[string]domain.Skill, len(existing))
	byName := make(map[string]domain.Skill, len(existing))
	for _, skill := range existing {
		if rel := strings.TrimSpace(skill.Metadata[metadataRelPath]); rel != "" {
			byRelativePath[rel] = skill
		}
		if name := strings.TrimSpace(skill.Name); name != "" {
			byName[name] = skill
		}
	}

	seen := make(map[string]bool, len(discovered))
	for _, item := range discovered {
		if err := ctx.Err(); err != nil {
			return result, err
		}
		matched, found := byRelativePath[item.RelativePath]
		if !found {
			matched, found = byName[item.Name]
		}

		metadata := cloneMetadata(item.Metadata)
		metadata[metadataChecksum] = checksum(item.Content)
		metadata[metadataSourceTyp] = string(domain.SkillSourceDirectory)
		metadata[metadataRelPath] = item.RelativePath
		if item.Version != "" {
			metadata["version"] = item.Version
		}

		if !found {
			_, err := s.store.CreateSkill(domain.Skill{
				SourceID:    source.ID,
				Name:        item.Name,
				Description: item.Description,
				Content:     item.Content,
				Path:        item.RelativePath,
				Enabled:     true,
				Metadata:    metadata,
			})
			if err != nil {
				return result, fmt.Errorf("create directory skill %q: %w", item.RelativePath, err)
			}
			result.Created++
			continue
		}

		seen[matched.ID] = true
		if strings.TrimSpace(matched.Metadata[metadataChecksum]) == metadata[metadataChecksum] {
			result.Unchanged++
			continue
		}
		_, err := s.store.UpdateSkill(matched.ID, domain.Skill{
			SourceID:    source.ID,
			Name:        item.Name,
			Description: item.Description,
			Content:     item.Content,
			Path:        item.RelativePath,
			Enabled:     matched.Enabled,
			Metadata:    metadata,
		})
		if err != nil {
			return result, fmt.Errorf("update directory skill %q: %w", matched.ID, err)
		}
		result.Updated++
	}

	for _, skill := range existing {
		if err := ctx.Err(); err != nil {
			return result, err
		}
		if seen[skill.ID] {
			continue
		}
		if err := s.store.DeleteSkill(skill.ID); err != nil {
			return result, fmt.Errorf("delete missing directory skill %q: %w", skill.ID, err)
		}
		result.Deleted++
	}

	return result, nil
}

func scanDirectory(ctx context.Context, root string) ([]discoveredSkill, error) {
	entries, err := os.ReadDir(root)
	if err != nil {
		return nil, fmt.Errorf("read skills directory %q: %w", root, err)
	}
	items := make([]discoveredSkill, 0, len(entries))
	for _, entry := range entries {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		if !entry.IsDir() {
			continue
		}
		discovered, ok, err := readSkillDirectory(root, entry.Name())
		if err != nil {
			return nil, err
		}
		if ok {
			items = append(items, discovered)
		}
	}
	return items, nil
}

func readSkillDirectory(root, dirName string) (discoveredSkill, bool, error) {
	candidates := []string{"SKILL.md", "skill.md", "skill.json"}
	for _, candidate := range candidates {
		path := filepath.Join(root, dirName, candidate)
		content, err := os.ReadFile(path)
		if err != nil {
			if errorsIsNotExist(err) {
				continue
			}
			return discoveredSkill{}, false, fmt.Errorf("read skill file %q: %w", path, err)
		}
		if strings.HasSuffix(candidate, ".json") {
			item, err := parseJSONSkill(dirName, content)
			if err != nil {
				return discoveredSkill{}, false, fmt.Errorf("parse skill json %q: %w", path, err)
			}
			return item, true, nil
		}
		return parseMarkdownSkill(dirName, string(content)), true, nil
	}
	return discoveredSkill{}, false, nil
}

func parseMarkdownSkill(dirName, content string) discoveredSkill {
	name := dirName
	description := ""
	lines := strings.Split(content, "\n")
	for index, line := range lines {
		trimmed := strings.TrimSpace(line)
		if index == 0 && strings.HasPrefix(trimmed, "# ") {
			if heading := strings.TrimSpace(strings.TrimPrefix(trimmed, "# ")); heading != "" {
				name = heading
			}
			continue
		}
		if description == "" && trimmed != "" && !strings.HasPrefix(trimmed, "#") {
			description = trimmed
			break
		}
	}
	return discoveredSkill{Name: name, Description: description, Content: content, Metadata: map[string]string{}, RelativePath: dirName}
}

type jsonSkillFile struct {
	Name        string            `json:"name"`
	Description string            `json:"description"`
	Content     string            `json:"content"`
	Version     string            `json:"version"`
	Metadata    map[string]string `json:"metadata"`
}

func parseJSONSkill(dirName string, content []byte) (discoveredSkill, error) {
	var data jsonSkillFile
	if err := json.Unmarshal(content, &data); err != nil {
		return discoveredSkill{}, err
	}
	data.Name = strings.TrimSpace(data.Name)
	if data.Name == "" {
		data.Name = dirName
	}
	if strings.TrimSpace(data.Content) == "" {
		return discoveredSkill{}, fmt.Errorf("content must not be empty")
	}
	return discoveredSkill{
		Name:         data.Name,
		Description:  strings.TrimSpace(data.Description),
		Content:      data.Content,
		Version:      strings.TrimSpace(data.Version),
		Metadata:     cloneMetadata(data.Metadata),
		RelativePath: dirName,
	}, nil
}

func cloneMetadata(metadata map[string]string) map[string]string {
	if len(metadata) == 0 {
		return map[string]string{}
	}
	cloned := make(map[string]string, len(metadata))
	for key, value := range metadata {
		cloned[key] = value
	}
	return cloned
}

func checksum(content string) string {
	sum := sha256.Sum256([]byte(content))
	return hex.EncodeToString(sum[:])
}

func errorsIsNotExist(err error) bool {
	return errors.Is(err, os.ErrNotExist)
}

package ruledfiles

import (
	"arch-agent/internal/agent"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"
)

var ErrReadOnly = fmt.Errorf("file is read only")

// Allow path for read only operations else return error 'ErrReadOnly'
func WithReadOnlyPath(guardedPath string) Option {
	return func(rfs *RuledFileSystem) {

		guard := func(path string) error {

			if strings.Contains(path, guardedPath) {
				return ErrReadOnly
			}
			return nil
		}

		rfs.WritePath.validators = append(rfs.WritePath.validators, guard)
		rfs.AppendPath.validators = append(rfs.AppendPath.validators, guard)
		rfs.DeletePath.validators = append(rfs.DeletePath.validators, guard)
	}
}

// All pathes stated with path can't be deleted
func WithUndeletable(guardedPath string) Option {
	return func(rfs *RuledFileSystem) {

		guard := func(path string) error {

			if strings.HasPrefix(path, guardedPath) {
				return fmt.Errorf("path can't be deleted")
			}
			return nil
		}

		rfs.DeletePath.validators = append(rfs.DeletePath.validators, guard)
	}
}

// Change path prefix
// e.g.
// /mnt/home/ -> file:///
// /mnt/home/some_folder/readme.md -> file:///some_folder/readme.md
// or on any given prefix
func WithChangedGlobalPrefix(oldPrefix, newPrefix string) Option {
	return func(rfs *RuledFileSystem) {

		changePrefix := func(path string) string {
			path = strings.TrimPrefix(path, oldPrefix)
			return filepath.Join(newPrefix, path)
		}

		rfs.ReadPath.modificators = append(rfs.ReadPath.modificators, changePrefix)
		rfs.DeletePath.modificators = append(rfs.DeletePath.modificators, changePrefix)
		rfs.AppendPath.modificators = append(rfs.AppendPath.modificators, changePrefix)
		rfs.WritePath.modificators = append(rfs.WritePath.modificators, changePrefix)
	}
}

// unix like mount directory
// mount `/mnt/some_folder/some_mount_point/` `/mnt/home/folder/`
// read `/mnt/home/folder/readme.md` return file stores in point
// even if path `/mnt/home/folder/` is not exist,
// when read dir `/mnt/home/` returns folder as existed directory
func WithMount(pointPath, targetPath string) Option {
	return func(rfs *RuledFileSystem) {

		dispatcher := func(path string) string {
			return strings.Replace(path, pointPath, targetPath, 1)
		}

		rfs.ReadPath.modificators = append(rfs.ReadPath.modificators, dispatcher)
		rfs.DeletePath.modificators = append(rfs.DeletePath.modificators, dispatcher)
		rfs.AppendPath.modificators = append(rfs.AppendPath.modificators, dispatcher)
		rfs.WritePath.modificators = append(rfs.WritePath.modificators, dispatcher)

		mountFecth := func(res DirResult) DirResult {
			if res.path != filepath.Dir(pointPath) {
				return res
			}

			res.entries = append(res.entries, &VirtualDirEntry{name: filepath.Base(pointPath), isDir: true})
			return res
		}

		rfs.DirOutput.modificators = append(rfs.DirOutput.modificators, mountFecth)
	}
}

// on provided path visible and interactable only selected subpathes
func WithWhiteListVisiblePaths(rootPath string, subPathWhitelist []string) Option {

	allowedPaths := []string{}
	for _, p := range subPathWhitelist {
		allowedPaths = append(allowedPaths, filepath.Join(rootPath, p))
	}

	return func(rfs *RuledFileSystem) {

		// operationAttempt guard
		guard := func(path string) error {
			var ok bool

			for _, ap := range allowedPaths {
				if strings.Contains(path, ap) {
					ok = true
					break
				}
			}

			if !ok {
				return fmt.Errorf("path is not exist '%s'", path)
			}

			return nil
		}

		rfs.WritePath.validators = append(rfs.WritePath.validators, guard)
		rfs.AppendPath.validators = append(rfs.AppendPath.validators, guard)
		rfs.DeletePath.validators = append(rfs.DeletePath.validators, guard)
		rfs.ReadPath.validators = append(rfs.ReadPath.validators, guard)

		// Directory visobility
		hider := func(res DirResult) DirResult {
			if res.path != rootPath {
				return res
			}

			allowedEntries := []os.DirEntry{}

			for _, e := range res.entries {
				if slices.Contains(allowedPaths, filepath.Join(res.path, e.Name())) {
					allowedEntries = append(allowedEntries, e)
				}
			}

			res.entries = allowedEntries

			return res
		}

		rfs.DirOutput.modificators = append(rfs.DirOutput.modificators, hider)
	}
}

// clears all paths /mnt/home/.. -> /mnt/home/
func WithClearedPathOnly() Option {
	return func(rfs *RuledFileSystem) {

		cleaner := func(path string) string {
			return filepath.Clean(path)
		}

		rfs.ReadPath.modificators = append(rfs.ReadPath.modificators, cleaner)
		rfs.DeletePath.modificators = append(rfs.DeletePath.modificators, cleaner)
		rfs.AppendPath.modificators = append(rfs.AppendPath.modificators, cleaner)
		rfs.WritePath.modificators = append(rfs.WritePath.modificators, cleaner)
	}
}

func AgentAccessRuleSet(agentID agent.ID) []Option {
	return []Option{
		WithChangedGlobalPrefix("file:///", fmt.Sprintf("/agent/%s", agentID)),
		WithReadOnlyPath(fmt.Sprintf("/agent/%s/activity", agentID)),
		WithReadOnlyPath(fmt.Sprintf("/agent/%s/memory", agentID)),
		WithReadOnlyPath(fmt.Sprintf("/agent/%s/skills", agentID)),
		WithWhiteListVisiblePaths(fmt.Sprintf("/agent/%s/", agentID), []string{"/memory", "/activity", "/skills", "/shared"}),
		WithMount("/skills", fmt.Sprintf("/agents/%s/skills", agentID)),
		WithMount("/shared", fmt.Sprintf("/agents/%s/shared", agentID)),
		WithClearedPathOnly(),
	}
}

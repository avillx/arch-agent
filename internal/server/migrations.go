package server

import (
	"ard/internal/files"
	"fmt"
	"io/fs"
	"log/slog"
	"path"
)

// Generated shit

func safeApplyMigrations(root files.FileStorage) {
	defer func() {
		if err := recover(); err != nil {
			slog.Error("migration failed,", "error", err)
		}
	}()

	renameSharedToRepos(root)
	renameLogFile(root)
	moveConfigsInDir(root)
	cleanAgentFolder(root)
	cleanRootFolder(root)
}

func moveConfigsInDir(root files.FileStorage) {
	const (
		configDir     = "conf"
		configDirMode = 0755
	)

	configs := []string{
		"mcp.toml",
		"memory.toml",
		"models.toml",
		"secrets.toml",
		"tasks.toml",
	}

	if err := root.MkdirAll(configDir, configDirMode); err != nil {
		return
	}

	for _, configFile := range configs {
		dst := path.Join(configDir, configFile)
		if exists(root, dst) {
			continue
		}
		_ = moveTo(root, configFile, dst)
	}
}

func renameLogFile(root files.FileStorage) {
	const logFile = "sys.log"

	if exists(root, logFile) {
		return
	}
	_ = moveTo(root, "agent.log", logFile)
}

func renameSharedToRepos(root files.FileStorage) {
	const (
		sharedDir = "shared"
		reposDir  = "repos"
	)

	if !exists(root, sharedDir) {
		return
	}

	// no repos yet: rename the whole folder
	if !exists(root, reposDir) {
		if err := moveTo(root, sharedDir, reposDir); err == nil {
			return
		}
	}

	// repos already exist: move shared content inside
	entries, err := fs.ReadDir(root.FS(), sharedDir)
	if err != nil {
		return
	}

	for _, e := range entries {
		src := path.Join(sharedDir, e.Name())
		dst := freePath(root, path.Join(reposDir, e.Name()))
		_ = moveTo(root, src, dst)
	}

	_ = root.Remove(sharedDir)
}

func cleanRootFolder(root files.FileStorage) {
	keep := map[string]bool{
		"repos":   true,
		"conf":    true,
		"tmp":     true,
		"skills":  true,
		"sys.log": true,
	}

	entries, err := fs.ReadDir(root.FS(), ".")
	if err != nil {
		return
	}

	for _, e := range entries {
		if keep[e.Name()] || isAgentDir(root, e) {
			continue
		}
		moveToRepos(root, e.Name(), e.Name(), e.IsDir())
	}
}

func cleanAgentFolder(root files.FileStorage) {
	entries, err := fs.ReadDir(root.FS(), ".")
	if err != nil {
		return
	}

	for _, e := range entries {
		if !isAgentDir(root, e) {
			continue
		}
		moveAgentSkills(root, e.Name())
		moveAgentLeftovers(root, e.Name())
	}
}

func moveAgentSkills(root files.FileStorage, agent string) {
	const (
		skillsDir     = "skills"
		skillsDirMode = 0777
	)

	agentSkills := path.Join(agent, skillsDir)

	entries, err := fs.ReadDir(root.FS(), agentSkills)
	if err != nil {
		return
	}

	_ = root.MkdirAll(skillsDir, skillsDirMode)

	for _, e := range entries {
		src := path.Join(agentSkills, e.Name())
		dst := freePath(root, path.Join(skillsDir, e.Name()+"-moved"))
		_ = moveTo(root, src, dst)
	}

	_ = root.Remove(agentSkills)
}

func moveAgentLeftovers(root files.FileStorage, agent string) {
	keep := map[string]bool{
		"activity": true,
		"memory":   true,
		"sessions": true,
		"agent.md": true,
	}

	entries, err := fs.ReadDir(root.FS(), agent)
	if err != nil {
		return
	}

	for _, e := range entries {
		if keep[e.Name()] {
			continue
		}
		moveToRepos(root, path.Join(agent, e.Name()), agent+"-"+e.Name(), e.IsDir())
	}
}

func isAgentDir(root files.FileStorage, e fs.DirEntry) bool {
	const agentFile = "agent.md"

	if !e.IsDir() {
		return false
	}
	stat, err := root.Stat(path.Join(e.Name(), agentFile))
	return err == nil && !stat.IsDir()
}

// moveToRepos: dir -> repos/<name>-moved, file -> repos/<name>-moved/<file>
func moveToRepos(root files.FileStorage, src, name string, isDir bool) {
	const (
		reposDir     = "repos"
		reposDirMode = 0777
		dirPerm      = 0755
	)

	if err := root.MkdirAll(reposDir, reposDirMode); err != nil {
		return
	}

	if isDir {
		dst := freePath(root, path.Join(reposDir, name+"-moved"))
		_ = moveTo(root, src, dst)
		return
	}

	// files share one folder per source name
	dstDir := path.Join(reposDir, name+"-moved")
	if stat, err := root.Stat(dstDir); err == nil && !stat.IsDir() {
		dstDir = freePath(root, dstDir)
	}
	if err := root.MkdirAll(dstDir, dirPerm); err != nil {
		return
	}

	dst := freePath(root, path.Join(dstDir, path.Base(src)))
	_ = moveTo(root, src, dst)
}

// moveTo renames src to dst, creating missing parents of dst.
func moveTo(root files.FileStorage, src, dst string) error {
	const dirPerm = 0755

	if parent := path.Dir(dst); parent != "." {
		if err := root.MkdirAll(parent, dirPerm); err != nil {
			return err
		}
	}
	return root.Rename(src, dst)
}

func exists(root files.FileStorage, p string) bool {
	_, err := root.Stat(p)
	return err == nil
}

func freePath(root files.FileStorage, p string) string {
	candidate := p
	for i := 1; exists(root, candidate); i++ {
		candidate = fmt.Sprintf("%s-%d", p, i)
	}
	return candidate
}

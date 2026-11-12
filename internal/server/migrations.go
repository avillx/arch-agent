package server

import (
	"ard/internal/files"
	"ard/internal/logging"
	"fmt"
	"io/fs"
	"path/filepath"
)

func applyMigrations(root files.FileStorage) {
	renameSharedToRepos(root)
	renameLogFile(root)
	moveConfigsInDir(root)
	cleanAgentFolder(root)
	cleanRootFolder(root)
}

func moveConfigsInDir(root files.FileStorage) {
	configs := []string{
		"mcp.toml",
		"memory.toml",
		"models.toml",
		"secrets.toml",
		"tasks.toml",
	}
	_ = root.MkdirAll("conf", 0o755)

	for _, configFile := range configs {
		root.Rename(configFile, filepath.Join("conf", configFile))
	}
}

func renameLogFile(root files.FileStorage) {
	root.Rename("agent.log", "sys.log")
}

func renameSharedToRepos(root files.FileStorage) {
	root.Rename("shared", "repos")
}

func moveAgentSkills(root files.FileStorage, agent string) {
	skillsDir := filepath.Join(agent, "skills")

	entries, err := fs.ReadDir(root.FS(), skillsDir)
	if err != nil {
		return
	}

	_ = root.MkdirAll("skills", 0o755)

	for _, e := range entries {
		src := filepath.Join(skillsDir, e.Name())
		dst := freePath(root, filepath.Join("skills", e.Name()+"-moved"))
		_ = root.Rename(src, dst)
	}

	_ = root.Remove(skillsDir) // non-recursive: не удалит, если что-то не переехало
}

func freePath(root files.FileStorage, path string) string {
	candidate := path
	for i := 1; ; i++ {
		if _, err := root.Stat(candidate); err != nil {
			return candidate
		}
		candidate = fmt.Sprintf("%s-%d", path, i)
	}
}

func cleanRootFolder(root files.FileStorage) {

	rootKeep := map[string]bool{
		"repos":              true,
		"conf":               true,
		"tmp":                true,
		"skills":             true,
		logging.AgentLogFile: true,
	}

	entries, err := fs.ReadDir(root.FS(), "")
	if err != nil {
		return
	}

	for _, e := range entries {
		if rootKeep[e.Name()] || isAgentDir(root, e) {
			continue
		}
		moveToRepos(root, e.Name(), e.Name(), e.IsDir())
	}
}

func cleanAgentFolder(root files.FileStorage) {
	entries, err := fs.ReadDir(root.FS(), "")
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

func moveAgentLeftovers(root files.FileStorage, agent string) {

	agentKeep := map[string]bool{
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
		if agentKeep[e.Name()] {
			continue
		}
		moveToRepos(root, filepath.Join(agent, e.Name()), agent+"-"+e.Name(), e.IsDir())
	}
}

func isAgentDir(root files.FileStorage, e fs.DirEntry) bool {
	if !e.IsDir() {
		return false
	}
	stat, err := root.Stat(filepath.Join(e.Name(), "agent.md"))
	return err == nil && !stat.IsDir()
}

// moveToRepos: dir -> repos/<name>-moved, file -> repos/<name>-moved/<file>
func moveToRepos(root files.FileStorage, src, name string, isDir bool) {
	_ = root.MkdirAll("repos", 0o755)

	dst := freePath(root, filepath.Join("repos", name+"-moved"))
	if !isDir {
		_ = root.MkdirAll(dst, 0o755)
		dst = filepath.Join(dst, filepath.Base(src))
	}
	_ = root.Rename(src, dst)
}

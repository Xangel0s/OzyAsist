//go:build !windows

package system

import (
	"context"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
)

type UnixNavigator struct{}

func NewWindowsNavigator() OSNavigator {
	return &UnixNavigator{}
}

func (u *UnixNavigator) GetKnownFolders(_ context.Context) (KnownFolders, error) {
	home := os.Getenv("HOME")
	return KnownFolders{
		UserDesktop:     filepath.Join(home, "Desktop"),
		PublicDesktop:   "/usr/share/applications",
		Documents:       filepath.Join(home, "Documents"),
		Downloads:       filepath.Join(home, "Downloads"),
		Pictures:        filepath.Join(home, "Pictures"),
		Videos:          filepath.Join(home, "Videos"),
		AppDataLocal:    filepath.Join(home, ".local/share"),
		AppDataRoam:     filepath.Join(home, ".config"),
		ProgramFiles:    "/usr/bin",
		ProgramFilesX86: "/usr/local/bin",
	}, nil
}

func (u *UnixNavigator) GetDesktopItems(ctx context.Context) ([]DesktopItem, error) {
	folders, _ := u.GetKnownFolders(ctx)
	var items []DesktopItem
	entries, _ := os.ReadDir(folders.UserDesktop)
	for _, e := range entries {
		info, _ := e.Info()
		var sz int64
		if info != nil {
			sz = info.Size()
		}
		items = append(items, DesktopItem{
			Name: e.Name(),
			Path: filepath.Join(folders.UserDesktop, e.Name()),
			Kind: ItemKindFile,
			Size: sz,
		})
	}
	return items, nil
}

func (u *UnixNavigator) GetInstalledSoftware(_ context.Context, filter string) ([]InstalledApp, error) {
	return []InstalledApp{}, nil
}

func (u *UnixNavigator) GetActiveWindows(_ context.Context) ([]WindowInfo, error) {
	return []WindowInfo{}, nil
}

func (u *UnixNavigator) ExplorePath(_ context.Context, targetPath string, maxDepth int) (*PathNode, error) {
	if targetPath == "" || targetPath == "." {
		targetPath, _ = os.Getwd()
	}
	info, err := os.Stat(targetPath)
	if err != nil {
		return nil, err
	}
	return &PathNode{
		Name:  filepath.Base(targetPath),
		Path:  targetPath,
		IsDir: info.IsDir(),
		Size:  info.Size(),
	}, nil
}

func (u *UnixNavigator) FindFiles(_ context.Context, rootDir string, pattern string, maxResults int) ([]PathNode, error) {
	var matches []PathNode
	_ = filepath.WalkDir(rootDir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		if pattern == "" || strings.Contains(strings.ToLower(d.Name()), strings.ToLower(pattern)) {
			info, _ := d.Info()
			var sz int64
			if info != nil {
				sz = info.Size()
			}
			matches = append(matches, PathNode{
				Name:  d.Name(),
				Path:  path,
				IsDir: d.IsDir(),
				Size:  sz,
			})
			if len(matches) >= maxResults {
				return fmt.Errorf("max")
			}
		}
		return nil
	})
	return matches, nil
}

func (u *UnixNavigator) CreateDirectory(_ context.Context, path string) error {
	return os.MkdirAll(path, 0755)
}

func (u *UnixNavigator) MoveItem(_ context.Context, srcPath, dstPath string) error {
	return os.Rename(srcPath, dstPath)
}

func (u *UnixNavigator) CopyItem(_ context.Context, srcPath, dstPath string) error {
	data, err := os.ReadFile(srcPath)
	if err != nil {
		return err
	}
	return os.WriteFile(dstPath, data, 0644)
}

func (u *UnixNavigator) DeleteItem(_ context.Context, path string, _ bool) error {
	return os.RemoveAll(path)
}

func (u *UnixNavigator) OrganizeFolder(_ context.Context, folderPath, _ string) (*OrganizeResult, error) {
	return &OrganizeResult{SourcePath: folderPath}, nil
}

func (u *UnixNavigator) LaunchApplication(_ context.Context, _ string, _ []string) error {
	return nil
}

func (u *UnixNavigator) FocusWindow(_ context.Context, _ uintptr) error {
	return nil
}

func (u *UnixNavigator) KillProcess(_ context.Context, _ uint32, _ bool) error {
	return nil
}

func (u *UnixNavigator) DetectDialogs(_ context.Context, _ string) ([]DialogInfo, error) {
	return []DialogInfo{}, nil
}

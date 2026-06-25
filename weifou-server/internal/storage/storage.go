// Package storage 抽象对象存储。当前为本地磁盘实现（挂 docker 命名卷持久化）；
// 接口刻意极小，未来换腾讯云 COS 只需新增一个实现，上层 handler 不动。
package storage

import (
	"io"
	"os"
	"path/filepath"
)

// Store 把读流落盘/对象存储，返回可公开访问的相对路径（如 "voices/abc.mp3"）。
type Store interface {
	Save(subdir, name string, r io.Reader) (relPath string, err error)
}

// Local 本地磁盘实现：写到 root/subdir/name。
type Local struct{ root string }

func NewLocal(root string) *Local { return &Local{root: root} }

func (l *Local) Save(subdir, name string, r io.Reader) (string, error) {
	dir := filepath.Join(l.root, subdir)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", err
	}
	f, err := os.Create(filepath.Join(dir, name))
	if err != nil {
		return "", err
	}
	defer f.Close()
	if _, err := io.Copy(f, r); err != nil {
		return "", err
	}
	return subdir + "/" + name, nil
}

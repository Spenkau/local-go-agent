package tools

import "os"

func NewDefaultRegistry(sandboxDir string) (*Registry, error) {
	r := NewRegistry()

	list := []Tool{
		CurrentTime{},
		ReadProjectFile{FS: os.DirFS(sandboxDir), MaxSize: 64 * 1024},
	}

	for _, t := range list {
		if err := r.Register(t); err != nil {
			return nil, err
		}
	}
	return r, nil
}

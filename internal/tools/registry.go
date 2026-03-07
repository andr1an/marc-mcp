package tools

import "context"

type Registry struct {
	tools map[string]Tool
}

func NewRegistry() *Registry {
	return &Registry{tools: make(map[string]Tool)}
}

func NewRegistryWithBuiltins() (*Registry, error) {
	r := NewRegistry()
	if err := RegisterBuiltins(r); err != nil {
		return nil, err
	}
	return r, nil
}

func (r *Registry) Register(t Tool) {
	r.tools[t.Name()] = t
}

func (r *Registry) List() []ToolInfo {
	out := make([]ToolInfo, 0, len(r.tools))
	for _, t := range r.tools {
		out = append(out, ToolInfo{
			Name:        t.Name(),
			Description: t.Description(),
			InputSchema: t.InputSchema(),
		})
	}
	return out
}

func (r *Registry) Invoke(ctx context.Context, name string, input []byte) (any, error) {
	t, ok := r.tools[name]
	if !ok {
		return nil, ErrToolNotFound
	}
	return t.Invoke(ctx, input)
}

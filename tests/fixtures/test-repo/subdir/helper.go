package subdir

// Helper provides utility functions
type Helper struct {
	name string
}

// NewHelper creates a new Helper
func NewHelper(name string) *Helper {
	return &Helper{name: name}
}

// GetName returns the helper name
func (h *Helper) GetName() string {
	return h.name
}

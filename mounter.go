package arpc

type Mounter struct {
	Manager *Manager
	Mux     Mux
}

// Mount mounts the handler to the mux
func (m *Mounter) Mount(pattern string, f any) {
	m.Manager.Mount(m.Mux, pattern, f)
}

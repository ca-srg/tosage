//go:build !darwin
// +build !darwin

package di

// DarwinContainer holds Darwin-specific components (stub for non-Darwin)
type DarwinContainer struct{}

// initDaemonPlatform initializes daemon components (stub for non-Darwin)
func (c *Container) initDaemonPlatform() error {
	// No daemon support on non-Darwin platforms
	return nil
}

// GetSystrayController returns nil on non-Darwin platforms
func (c *Container) GetSystrayController() interface{} {
	return nil
}

// GetDaemonController returns nil on non-Darwin platforms  
func (c *Container) GetDaemonController() interface{} {
	return nil
}
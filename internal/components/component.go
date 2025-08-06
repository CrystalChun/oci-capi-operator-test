package components

import "sigs.k8s.io/controller-runtime/pkg/client"

type Component struct {
	Name          string
	Subcomponents SubcomponentList
}

type Subcomponent struct {
	Name     string
	Object   client.Object
	MutateFn func() error
}

type SubcomponentList []Subcomponent

// GetName returns the name of the component
func (c *Component) GetName() string {
	return c.Name
}

// GetSubcomponents returns the subcomponents of the component
func (c *Component) GetSubcomponents() SubcomponentList {
	return c.Subcomponents
}

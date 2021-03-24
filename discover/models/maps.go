package models

import (
	"fmt"
	"reflect"
	"sort"
)

// ServiceMap is a set of Service using Hostname as the unique key.
type ServiceMap map[string]*Service

// NewServiceMap creates a ServiceMap from a list of Service.
func NewServiceMap(items ...*Service) ServiceMap {
	m := ServiceMap{}
	m.Insert(items...)

	return m
}

// Insert adds items to the set.
func (m ServiceMap) Insert(items ...*Service) {
	for _, item := range items {
		// if the item does not exist already or the value is not the same
		// then add/update the entry. Otherwise do nothing.
		if !m.Has(item) || !reflect.DeepEqual(m[item.Hostname], item) {
			m[item.Hostname] = item
		}
	}
}

// Delete removes all items from the set.
func (m ServiceMap) Delete(items ...*Service) {
	for _, item := range items {
		delete(m, item.Hostname)
	}
}

// Has returns true if and only if item is contained in the set.
func (m ServiceMap) Has(item *Service) bool {
	_, exists := m[item.Hostname]

	return exists
}

// HasAll returns true if and only if all items are contained in the set.
func (m ServiceMap) HasAll(items ...*Service) bool {
	for _, item := range items {
		if !m.Has(item) {
			return false
		}
	}

	return true
}

// HasAny returns true if any items are contained in the set.
func (m ServiceMap) HasAny(items ...*Service) bool {
	for _, item := range items {
		if m.Has(item) {
			return true
		}
	}

	return false
}

// List returns the contents as a sorted Service slice.
// sorted by service.Hostname.
func (m ServiceMap) List() []*Service {
	services := make([]*Service, 0, len(m))
	for _, v := range m {
		services = append(services, v)
	}

	// sort by hostnames
	sort.SliceStable(services, func(i, j int) bool { return services[i].Hostname < services[j].Hostname })

	return services
}

// Len returns the size of the set.
func (m ServiceMap) Len() int {
	return len(m)
}

// DeploymentMap is a set of Deployment using Name with Version as the unique Key.
type DeploymentMap map[string]*Deployment

// NewDeploymentMap creates a DeploymentMap from a list of Deployment.
func NewDeploymentMap(items ...*Deployment) DeploymentMap {
	m := DeploymentMap{}
	m.Insert(items...)

	return m
}

// Insert adds items to the set.
func (m DeploymentMap) Insert(items ...*Deployment) {
	for _, item := range items {
		// if the item does not exist already or the value is not the same
		// then add/update the entry. Otherwise do nothing.
		key := fmt.Sprintf("%s:%s", item.Name, item.Version)
		if !m.Has(item) || !reflect.DeepEqual(m[key], item) {
			m[key] = item
		}
	}
}

// Delete removes all items from the set.
func (m DeploymentMap) Delete(items ...*Deployment) {
	for _, item := range items {
		delete(m, fmt.Sprintf("%s:%s", item.Name, item.Version))
	}
}

// Has returns true if and only if item is contained in the set.
func (m DeploymentMap) Has(item *Deployment) bool {
	_, exists := m[fmt.Sprintf("%s:%s", item.Name, item.Version)]

	return exists
}

// HasAll returns true if and only if all items are contained in the set.
func (m DeploymentMap) HasAll(items ...*Deployment) bool {
	for _, item := range items {
		if !m.Has(item) {
			return false
		}
	}

	return true
}

// HasAny returns true if any items are contained in the set.
func (m DeploymentMap) HasAny(items ...*Deployment) bool {
	for _, item := range items {
		if m.Has(item) {
			return true
		}
	}

	return false
}

// List returns the contents as a sorted Deployment slice.
// sorted by deployment.Name.
func (m DeploymentMap) List() []*Deployment {
	deployments := make([]*Deployment, 0, len(m))
	for _, v := range m {
		deployments = append(deployments, v)
	}

	// sort by hostnames
	sort.SliceStable(deployments, func(i, j int) bool { return deployments[i].Name < deployments[j].Name })

	return deployments
}

// Len returns the size of the set.
func (m DeploymentMap) Len() int {
	return len(m)
}

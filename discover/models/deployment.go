package models

import (
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// Deployment represents a kubernetes deployment object.
type Deployment struct {
	// Name
	Name string `json:"name"`

	// Namespace
	Namespace string `json:"namespace"`

	// CreationTimestamp
	CreationTimestamp v1.Time `json:"creation_timestamp"`

	// Version
	Version string `json:"version"`
}

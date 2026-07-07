// Package v1alpha1 contains API types for the orb-operator extension object management system.
//
// +kubebuilder:object:generate=true
// +groupName=orb.operatorframework.io
// +kubebuilder:ac:generate=true
// +kubebuilder:ac:output:package=../../applyconfigurations
package v1alpha1

//go:generate go tool controller-gen crd output:crd:dir=../../deploy/crds paths=./...
//go:generate go tool controller-gen object paths=./...

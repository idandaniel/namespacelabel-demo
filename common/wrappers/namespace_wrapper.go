package wrappers

import (
	"strings"

	"golang.org/x/exp/maps"
	v1 "k8s.io/api/core/v1"
)

type NamespaceWrapper struct {
	*v1.Namespace
}

func (n *NamespaceWrapper) getManagementLabels() map[string]string {
	protectedLabels := make(map[string]string)

	for labelKey, labelValue := range n.Labels {
		if strings.Contains(labelKey, "kubernetes.io") {
			protectedLabels[labelKey] = labelValue
		}
	}

	return protectedLabels
}

func (n *NamespaceWrapper) UpdateLabels(safe bool, newLabels map[string]string) {
	if !safe {
		n.Labels = newLabels
		return
	}

	managementLabels := n.getManagementLabels()
	maps.Copy(managementLabels, newLabels)
	n.Labels = managementLabels
}

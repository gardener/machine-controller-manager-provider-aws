package helpers

import (
	rbacapi "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func (c *Cluster) ControlClusterRolesAndRoleBindingSetup(namespace string) {
	c.RbacClient.ClusterRoles().Create(&rbacapi.ClusterRole{
		ObjectMeta: metav1.ObjectMeta{
			Name: "machine-controller-manager",
		},
		Rules: []rbacapi.PolicyRule{
			{
				APIGroups: []string{
					"machine.sapcloud.io",
				},
				Resources: []string{
					"awsmachineclasses",
					"azuremachineclasses",
					"gcpmachineclasses",
					"openstackmachineclasses",
					"alicloudmachineclasses",
					"packetmachineclasses",
					"machineclasses",
					"machinedeployments",
					"machines",
					"machinesets",
					"machines/status",
					"machinesets/status",
					"machinedeployments/status",
				},
				Verbs: []string{
					"create",
					"delete",
					"deletecollection",
					"get",
					"list",
					"patch",
					"update",
					"watch",
				},
			},
			{
				APIGroups: []string{
					"",
				},
				Resources: []string{
					"nodes",
					"nodes/status",
					"configmaps",
					"secrets",
					"endpoints",
					"events",
				},
				Verbs: []string{
					"create",
					"delete",
					"deletecollection",
					"get",
					"list",
					"patch",
					"update",
					"watch",
				},
			},
		},
	})

	c.RbacClient.ClusterRoleBindings().Create(&rbacapi.ClusterRoleBinding{
		ObjectMeta: metav1.ObjectMeta{
			Name: "machine-controller-manager",
		},
		Subjects: []rbacapi.Subject{
			{
				Kind:      "ServiceAccount",
				Name:      "default",
				Namespace: namespace,
			},
		},
		RoleRef: rbacapi.RoleRef{
			APIGroup: "rbac.authorization.k8s.io",
			Kind:     "ClusterRole",
			Name:     "machine-controller-manager",
		},
	})
}

func (t *Cluster) TargetClusterRolesAndRoleBindingSetup() {
	t.RbacClient.ClusterRoles().Create(&rbacapi.ClusterRole{
		ObjectMeta: metav1.ObjectMeta{
			Name: "machine-controller-manager",
		},
		Rules: []rbacapi.PolicyRule{
			{
				APIGroups: []string{
					"",
				},
				Resources: []string{
					"nodes",
					"endpoints",
					"replicationcontrollers",
					"pods",
				},
				Verbs: []string{
					"create",
					"delete",
					"deletecollection",
					"get",
					"list",
					"patch",
					"update",
					"watch",
				},
			},
			{
				APIGroups: []string{
					"",
				},
				Resources: []string{
					"pods/eviction",
				},
				Verbs: []string{
					"create",
				},
			},
			{
				APIGroups: []string{
					"extensions",
					"apps",
				},
				Resources: []string{
					"replicasets",
					"statefulsets",
					"daemonsets",
					"deployments",
				},
				Verbs: []string{
					"create",
					"delete",
					"deletecollection",
					"get",
					"list",
					"patch",
					"update",
					"watch",
				},
			},
			{
				APIGroups: []string{
					"policy",
				},
				Resources: []string{
					"poddisruptionbudgets",
				},
				Verbs: []string{
					"list",
					"watch",
				},
			},
			{
				APIGroups: []string{
					"",
				},
				Resources: []string{
					"persistentvolumeclaims",
					"persistentvolumes",
				},
				Verbs: []string{
					"list",
					"watch",
				},
			},
		},
	})
	t.RbacClient.ClusterRoleBindings().Create(&rbacapi.ClusterRoleBinding{
		ObjectMeta: metav1.ObjectMeta{
			Name: "machine-controller-manager",
		},
		Subjects: []rbacapi.Subject{
			{
				Kind:      "ServiceAccount",
				Name:      "default",
				Namespace: "default",
			},
		},
		RoleRef: rbacapi.RoleRef{
			APIGroup: "rbac.authorization.k8s.io",
			Kind:     "ClusterRole",
			Name:     "machine-controller-manager",
		},
	})
}

func (c *Cluster) ClusterRolesAndRoleBindingCleanup() {
	c.RbacClient.ClusterRoles().Delete("machine-controller-manager", &metav1.DeleteOptions{})
	c.RbacClient.ClusterRoleBindings().Delete("machine-controller-manager", &metav1.DeleteOptions{})
}

package server

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	authzv1 "k8s.io/api/authorization/v1"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"devops-platform/internal/api"
	"devops-platform/internal/crypto"
	"devops-platform/internal/repository"
)

const (
	// PlatformRBACLabelKey is the mandatory Kubernetes label key identifying all RBAC resources
	// created or managed by the Tekton DevOps Dashboard, matching release.yaml.
	PlatformRBACLabelKey      = "tekton-devops-dashboard"
	PlatformRBACLabelVal      = "true"
	PlatformRBACLabelSelector = "tekton-devops-dashboard=true"

	RoleNameAdmin    = "tekton-devops-admin"
	RoleNameWorker   = "tekton-devops-worker"
	RoleNameReadonly = "tekton-devops-readonly"
)

// ModulePermissionRule defines granular verb permissions for a functional resource group
type ModulePermissionRule struct {
	Get    bool `json:"get"`
	List   bool `json:"list"`
	Create bool `json:"create"`
	Update bool `json:"update"`
	Delete bool `json:"delete"`
	Run    bool `json:"run,omitempty"`
}

// RolePermissionMatrix defines the configurable permission matrix of a labeled K8s ClusterRole
type RolePermissionMatrix struct {
	Pipelines    ModulePermissionRule `json:"pipelines"`
	Tasks        ModulePermissionRule `json:"tasks"`
	PipelineRuns ModulePermissionRule `json:"pipelineruns"`
	Triggers     ModulePermissionRule `json:"triggers"`
	Secrets      ModulePermissionRule `json:"secrets"`
	Namespaces   ModulePermissionRule `json:"namespaces"`
	RBAC         struct {
		Manage           bool   `json:"manage"`
		RequiredLabelKey string `json:"required_label_key"`
		RequiredLabelVal string `json:"required_label_val"`
	} `json:"rbac"`
}

// LabeledRoleItem represents a K8s ClusterRole carrying the `tekton-devops-dashboard: "true"` label
// combined with its Namespace-level RoleBindings (or ClusterRoleBinding for "*")
type LabeledRoleItem struct {
	Name            string               `json:"name"`
	DisplayName     string               `json:"display_name"`
	RoleType        string               `json:"role_type"` // "admin" (平台管理员), "worker" (工作人员), "readonly" (只读角色)
	Description     string               `json:"description"`
	K8sKind         string               `json:"k8s_kind"`
	Labels          map[string]string    `json:"labels"`
	Verbs           []string             `json:"verbs"`
	Resources       []string             `json:"resources"`
	Permissions     RolePermissionMatrix `json:"permissions"`
	BoundNamespaces []string             `json:"bound_namespaces"`
	IsBuiltin       bool                 `json:"is_builtin"`
	UpdatedAt       time.Time            `json:"updated_at"`
}

// K8sNamespaceItem represents a read-only Kubernetes Namespace queried from the cluster
type K8sNamespaceItem struct {
	Name           string            `json:"name"`
	Alias          string            `json:"alias"`
	Description    string            `json:"description"`
	Status         string            `json:"status"`
	Labels         map[string]string `json:"labels,omitempty"`
	EffectiveRole  string            `json:"effective_role"`
	RoleDisplay    string            `json:"role_display"`
	K8sRoleName    string            `json:"k8s_role_name"`
	PipelinesCount int               `json:"pipelines_count"`
	TasksCount     int               `json:"tasks_count"`
	SecretsCount   int               `json:"secrets_count"`
	TriggersCount  int               `json:"triggers_count"`
	CreatedAt      time.Time         `json:"created_at"`
}

// K8sRoleBindingItem represents a K8s RoleBinding / ClusterRoleBinding carrying `tekton-devops-dashboard: "true"`
type K8sRoleBindingItem struct {
	ID        string            `json:"id"`
	Username  string            `json:"username"`
	Namespace string            `json:"namespace"` // "*" for ClusterRoleBinding across all namespaces
	RoleName  string            `json:"role_name"` // tekton-devops-admin, tekton-devops-worker, tekton-devops-readonly
	K8sGroup  string            `json:"k8s_group"`
	Labels    map[string]string `json:"labels"`
	CreatedAt time.Time         `json:"created_at"`
}

// K8sSecretItem represents a Tekton-managed Kubernetes Secret bound to a ServiceAccount
type K8sSecretItem struct {
	ID             string    `json:"id"`
	Name           string    `json:"name"`
	Namespace      string    `json:"namespace"`
	SecretType     string    `json:"secret_type"`
	AnnotationType string    `json:"annotation_type"`
	TargetURL      string    `json:"target_url"`
	Username       string    `json:"username"`
	ServiceAccount string    `json:"service_account"`
	AutoBindSA     bool      `json:"auto_bind_sa"`
	MaskedValue    string    `json:"masked_value"`
	CreatedAt      time.Time `json:"created_at"`
}

func normalizeRoleType(role string) string {
	switch strings.ToLower(strings.TrimSpace(role)) {
	case "admin", RoleNameAdmin, "tekton-dashboard-admin", "平台管理员":
		return "admin"
	case "worker", "editor", "dev", RoleNameWorker, "tekton-dashboard-editor", "工作人员":
		return "worker"
	case "readonly", "viewer", RoleNameReadonly, "tekton-dashboard-viewer", "只读角色":
		return "readonly"
	default:
		return "worker"
	}
}

func roleTypeToClusterRoleName(roleType string) string {
	switch normalizeRoleType(roleType) {
	case "admin":
		return RoleNameAdmin
	case "worker":
		return RoleNameWorker
	default:
		return RoleNameReadonly
	}
}

func roleTypeToDisplayName(roleType string) string {
	switch normalizeRoleType(roleType) {
	case "admin":
		return "平台管理员"
	case "worker":
		return "工作人员"
	default:
		return "只读角色"
	}
}

func defaultPermissionsForRoleType(roleType string) RolePermissionMatrix {
	rt := normalizeRoleType(roleType)
	canWrite := rt == "admin" || rt == "worker"
	isAdmin := rt == "admin"

	fullModule := ModulePermissionRule{
		Get:    true,
		List:   true,
		Create: canWrite,
		Update: canWrite,
		Delete: canWrite,
		Run:    canWrite,
	}
	readOnlyNS := ModulePermissionRule{
		Get:    true,
		List:   true,
		Create: false, // Namespaces are strictly managed by K8s cluster admin
		Update: false,
		Delete: false,
	}

	var m RolePermissionMatrix
	m.Pipelines = fullModule
	m.Tasks = fullModule
	m.PipelineRuns = fullModule
	m.Triggers = fullModule
	m.Secrets = ModulePermissionRule{
		Get:    true,
		List:   true,
		Create: canWrite,
		Update: canWrite,
		Delete: canWrite,
	}
	m.Namespaces = readOnlyNS
	m.RBAC.Manage = isAdmin
	m.RBAC.RequiredLabelKey = PlatformRBACLabelKey
	m.RBAC.RequiredLabelVal = PlatformRBACLabelVal
	return m
}

func compileMatrixToK8sPolicyRules(m RolePermissionMatrix) ([]rbacv1.PolicyRule, []string, []string) {
	rules := []rbacv1.PolicyRule{
		{
			APIGroups: []string{""},
			Resources: []string{"namespaces", "pods", "pods/log", "events"},
			Verbs:     []string{"get", "list", "watch"},
		},
	}

	verbSet := map[string]bool{"get": true, "list": true, "watch": true}
	resList := []string{"namespaces", "pods/log"}

	addModuleRule := func(apiGroup string, resources []string, rule ModulePermissionRule) {
		var verbs []string
		if rule.Get || rule.List {
			verbs = append(verbs, "get", "list", "watch")
		}
		if rule.Create || rule.Run {
			verbs = append(verbs, "create")
			verbSet["create"] = true
		}
		if rule.Update {
			verbs = append(verbs, "update", "patch")
			verbSet["update"] = true
			verbSet["patch"] = true
		}
		if rule.Delete {
			verbs = append(verbs, "delete")
			verbSet["delete"] = true
		}
		if len(verbs) > 0 {
			rules = append(rules, rbacv1.PolicyRule{
				APIGroups: []string{apiGroup},
				Resources: resources,
				Verbs:     verbs,
			})
			resList = append(resList, resources...)
		}
	}

	addModuleRule("tekton.dev", []string{"pipelines", "pipelineruns"}, m.Pipelines)
	addModuleRule("tekton.dev", []string{"tasks", "taskruns", "stepactions", "customruns"}, m.Tasks)
	addModuleRule("triggers.tekton.dev", []string{"eventlisteners", "triggers", "triggerbindings", "clustertriggerbindings", "triggertemplates", "interceptors", "clusterinterceptors"}, m.Triggers)
	addModuleRule("", []string{"secrets", "serviceaccounts", "configmaps", "persistentvolumeclaims"}, m.Secrets)

	if m.RBAC.Manage {
		rules = append(rules, rbacv1.PolicyRule{
			APIGroups: []string{"rbac.authorization.k8s.io"},
			Resources: []string{"roles", "clusterroles", "rolebindings", "clusterrolebindings"},
			Verbs:     []string{"get", "list", "watch", "create", "update", "patch", "delete", "bind", "escalate"},
		})
		resList = append(resList, "roles", "clusterroles", "rolebindings", "clusterrolebindings")
	}

	var allVerbs []string
	for _, v := range []string{"get", "list", "watch", "create", "update", "patch", "delete"} {
		if verbSet[v] {
			allVerbs = append(allVerbs, v)
		}
	}
	return rules, allVerbs, resList
}

// ensureDefaultRBACAndNamespaces seeds the 3 release.yaml roles, default admin binding, read-only namespaces, and default Triggers
func (s *Server) ensureDefaultRBACAndNamespaces(ctx context.Context) {
	// 1. Normalize any legacy role values (initial state only has 'admin' user created by EnsureDefaultAdmin)
	_, _ = s.db.ExecContext(ctx, `UPDATE users SET role = 'worker' WHERE role IN ('editor', 'dev')`)
	_, _ = s.db.ExecContext(ctx, `UPDATE users SET role = 'readonly' WHERE role = 'viewer'`)

	// 2. Seed read-only cluster namespaces mirror for offline/fallback mode
	defaultNS := []struct {
		name, alias, desc string
	}{
		{"default", "默认空间 (default)", "Kubernetes 集群默认命名空间"},
		{"tekton-pipelines", "Tekton 核心控制器空间", "Tekton Pipelines / Triggers / Dashboard 组件部署命名空间"},
	}
	for _, ns := range defaultNS {
		var exists int
		_ = s.db.QueryRowContext(ctx, repository.Rebind(s.cfg.Database.Driver, `SELECT count(*) FROM k8s_namespaces WHERE name = ?`), ns.name).Scan(&exists)
		if exists == 0 {
			_, _ = s.db.ExecContext(ctx, repository.Rebind(s.cfg.Database.Driver,
				`INSERT INTO k8s_namespaces (name, alias, description, status, created_at) VALUES (?, ?, ?, 'Active', ?)`),
				ns.name, ns.alias, ns.desc, time.Now().UTC())
		}
	}

	// 3. Seed the 3 built-in release.yaml roles carrying `tekton-devops-dashboard: "true"`
	builtinRoles := []struct {
		name, display, rType, desc string
	}{
		{
			RoleNameAdmin,
			"平台管理员",
			"admin",
			"全局最高管理权限",
		},
		{
			RoleNameWorker,
			"工作人员",
			"worker",
			"命名空间内资源读写与执行权限",
		},
		{
			RoleNameReadonly,
			"只读角色",
			"readonly",
			"命名空间内只读查看权限",
		},
	}

	for _, br := range builtinRoles {
		var exists int
		_ = s.db.QueryRowContext(ctx, repository.Rebind(s.cfg.Database.Driver,
			`SELECT count(*) FROM k8s_labeled_roles WHERE name = ?`), br.name).Scan(&exists)
		if exists == 0 {
			permBytes, _ := json.Marshal(defaultPermissionsForRoleType(br.rType))
			_, _ = s.db.ExecContext(ctx, repository.Rebind(s.cfg.Database.Driver,
				`INSERT INTO k8s_labeled_roles (name, display_name, role_type, description, label_key, label_value, permissions_json, is_builtin, updated_at)
				 VALUES (?, ?, ?, ?, ?, ?, ?, 1, ?)`),
				br.name, br.display, br.rType, br.desc, PlatformRBACLabelKey, PlatformRBACLabelVal, string(permBytes), time.Now().UTC())
		} else {
			_, _ = s.db.ExecContext(ctx, repository.Rebind(s.cfg.Database.Driver,
				`UPDATE k8s_labeled_roles SET description = ? WHERE name = ? AND is_builtin = 1`),
				br.desc, br.name)
		}
	}

	// 4. Normalize legacy role names in k8s_role_bindings and seed only admin binding if empty
	_, _ = s.db.ExecContext(ctx, `UPDATE k8s_role_bindings SET role_name = 'tekton-devops-admin', k8s_group = 'tekton-devops:admin' WHERE role_name = 'tekton-dashboard-admin'`)
	_, _ = s.db.ExecContext(ctx, `UPDATE k8s_role_bindings SET role_name = 'tekton-devops-worker', k8s_group = 'tekton-devops:worker' WHERE role_name = 'tekton-dashboard-editor'`)
	_, _ = s.db.ExecContext(ctx, `UPDATE k8s_role_bindings SET role_name = 'tekton-devops-readonly', k8s_group = 'tekton-devops:readonly' WHERE role_name = 'tekton-dashboard-viewer'`)

	var bindingCount int
	_ = s.db.QueryRowContext(ctx, `SELECT count(*) FROM k8s_role_bindings WHERE username = 'admin'`).Scan(&bindingCount)
	if bindingCount == 0 {
		_, _ = s.db.ExecContext(ctx, repository.Rebind(s.cfg.Database.Driver,
			`INSERT INTO k8s_role_bindings (id, username, namespace, role_name, k8s_group, created_at) VALUES (?, 'admin', '*', ?, 'tekton-devops:admin', ?)`),
			uuid.New().String(), RoleNameAdmin, time.Now().UTC())
	}

	// 5. Seed realistic Tekton Triggers resources if empty so all 6 Triggers pages work out of the box
	s.ensureDefaultTriggersResources(ctx)

	// 6. Reconcile labeled ClusterRoles & admin binding to real K8s cluster if connected
	if s.k8sBundle != nil && s.k8sBundle.Clientset != nil {
		go s.reconcileLabeledK8sRBAC(context.Background())
	}
}

func (s *Server) reconcileLabeledK8sRBAC(ctx context.Context) {
	if s.k8sBundle == nil || s.k8sBundle.Clientset == nil {
		return
	}
	roles := s.listLabeledRolesInternal(ctx)
	crClient := s.k8sBundle.Clientset.RbacV1().ClusterRoles()
	for _, r := range roles {
		policyRules, _, _ := compileMatrixToK8sPolicyRules(r.Permissions)
		permBytes, _ := json.Marshal(r.Permissions)
		boundNS := r.BoundNamespaces
		if len(boundNS) == 0 {
			if r.RoleType == "admin" || r.Name == RoleNameAdmin {
				boundNS = []string{"*"}
			} else {
				boundNS = []string{"default"}
			}
		}
		cr := &rbacv1.ClusterRole{
			ObjectMeta: metav1.ObjectMeta{
				Name: r.Name,
				Labels: map[string]string{
					"app.kubernetes.io/part-of":       "tekton-dashboard",
					"app.kubernetes.io/managed-by":    "tekton-devops-dashboard",
					PlatformRBACLabelKey:              PlatformRBACLabelVal,
					"rbac.tekton-devops.io/role-type": r.RoleType,
				},
				Annotations: map[string]string{
					"rbac.tekton-devops.io/display-name":     r.DisplayName,
					"rbac.tekton-devops.io/description":      r.Description,
					"rbac.tekton-devops.io/permissions":      string(permBytes),
					"rbac.tekton-devops.io/bound-namespaces": strings.Join(boundNS, ","),
				},
			},
			Rules: policyRules,
		}
		if existing, err := crClient.Get(ctx, r.Name, metav1.GetOptions{}); err == nil && existing != nil {
			// Only update if it carries our mandatory label
			if existing.Labels[PlatformRBACLabelKey] == PlatformRBACLabelVal {
				if existing.Annotations["rbac.tekton-devops.io/bound-namespaces"] != "" {
					cr.Annotations["rbac.tekton-devops.io/bound-namespaces"] = existing.Annotations["rbac.tekton-devops.io/bound-namespaces"]
					boundNS = strings.Split(existing.Annotations["rbac.tekton-devops.io/bound-namespaces"], ",")
				}
				cr.ResourceVersion = existing.ResourceVersion
				_, _ = crClient.Update(ctx, cr, metav1.UpdateOptions{})
			}
		} else {
			_, _ = crClient.Create(ctx, cr, metav1.CreateOptions{})
		}
		s.reconcileRoleNamespaceBindings(ctx, r.Name, r.RoleType, boundNS)
	}
}

// getRoleSubjects gathers all Subject entries (Group + assigned Users) for a given ClusterRole
func (s *Server) getRoleSubjects(ctx context.Context, roleName, roleType string) []rbacv1.Subject {
	rType := normalizeRoleType(roleType)
	groupName := "tekton-devops:" + roleName
	subjects := []rbacv1.Subject{
		{
			Kind:     "Group",
			Name:     groupName,
			APIGroup: "rbac.authorization.k8s.io",
		},
		{
			Kind:     "Group",
			Name:     "tekton-devops:" + rType,
			APIGroup: "rbac.authorization.k8s.io",
		},
	}
	seenUsers := make(map[string]bool)
	if roleName == RoleNameAdmin || rType == "admin" {
		subjects = append(subjects, rbacv1.Subject{
			Kind:     "User",
			Name:     "admin",
			APIGroup: "rbac.authorization.k8s.io",
		})
		seenUsers["admin"] = true
	}

	rows, err := s.db.QueryContext(ctx, repository.Rebind(s.cfg.Database.Driver,
		`SELECT DISTINCT username FROM k8s_role_bindings WHERE role_name = ?`), roleName)
	if err == nil {
		defer rows.Close()
		for rows.Next() {
			var uName string
			if rows.Scan(&uName) == nil && uName != "" && !seenUsers[uName] {
				seenUsers[uName] = true
				subjects = append(subjects, rbacv1.Subject{
					Kind:     "User",
					Name:     uName,
					APIGroup: "rbac.authorization.k8s.io",
				})
			}
		}
	}
	return subjects
}

// reconcileRoleNamespaceBindings implements the "ClusterRole + multiple Namespace-level RoleBindings" pattern:
// For each target namespace assigned to a role, it creates a RoleBinding in that namespace referencing the ClusterRole.
// If the role is granted "*" (all namespaces), it creates a ClusterRoleBinding referencing the ClusterRole.
func (s *Server) reconcileRoleNamespaceBindings(ctx context.Context, roleName, roleType string, targetNamespaces []string) []string {
	rType := normalizeRoleType(roleType)
	cleanSet := make(map[string]bool)
	var cleanNS []string
	hasWildcard := rType == "admin" || roleName == RoleNameAdmin
	for _, raw := range targetNamespaces {
		ns := strings.TrimSpace(raw)
		if ns == "" {
			continue
		}
		if ns == "*" || ns == "ALL_NAMESPACES" {
			hasWildcard = true
		} else if !cleanSet[ns] {
			cleanSet[ns] = true
			cleanNS = append(cleanNS, ns)
		}
	}
	if hasWildcard {
		cleanNS = []string{"*"}
	} else if len(cleanNS) == 0 {
		cleanNS = []string{"default"}
	}

	// Update role-user bindings in local fallback store so offline tests reflect the role's namespaces
	var assignedUsers []string
	uRows, err := s.db.QueryContext(ctx, repository.Rebind(s.cfg.Database.Driver,
		`SELECT DISTINCT username FROM k8s_role_bindings WHERE role_name = ?`), roleName)
	if err == nil {
		for uRows.Next() {
			var u string
			if uRows.Scan(&u) == nil && u != "" {
				assignedUsers = append(assignedUsers, u)
			}
		}
		uRows.Close()
	}
	if (roleName == RoleNameAdmin || rType == "admin") && len(assignedUsers) == 0 {
		assignedUsers = []string{"admin"}
	}
	_, _ = s.db.ExecContext(ctx, repository.Rebind(s.cfg.Database.Driver,
		`DELETE FROM k8s_role_bindings WHERE role_name = ?`), roleName)
	k8sGroup := "tekton-devops:" + roleName
	for _, uName := range assignedUsers {
		for _, ns := range cleanNS {
			_, _ = s.db.ExecContext(ctx, repository.Rebind(s.cfg.Database.Driver,
				`INSERT INTO k8s_role_bindings (id, username, namespace, role_name, k8s_group, created_at) VALUES (?, ?, ?, ?, ?, ?)`),
				uuid.New().String(), uName, ns, roleName, k8sGroup, time.Now().UTC())
		}
	}
	// Also record a role-scope sentinel row (username = '@role') so the role's bound namespaces persist even with 0 users
	for _, ns := range cleanNS {
		_, _ = s.db.ExecContext(ctx, repository.Rebind(s.cfg.Database.Driver,
			`INSERT INTO k8s_role_bindings (id, username, namespace, role_name, k8s_group, created_at) VALUES (?, '@role', ?, ?, ?, ?)`),
			uuid.New().String(), ns, roleName, k8sGroup, time.Now().UTC())
	}

	if s.k8sBundle == nil || s.k8sBundle.Clientset == nil {
		return cleanNS
	}

	bindingName := fmt.Sprintf("tekton-devops-role-%s-binding", roleName)
	labels := map[string]string{
		"app.kubernetes.io/part-of":       "tekton-dashboard",
		"app.kubernetes.io/managed-by":    "tekton-devops-dashboard",
		PlatformRBACLabelKey:              PlatformRBACLabelVal,
		"rbac.tekton-devops.io/role-name": roleName,
	}
	subjects := s.getRoleSubjects(ctx, roleName, rType)
	roleRef := rbacv1.RoleRef{
		APIGroup: "rbac.authorization.k8s.io",
		Kind:     "ClusterRole",
		Name:     roleName,
	}

	// 1. Clean up existing RoleBindings / ClusterRoleBindings for this roleName
	if rbList, lErr := s.k8sBundle.Clientset.RbacV1().RoleBindings("").List(ctx, metav1.ListOptions{
		LabelSelector: fmt.Sprintf("%s=%s,rbac.tekton-devops.io/role-name=%s", PlatformRBACLabelKey, PlatformRBACLabelVal, roleName),
	}); lErr == nil && rbList != nil {
		for _, rb := range rbList.Items {
			_ = s.k8sBundle.Clientset.RbacV1().RoleBindings(rb.Namespace).Delete(ctx, rb.Name, metav1.DeleteOptions{})
		}
	}
	if crbList, lErr := s.k8sBundle.Clientset.RbacV1().ClusterRoleBindings().List(ctx, metav1.ListOptions{
		LabelSelector: fmt.Sprintf("%s=%s,rbac.tekton-devops.io/role-name=%s", PlatformRBACLabelKey, PlatformRBACLabelVal, roleName),
	}); lErr == nil && crbList != nil {
		for _, crb := range crbList.Items {
			_ = s.k8sBundle.Clientset.RbacV1().ClusterRoleBindings().Delete(ctx, crb.Name, metav1.DeleteOptions{})
		}
	}

	// 2. Create either ClusterRoleBinding (for "*") or Namespace-level RoleBinding per target namespace
	if hasWildcard {
		crb := &rbacv1.ClusterRoleBinding{
			ObjectMeta: metav1.ObjectMeta{
				Name:   bindingName,
				Labels: labels,
			},
			Subjects: subjects,
			RoleRef:  roleRef,
		}
		_ = s.k8sBundle.Clientset.RbacV1().ClusterRoleBindings().Delete(ctx, bindingName, metav1.DeleteOptions{})
		_, _ = s.k8sBundle.Clientset.RbacV1().ClusterRoleBindings().Create(ctx, crb, metav1.CreateOptions{})
	} else {
		for _, ns := range cleanNS {
			rb := &rbacv1.RoleBinding{
				ObjectMeta: metav1.ObjectMeta{
					Name:      bindingName,
					Namespace: ns,
					Labels:    labels,
				},
				Subjects: subjects,
				RoleRef:  roleRef,
			}
			_ = s.k8sBundle.Clientset.RbacV1().RoleBindings(ns).Delete(ctx, bindingName, metav1.DeleteOptions{})
			_, _ = s.k8sBundle.Clientset.RbacV1().RoleBindings(ns).Create(ctx, rb, metav1.CreateOptions{})
		}
	}
	return cleanNS
}

// getRoleBoundNamespaces queries Kubernetes RoleBindings / ClusterRoleBindings to find which namespaces a ClusterRole is bound to
func (s *Server) getRoleBoundNamespaces(ctx context.Context, roleName, roleType string, cr *rbacv1.ClusterRole) []string {
	rType := normalizeRoleType(roleType)
	if roleName == RoleNameAdmin || rType == "admin" {
		return []string{"*"}
	}

	if s.k8sBundle != nil && s.k8sBundle.Clientset != nil {
		var nsList []string
		seen := make(map[string]bool)

		if crbList, err := s.k8sBundle.Clientset.RbacV1().ClusterRoleBindings().List(ctx, metav1.ListOptions{
			LabelSelector: PlatformRBACLabelSelector,
		}); err == nil && crbList != nil {
			for _, crb := range crbList.Items {
				if crb.RoleRef.Name == roleName {
					return []string{"*"}
				}
			}
		}

		if rbList, err := s.k8sBundle.Clientset.RbacV1().RoleBindings("").List(ctx, metav1.ListOptions{
			LabelSelector: PlatformRBACLabelSelector,
		}); err == nil && rbList != nil {
			for _, rb := range rbList.Items {
				if rb.RoleRef.Name == roleName && !seen[rb.Namespace] {
					seen[rb.Namespace] = true
					nsList = append(nsList, rb.Namespace)
				}
			}
		}
		if len(nsList) > 0 {
			return nsList
		}
	}

	if cr != nil && cr.Annotations["rbac.tekton-devops.io/bound-namespaces"] != "" {
		var list []string
		for _, p := range strings.Split(cr.Annotations["rbac.tekton-devops.io/bound-namespaces"], ",") {
			if trimmed := strings.TrimSpace(p); trimmed != "" {
				list = append(list, trimmed)
			}
		}
		if len(list) > 0 {
			return list
		}
	}

	// Fallback query from local k8s_role_bindings
	rows, err := s.db.QueryContext(ctx, repository.Rebind(s.cfg.Database.Driver,
		`SELECT DISTINCT namespace FROM k8s_role_bindings WHERE role_name = ?`), roleName)
	if err == nil {
		defer rows.Close()
		var list []string
		for rows.Next() {
			var ns string
			if rows.Scan(&ns) == nil && ns != "" {
				if ns == "*" {
					return []string{"*"}
				}
				list = append(list, ns)
			}
		}
		if len(list) > 0 {
			return list
		}
	}

	return []string{"default"}
}

func parseK8sPolicyRulesToMatrix(rules []rbacv1.PolicyRule, roleType string) RolePermissionMatrix {
	m := defaultPermissionsForRoleType(roleType)
	// Reset module flags so we derive them directly from the K8s ClusterRole's PolicyRules
	m.Pipelines = ModulePermissionRule{}
	m.Tasks = ModulePermissionRule{}
	m.PipelineRuns = ModulePermissionRule{}
	m.Triggers = ModulePermissionRule{}
	m.Secrets = ModulePermissionRule{}

	applyRuleToModule := func(mod *ModulePermissionRule, verbs []string) {
		for _, v := range verbs {
			switch v {
			case "*":
				mod.Get, mod.List, mod.Create, mod.Update, mod.Delete, mod.Run = true, true, true, true, true, true
			case "get", "watch":
				mod.Get = true
			case "list":
				mod.List = true
				mod.Get = true
			case "create":
				mod.Create = true
				mod.Run = true
			case "update", "patch":
				mod.Update = true
			case "delete", "deletecollection":
				mod.Delete = true
			}
		}
	}

	hasResource := func(list []string, targets ...string) bool {
		for _, item := range list {
			if item == "*" {
				return true
			}
			for _, t := range targets {
				if item == t {
					return true
				}
			}
		}
		return false
	}

	hasGroup := func(groups []string, target string) bool {
		for _, g := range groups {
			if g == "*" || g == target {
				return true
			}
		}
		return false
	}

	for _, pr := range rules {
		if hasGroup(pr.APIGroups, "tekton.dev") {
			if hasResource(pr.Resources, "pipelines") {
				applyRuleToModule(&m.Pipelines, pr.Verbs)
			}
			if hasResource(pr.Resources, "pipelineruns") {
				applyRuleToModule(&m.PipelineRuns, pr.Verbs)
			}
			if hasResource(pr.Resources, "tasks", "taskruns", "stepactions", "customruns") {
				applyRuleToModule(&m.Tasks, pr.Verbs)
			}
		}
		if hasGroup(pr.APIGroups, "triggers.tekton.dev") {
			if hasResource(pr.Resources, "eventlisteners", "triggers", "triggerbindings", "triggertemplates") {
				applyRuleToModule(&m.Triggers, pr.Verbs)
			}
		}
		if hasGroup(pr.APIGroups, "") {
			if hasResource(pr.Resources, "secrets") {
				applyRuleToModule(&m.Secrets, pr.Verbs)
			}
		}
		if hasGroup(pr.APIGroups, "rbac.authorization.k8s.io") {
			if hasResource(pr.Resources, "clusterroles", "rolebindings", "clusterrolebindings") {
				for _, v := range pr.Verbs {
					if v == "*" || v == "create" || v == "update" || v == "bind" {
						m.RBAC.Manage = true
					}
				}
			}
		}
	}
	return m
}

func clusterRoleToLabeledRoleItem(cr rbacv1.ClusterRole) LabeledRoleItem {
	rType := cr.Labels["rbac.tekton-devops.io/role-type"]
	if rType == "" {
		if cr.Name == RoleNameAdmin {
			rType = "admin"
		} else if cr.Name == RoleNameReadonly {
			rType = "readonly"
		} else {
			rType = "worker"
		}
	}
	rType = normalizeRoleType(rType)

	displayName := cr.Annotations["rbac.tekton-devops.io/display-name"]
	if displayName == "" {
		if cr.Name == RoleNameAdmin || cr.Name == RoleNameWorker || cr.Name == RoleNameReadonly {
			displayName = roleTypeToDisplayName(rType)
		} else {
			displayName = cr.Name
		}
	}
	desc := cr.Annotations["rbac.tekton-devops.io/description"]
	if desc == "" || strings.Contains(desc, "拥有平台全部流水线") || strings.Contains(desc, "在指定命名空间内拥有流水线") || strings.Contains(desc, "仅具备授权命名空间内流水线") || strings.Contains(desc, "受管 K8s ClusterRole") {
		switch cr.Name {
		case RoleNameAdmin:
			desc = "全局最高管理权限"
		case RoleNameWorker:
			desc = "命名空间内资源读写与执行权限"
		case RoleNameReadonly:
			desc = "命名空间内只读查看权限"
		default:
			if strings.Contains(desc, "受管 K8s ClusterRole") {
				desc = "自定义角色"
			}
		}
	}

	var matrix RolePermissionMatrix
	if rawJSON := cr.Annotations["rbac.tekton-devops.io/permissions"]; rawJSON != "" {
		if json.Unmarshal([]byte(rawJSON), &matrix) != nil {
			matrix = parseK8sPolicyRulesToMatrix(cr.Rules, rType)
		}
	} else {
		matrix = parseK8sPolicyRulesToMatrix(cr.Rules, rType)
	}
	matrix.RBAC.RequiredLabelKey = PlatformRBACLabelKey
	matrix.RBAC.RequiredLabelVal = PlatformRBACLabelVal

	_, verbs, resources := compileMatrixToK8sPolicyRules(matrix)
	isBuiltin := cr.Name == RoleNameAdmin || cr.Name == RoleNameWorker || cr.Name == RoleNameReadonly
	updatedAt := cr.CreationTimestamp.Time
	if updatedAt.IsZero() {
		updatedAt = time.Now().UTC()
	}

	return LabeledRoleItem{
		Name:        cr.Name,
		DisplayName: displayName,
		RoleType:    rType,
		Description: desc,
		K8sKind:     "ClusterRole",
		Labels:      cr.Labels,
		Verbs:       verbs,
		Resources:   resources,
		Permissions: matrix,
		IsBuiltin:   isBuiltin,
		UpdatedAt:   updatedAt,
	}
}

func (s *Server) listLabeledRolesInternal(ctx context.Context) []LabeledRoleItem {
	// Directly query Kubernetes Cluster API for ClusterRoles carrying label `tekton-devops-dashboard=true`
	if s.k8sBundle != nil && s.k8sBundle.Clientset != nil {
		crList, err := s.k8sBundle.Clientset.RbacV1().ClusterRoles().List(ctx, metav1.ListOptions{
			LabelSelector: PlatformRBACLabelSelector,
		})
		if err == nil && crList != nil && len(crList.Items) > 0 {
			list := make([]LabeledRoleItem, 0, len(crList.Items))
			for _, cr := range crList.Items {
				if cr.Labels[PlatformRBACLabelKey] != PlatformRBACLabelVal {
					continue
				}
				crCopy := cr
				item := clusterRoleToLabeledRoleItem(crCopy)
				item.BoundNamespaces = s.getRoleBoundNamespaces(ctx, item.Name, item.RoleType, &crCopy)
				list = append(list, item)
			}
			return list
		}
	}

	// Fallback for unit-test / disconnected environment without a live K8s API server
	rows, err := s.db.QueryContext(ctx,
		`SELECT name, display_name, role_type, description, label_key, label_value, permissions_json, is_builtin, updated_at FROM k8s_labeled_roles ORDER BY is_builtin DESC, updated_at ASC`)
	if err != nil {
		return nil
	}
	defer rows.Close()

	var list []LabeledRoleItem
	for rows.Next() {
		var item LabeledRoleItem
		var lKey, lVal, permJSON string
		var isBuiltinInt int
		if err := rows.Scan(&item.Name, &item.DisplayName, &item.RoleType, &item.Description, &lKey, &lVal, &permJSON, &isBuiltinInt, &item.UpdatedAt); err == nil {
			if lKey != PlatformRBACLabelKey || lVal != PlatformRBACLabelVal {
				continue
			}
			item.K8sKind = "ClusterRole"
			item.IsBuiltin = isBuiltinInt != 0
			item.Labels = map[string]string{
				PlatformRBACLabelKey:           PlatformRBACLabelVal,
				"app.kubernetes.io/managed-by": "tekton-devops-dashboard",
			}
			var matrix RolePermissionMatrix
			if json.Unmarshal([]byte(permJSON), &matrix) != nil {
				matrix = defaultPermissionsForRoleType(item.RoleType)
			}
			matrix.RBAC.RequiredLabelKey = PlatformRBACLabelKey
			matrix.RBAC.RequiredLabelVal = PlatformRBACLabelVal
			item.Permissions = matrix
			_, item.Verbs, item.Resources = compileMatrixToK8sPolicyRules(matrix)
			item.BoundNamespaces = s.getRoleBoundNamespaces(ctx, item.Name, item.RoleType, nil)
			list = append(list, item)
		}
	}
	return list
}

func (s *Server) getRolePermissionsByName(ctx context.Context, roleName, fallbackType string) RolePermissionMatrix {
	// Directly query Kubernetes ClusterRole by name
	if s.k8sBundle != nil && s.k8sBundle.Clientset != nil && roleName != "" {
		if cr, err := s.k8sBundle.Clientset.RbacV1().ClusterRoles().Get(ctx, roleName, metav1.GetOptions{}); err == nil && cr != nil {
			if cr.Labels[PlatformRBACLabelKey] == PlatformRBACLabelVal {
				return clusterRoleToLabeledRoleItem(*cr).Permissions
			}
		}
	}

	var permJSON, rType string
	err := s.db.QueryRowContext(ctx, repository.Rebind(s.cfg.Database.Driver,
		`SELECT role_type, permissions_json FROM k8s_labeled_roles WHERE name = ? AND label_key = ? AND label_value = ?`),
		roleName, PlatformRBACLabelKey, PlatformRBACLabelVal).Scan(&rType, &permJSON)
	if err == nil && permJSON != "" {
		var m RolePermissionMatrix
		if json.Unmarshal([]byte(permJSON), &m) == nil {
			m.RBAC.RequiredLabelKey = PlatformRBACLabelKey
			m.RBAC.RequiredLabelVal = PlatformRBACLabelVal
			return m
		}
	}
	return defaultPermissionsForRoleType(fallbackType)
}

// extractClaimsFromRequest parses the user's JWT from X-Platform-Token, Authorization header, or query parameter
func (s *Server) extractClaimsFromRequest(c *gin.Context) *crypto.Claims {
	if v, ok := c.Get("claims"); ok {
		if cl, ok := v.(*crypto.Claims); ok {
			return cl
		}
	}
	var tokenStr string
	if pToken := strings.TrimSpace(c.GetHeader("X-Platform-Token")); pToken != "" {
		tokenStr = strings.TrimSpace(strings.TrimPrefix(pToken, "Bearer "))
	} else {
		authHeader := c.GetHeader("Authorization")
		if strings.HasPrefix(strings.ToLower(authHeader), "bearer ") {
			tokenStr = strings.TrimSpace(authHeader[7:])
		} else if qToken := c.Query("token"); qToken != "" {
			tokenStr = qToken
		}
	}
	if tokenStr != "" {
		if claims, err := crypto.ParseToken(tokenStr, s.cfg.JWTSecret); err == nil {
			return claims
		}
	}
	return nil
}

// resolveUserNamespaceRole evaluates a user's effective role ("admin", "worker", "readonly", or "none")
// and bound ClusterRole name for a given namespace directly from Kubernetes ClusterRoleBindings & RoleBindings.
func (s *Server) resolveUserNamespaceRole(ctx context.Context, username, globalRole, namespace string) (string, string, []string, []string) {
	if username == "" {
		username = "admin"
	}
	normGlobal := normalizeRoleType(globalRole)
	if normGlobal == "admin" || username == "admin" {
		return "admin", RoleNameAdmin, []string{"tekton-devops:admin", "system:authenticated"}, []string{"*"}
	}

	roleRank := map[string]int{
		"admin":    3,
		"worker":   2,
		"readonly": 1,
	}

	// 1. Directly query Kubernetes ClusterRoleBindings and RoleBindings carrying `tekton-devops-dashboard=true`
	if s.k8sBundle != nil && s.k8sBundle.Clientset != nil {
		bestRoleType := ""
		bestRoleName := ""
		groupsMap := map[string]bool{"system:authenticated": true}
		boundSet := make(map[string]bool)
		var boundNamespaces []string
		hasBindings := false

		checkSubjects := func(subjects []rbacv1.Subject, roleRefName, ns string) {
			for _, subj := range subjects {
				isUserMatch := subj.Kind == "User" && subj.Name == username
				isGroupMatch := subj.Kind == "Group" && (subj.Name == "tekton-devops:"+roleRefName || subj.Name == "tekton-devops:"+normGlobal)
				if isUserMatch || (isGroupMatch && normalizeRoleType(roleRefName) == normGlobal) {
					hasBindings = true
					if !boundSet[ns] {
						boundSet[ns] = true
						boundNamespaces = append(boundNamespaces, ns)
					}
					rType := normalizeRoleType(roleRefName)
					groupsMap["tekton-devops:"+rType] = true
					groupsMap["tekton-devops:"+roleRefName] = true
					if ns == "*" || namespace == "" || namespace == "ALL_NAMESPACES" || ns == namespace {
						if roleRank[rType] > roleRank[bestRoleType] {
							bestRoleType = rType
							bestRoleName = roleRefName
						}
					}
				}
			}
		}

		if crbList, err := s.k8sBundle.Clientset.RbacV1().ClusterRoleBindings().List(ctx, metav1.ListOptions{
			LabelSelector: PlatformRBACLabelSelector,
		}); err == nil && crbList != nil {
			for _, crb := range crbList.Items {
				if crb.Labels[PlatformRBACLabelKey] == PlatformRBACLabelVal {
					checkSubjects(crb.Subjects, crb.RoleRef.Name, "*")
				}
			}
		}

		if rbList, err := s.k8sBundle.Clientset.RbacV1().RoleBindings("").List(ctx, metav1.ListOptions{
			LabelSelector: PlatformRBACLabelSelector,
		}); err == nil && rbList != nil {
			for _, rb := range rbList.Items {
				if rb.Labels[PlatformRBACLabelKey] == PlatformRBACLabelVal {
					checkSubjects(rb.Subjects, rb.RoleRef.Name, rb.Namespace)
				}
			}
		}

		if hasBindings {
			var groups []string
			for g := range groupsMap {
				groups = append(groups, g)
			}
			if bestRoleType != "" {
				return bestRoleType, bestRoleName, groups, boundNamespaces
			}
			return "none", RoleNameReadonly, groups, boundNamespaces
		}
	}

	// Fallback when running in disconnected / unit-test mode without a live K8s API server
	rows, err := s.db.QueryContext(ctx, repository.Rebind(s.cfg.Database.Driver,
		`SELECT namespace, role_name, k8s_group FROM k8s_role_bindings WHERE username = ?`), username)
	if err != nil {
		crName := roleTypeToClusterRoleName(normGlobal)
		return normGlobal, crName, []string{"tekton-devops:" + normGlobal, "system:authenticated"}, []string{"default"}
	}
	defer rows.Close()

	bestRoleType := ""
	bestRoleName := ""
	groupsMap := map[string]bool{"system:authenticated": true}
	var boundNamespaces []string
	hasBindings := false

	for rows.Next() {
		var ns, rName, grp string
		if err := rows.Scan(&ns, &rName, &grp); err == nil {
			hasBindings = true
			boundNamespaces = append(boundNamespaces, ns)
			if grp != "" {
				groupsMap[grp] = true
			}
			rType := normalizeRoleType(rName)
			if ns == "*" || namespace == "" || namespace == "ALL_NAMESPACES" || ns == namespace {
				if roleRank[rType] > roleRank[bestRoleType] {
					bestRoleType = rType
					bestRoleName = rName
				}
			}
		}
	}

	var groups []string
	for g := range groupsMap {
		groups = append(groups, g)
	}

	if bestRoleType != "" {
		return bestRoleType, bestRoleName, groups, boundNamespaces
	}
	if !hasBindings {
		crName := roleTypeToClusterRoleName(normGlobal)
		return normGlobal, crName, groups, []string{"*"}
	}
	return "none", RoleNameReadonly, groups, boundNamespaces
}

// k8sNativeSubjectAccessReview delegates authorization decision directly to Kubernetes API Server
// via authorization.k8s.io/v1 SubjectAccessReview (evaluating the ClusterRole + Namespace RoleBindings).
func (s *Server) k8sNativeSubjectAccessReview(ctx context.Context, username string, groups []string, namespace, apiGroup, resource, verb string) (bool, bool) {
	if s.k8sBundle == nil || s.k8sBundle.Clientset == nil {
		return false, false
	}
	sar := &authzv1.SubjectAccessReview{
		Spec: authzv1.SubjectAccessReviewSpec{
			User:   username,
			Groups: groups,
			ResourceAttributes: &authzv1.ResourceAttributes{
				Namespace: namespace,
				Verb:      verb,
				Group:     apiGroup,
				Resource:  resource,
			},
		},
	}
	res, err := s.k8sBundle.Clientset.AuthorizationV1().SubjectAccessReviews().Create(ctx, sar, metav1.CreateOptions{})
	if err != nil || res == nil {
		return false, false
	}
	return res.Status.Allowed, true
}

func httpMethodToK8sVerb(method string, isCollection bool) string {
	switch method {
	case http.MethodPost:
		return "create"
	case http.MethodPut, http.MethodPatch:
		return "update"
	case http.MethodDelete:
		return "delete"
	default:
		if isCollection {
			return "list"
		}
		return "get"
	}
}

func parseK8sGroupAndResourceFromPath(urlPath string) (string, string, bool) {
	parts := strings.Split(strings.Trim(urlPath, "/"), "/")
	if len(parts) < 2 {
		return "", "pods", true
	}
	// /api/v1/namespaces/{ns}/{resource}[/{name}] or /api/v1/{resource}
	if parts[0] == "api" && parts[1] == "v1" {
		if len(parts) >= 5 && parts[2] == "namespaces" {
			return "", parts[4], len(parts) == 5
		}
		if len(parts) >= 3 {
			return "", parts[2], len(parts) == 3
		}
	}
	// /apis/{group}/{version}/namespaces/{ns}/{resource}[/{name}] or /apis/{group}/{version}/{resource}
	if parts[0] == "apis" && len(parts) >= 4 {
		apiGroup := parts[1]
		if len(parts) >= 6 && parts[3] == "namespaces" {
			return apiGroup, parts[5], len(parts) == 6
		}
		return apiGroup, parts[3], len(parts) == 4
	}
	return "tekton.dev", "pipelines", true
}

// sanitizeAndPrepareK8sProxy strips forged Impersonate headers, delegates RBAC evaluation to Kubernetes
// (ClusterRole + Namespace RoleBinding via SubjectAccessReview), and cleans platform tokens.
func (s *Server) sanitizeAndPrepareK8sProxy(c *gin.Context) bool {
	// 1. Always strip any client-supplied Impersonate-* headers to prevent privilege escalation
	for headerName := range c.Request.Header {
		lower := strings.ToLower(headerName)
		if strings.HasPrefix(lower, "impersonate-") {
			c.Request.Header.Del(headerName)
		}
	}

	// 2. Determine target namespace from URL path
	reqNamespace := "default"
	hasExplicitNS := false
	parts := strings.Split(strings.Trim(c.Request.URL.Path, "/"), "/")
	for i, p := range parts {
		if p == "namespaces" && i+1 < len(parts) && parts[i+1] != "" {
			reqNamespace = parts[i+1]
			hasExplicitNS = true
			break
		}
	}

	claims := s.extractClaimsFromRequest(c)
	username := "admin"
	globalRole := "admin"

	if claims != nil {
		username = claims.Username
		globalRole = claims.Role
	}

	effRole, roleName, groups, _ := s.resolveUserNamespaceRole(c.Request.Context(), username, globalRole, reqNamespace)

	// 3. Enforce RBAC write protection on mutating HTTP verbs (POST, PUT, PATCH, DELETE)
	method := c.Request.Method
	isMutating := method == http.MethodPost || method == http.MethodPut || method == http.MethodPatch || method == http.MethodDelete
	if isMutating {
		// Disallow creating or deleting K8s Namespaces through the platform proxy
		if strings.HasSuffix(strings.Trim(c.Request.URL.Path, "/"), "namespaces") || (hasExplicitNS && len(parts) == 4 && parts[2] == "namespaces") {
			c.JSON(http.StatusForbidden, gin.H{
				"apiVersion": "v1",
				"kind":       "Status",
				"status":     "Failure",
				"reason":     "Forbidden",
				"code":       http.StatusForbidden,
				"message":    "职责边界限制: Kubernetes Namespace 仅由集群管理员在 K8s 平台创建与维护，DevOps 平台仅提供只读展示",
			})
			c.Abort()
			return false
		}

		apiGroup, resource, isCollection := parseK8sGroupAndResourceFromPath(c.Request.URL.Path)
		verb := httpMethodToK8sVerb(method, isCollection)

		// Delegate directly to Kubernetes native RBAC (SubjectAccessReview evaluates ClusterRole + Namespace RoleBinding)
		if allowed, evaluatedByK8s := s.k8sNativeSubjectAccessReview(c.Request.Context(), username, groups, reqNamespace, apiGroup, resource, verb); evaluatedByK8s {
			if !allowed {
				c.JSON(http.StatusForbidden, gin.H{
					"apiVersion": "v1",
					"kind":       "Status",
					"status":     "Failure",
					"reason":     "Forbidden",
					"code":       http.StatusForbidden,
					"message":    fmt.Sprintf("Kubernetes RBAC 拒绝: 用户 '%s' (角色 %s) 在命名空间 '%s' 下未绑定 %s/%s 的 %s RoleBinding 权限", username, roleName, reqNamespace, apiGroup, resource, verb),
				})
				c.Abort()
				return false
			}
		} else {
			// Offline / unit-test fallback when no live K8s API server is connected
			matrix := s.getRolePermissionsByName(c.Request.Context(), roleName, effRole)
			if effRole == "readonly" || effRole == "none" || !matrix.Pipelines.Create {
				c.JSON(http.StatusForbidden, gin.H{
					"apiVersion": "v1",
					"kind":       "Status",
					"status":     "Failure",
					"reason":     "Forbidden",
					"code":       http.StatusForbidden,
					"message":    fmt.Sprintf("K8s RBAC 权限拒绝: 用户 '%s' 在命名空间 '%s' 下的角色 (%s) 无权执行 %s 操作", username, reqNamespace, roleName, method),
				})
				c.Abort()
				return false
			}
		}
	}

	// 4. Clean platform JWT headers before forwarding to kube-apiserver
	c.Request.Header.Del("Authorization")
	c.Request.Header.Del("X-Platform-Token")
	c.Set("k8s_effective_role", effRole)
	c.Set("k8s_role_name", roleName)
	c.Set("k8s_username", username)
	c.Set("k8s_groups", groups)
	return true
}

// handleDualModeLogin authenticates user via Username + Password and maps directly to release.yaml K8s RBAC roles
func (s *Server) handleDualModeLogin(c *gin.Context) {
	var req struct {
		Username string `json:"username"`
		Password string `json:"password"`
	}
	if err := c.ShouldBindJSON(&req); err != nil || strings.TrimSpace(req.Username) == "" || strings.TrimSpace(req.Password) == "" {
		api.Error(c, http.StatusBadRequest, 40001, "请输入用户名和密码")
		return
	}

	res, err := s.authService.Login(c.Request.Context(), strings.TrimSpace(req.Username), req.Password)
	if err != nil {
		api.Error(c, http.StatusUnauthorized, 40110, "用户名或密码错误")
		return
	}

	normRole := normalizeRoleType(res.User.Role)
	_, roleName, groups, boundNS := s.resolveUserNamespaceRole(c.Request.Context(), res.User.Username, normRole, "*")
	extClaims := crypto.Claims{
		UserID:   res.User.ID,
		Username: res.User.Username,
		Role:     normRole,
		AuthMode: "password",
		Groups:   groups,
	}
	signedJWT, err := crypto.GenerateExtendedToken(extClaims, s.cfg.JWTSecret, 24*time.Hour)
	if err != nil {
		signedJWT = res.Token
	}

	api.Success(c, gin.H{
		"token":     signedJWT,
		"auth_mode": "password",
		"user": gin.H{
			"id":               res.User.ID,
			"username":         res.User.Username,
			"role":             normRole,
			"role_display":     roleTypeToDisplayName(normRole),
			"k8s_cluster_role": roleName,
			"bound_namespaces": boundNS,
			"groups":           groups,
		},
	})
}

// handleAuthPermissions returns the current user's K8s RBAC permission matrix and allowed namespaces
func (s *Server) handleAuthPermissions(c *gin.Context) {
	reqNS := strings.TrimSpace(c.Query("namespace"))
	if reqNS == "" || reqNS == "*" {
		reqNS = "ALL_NAMESPACES"
	}

	claims := s.extractClaimsFromRequest(c)
	username := "admin"
	globalRole := "admin"
	if claims != nil {
		username = claims.Username
		globalRole = claims.Role
	}

	normGlobal := normalizeRoleType(globalRole)
	effRole, roleName, groups, boundNS := s.resolveUserNamespaceRole(c.Request.Context(), username, normGlobal, reqNS)
	matrix := s.getRolePermissionsByName(c.Request.Context(), roleName, effRole)

	if effRole == "none" {
		matrix = defaultPermissionsForRoleType("readonly")
		matrix.Pipelines.Create = false
		matrix.Pipelines.Update = false
		matrix.Pipelines.Delete = false
		matrix.Pipelines.Run = false
	}

	canWrite := effRole != "readonly" && effRole != "none" && matrix.Pipelines.Create
	isAdmin := effRole == "admin" || normGlobal == "admin"

	nsList := s.listNamespacesInternal(c.Request.Context(), username, normGlobal)
	allowedNS := make([]string, 0, len(nsList))
	for _, n := range nsList {
		allowedNS = append(allowedNS, n.Name)
	}

	api.Success(c, gin.H{
		"username":           username,
		"global_role":        normGlobal,
		"role_display":       roleTypeToDisplayName(effRole),
		"namespace":          reqNS,
		"effective_role":     effRole,
		"k8s_cluster_role":   roleName,
		"rbac_label_scope":   PlatformRBACLabelSelector,
		"bound_namespaces":   boundNS,
		"groups":             groups,
		"read_only":          !canWrite,
		"is_admin":           isAdmin,
		"allowed_namespaces": allowedNS,
		"rules":              matrix,
	})
}

// listNamespacesInternal returns read-only K8s cluster namespaces filtered by the user's RBAC bindings
func (s *Server) listNamespacesInternal(ctx context.Context, username, globalRole string) []K8sNamespaceItem {
	nsMap := make(map[string]*K8sNamespaceItem)

	// 1. Load cluster namespaces from local read-only cache / seed
	rows, err := s.db.QueryContext(ctx, `SELECT name, alias, description, status, created_at FROM k8s_namespaces ORDER BY created_at ASC`)
	if err == nil {
		defer rows.Close()
		for rows.Next() {
			var item K8sNamespaceItem
			if err := rows.Scan(&item.Name, &item.Alias, &item.Description, &item.Status, &item.CreatedAt); err == nil {
				nsMap[item.Name] = &item
			}
		}
	}

	if _, ok := nsMap["default"]; !ok {
		nsMap["default"] = &K8sNamespaceItem{
			Name:        "default",
			Alias:       "默认空间 (default)",
			Description: "Kubernetes 集群默认工作命名空间",
			Status:      "Active",
			CreatedAt:   time.Now().UTC(),
		}
	}
	if _, ok := nsMap["tekton-pipelines"]; !ok {
		nsMap["tekton-pipelines"] = &K8sNamespaceItem{
			Name:        "tekton-pipelines",
			Alias:       "Tekton 核心控制器空间",
			Description: "Tekton Pipelines / Triggers / Dashboard 组件命名空间",
			Status:      "Active",
			CreatedAt:   time.Now().UTC(),
		}
	}

	// 2. Merge live namespaces from Kubernetes cluster if connected
	if s.k8sBundle != nil && s.k8sBundle.Clientset != nil {
		if k8sList, kErr := s.k8sBundle.Clientset.CoreV1().Namespaces().List(ctx, metav1.ListOptions{}); kErr == nil {
			for _, kn := range k8sList.Items {
				existing, exists := nsMap[kn.Name]
				if !exists {
					nsMap[kn.Name] = &K8sNamespaceItem{
						Name:        kn.Name,
						Alias:       kn.Annotations["dashboard.tekton.dev/alias"],
						Description: kn.Annotations["dashboard.tekton.dev/description"],
						Status:      string(kn.Status.Phase),
						Labels:      kn.Labels,
						CreatedAt:   kn.CreationTimestamp.Time,
					}
				} else {
					existing.Labels = kn.Labels
					if kn.Status.Phase != "" {
						existing.Status = string(kn.Status.Phase)
					}
				}
			}
		}
	}

	// 3. Count tasks, pipelines, secrets, and triggers per namespace
	if tasks, tErr := s.taskRepo.ListAll(ctx); tErr == nil {
		for _, t := range tasks {
			ns := t.Namespace
			if ns == "" {
				ns = "default"
			}
			if item, ok := nsMap[ns]; ok {
				item.TasksCount++
			}
		}
	}
	if pipes, pErr := s.pipelineRepo.ListAll(ctx); pErr == nil {
		for _, p := range pipes {
			ns := extractPipelineNamespace(p)
			if item, ok := nsMap[ns]; ok {
				item.PipelinesCount++
			}
		}
	}
	if secRows, sErr := s.db.QueryContext(ctx, `SELECT namespace, count(*) FROM k8s_secrets GROUP BY namespace`); sErr == nil {
		defer secRows.Close()
		for secRows.Next() {
			var ns string
			var cnt int
			if secRows.Scan(&ns, &cnt) == nil {
				if item, ok := nsMap[ns]; ok {
					item.SecretsCount = cnt
				}
			}
		}
	}
	if trigRows, trErr := s.db.QueryContext(ctx, `SELECT namespace, count(*) FROM k8s_triggers_resources GROUP BY namespace`); trErr == nil {
		defer trigRows.Close()
		for trigRows.Next() {
			var ns string
			var cnt int
			if trigRows.Scan(&ns, &cnt) == nil {
				if item, ok := nsMap[ns]; ok {
					item.TriggersCount = cnt
				}
			}
		}
	}

	normGlobal := normalizeRoleType(globalRole)
	_, _, _, boundNS := s.resolveUserNamespaceRole(ctx, username, normGlobal, "*")
	hasWildcard := normGlobal == "admin" || username == "admin"
	boundSet := make(map[string]bool)
	for _, bns := range boundNS {
		if bns == "*" {
			hasWildcard = true
		}
		boundSet[bns] = true
	}

	result := make([]K8sNamespaceItem, 0, len(nsMap))
	for _, item := range nsMap {
		if !hasWildcard && len(boundSet) > 0 && !boundSet[item.Name] {
			continue
		}
		effRole, roleName, _, _ := s.resolveUserNamespaceRole(ctx, username, normGlobal, item.Name)
		item.EffectiveRole = effRole
		item.RoleDisplay = roleTypeToDisplayName(effRole)
		item.K8sRoleName = roleName
		result = append(result, *item)
	}
	return result
}

func extractPipelineNamespace(p repository.Pipeline) string {
	if strings.HasPrefix(strings.TrimSpace(p.DSLContent), "{") {
		var meta struct {
			Namespace string `json:"namespace"`
		}
		if json.Unmarshal([]byte(p.DSLContent), &meta) == nil && meta.Namespace != "" {
			return meta.Namespace
		}
	}
	return "default"
}

func extractPipelineSpec(p repository.Pipeline) gin.H {
	if strings.HasPrefix(strings.TrimSpace(p.DSLContent), "{") {
		var wrapper struct {
			Spec gin.H `json:"spec"`
		}
		if json.Unmarshal([]byte(p.DSLContent), &wrapper) == nil && len(wrapper.Spec) > 0 {
			return wrapper.Spec
		}
	}
	return gin.H{
		"description": p.Name,
		"tasks": []gin.H{
			{
				"name": "execute",
				"taskSpec": gin.H{
					"steps": []gin.H{
						{"name": "execute", "image": "alpine:latest", "script": "echo 'Executing Tekton pipeline'"},
					},
				},
			},
		},
	}
}

// handleListK8sNamespaces returns read-only K8s namespaces from the cluster
func (s *Server) handleListK8sNamespaces(c *gin.Context) {
	claims := s.extractClaimsFromRequest(c)
	username, globalRole := "admin", "admin"
	if claims != nil {
		username = claims.Username
		globalRole = claims.Role
	}
	items := s.listNamespacesInternal(c.Request.Context(), username, globalRole)
	api.Success(c, gin.H{"items": items})
}

// handleListK8sSecrets lists Tekton-managed Kubernetes Secrets in the selected namespace
func (s *Server) handleListK8sSecrets(c *gin.Context) {
	ns := strings.TrimSpace(c.Param("namespace"))
	var rows interface {
		Next() bool
		Scan(dest ...interface{}) error
		Close() error
	}
	var err error
	if ns == "" || ns == "*" || ns == "ALL_NAMESPACES" {
		rows, err = s.db.QueryContext(c.Request.Context(),
			`SELECT id, name, namespace, secret_type, annotation_type, target_url, username, service_account, auto_bind_sa, masked_value, created_at FROM k8s_secrets ORDER BY created_at DESC`)
	} else {
		rows, err = s.db.QueryContext(c.Request.Context(), repository.Rebind(s.cfg.Database.Driver,
			`SELECT id, name, namespace, secret_type, annotation_type, target_url, username, service_account, auto_bind_sa, masked_value, created_at FROM k8s_secrets WHERE namespace = ? ORDER BY created_at DESC`), ns)
	}
	if err != nil {
		api.Error(c, http.StatusInternalServerError, 50001, err.Error())
		return
	}
	defer rows.Close()

	items := make([]K8sSecretItem, 0)
	for rows.Next() {
		var item K8sSecretItem
		var autoBindInt int
		if err := rows.Scan(&item.ID, &item.Name, &item.Namespace, &item.SecretType, &item.AnnotationType, &item.TargetURL, &item.Username, &item.ServiceAccount, &autoBindInt, &item.MaskedValue, &item.CreatedAt); err == nil {
			item.AutoBindSA = autoBindInt != 0
			items = append(items, item)
		}
	}
	api.Success(c, gin.H{"items": items})
}

// handleCreateK8sSecret creates a native Kubernetes Secret with Tekton annotations and auto-binds it to ServiceAccount
func (s *Server) handleCreateK8sSecret(c *gin.Context) {
	ns := strings.TrimSpace(c.Param("namespace"))
	if ns == "" || ns == "*" || ns == "ALL_NAMESPACES" {
		ns = "default"
	}

	var req struct {
		Name           string `json:"name"`
		Namespace      string `json:"namespace"`
		SecretType     string `json:"secret_type"`
		TargetURL      string `json:"target_url"`
		Username       string `json:"username"`
		SecretValue    string `json:"secret_value"`
		ServiceAccount string `json:"service_account"`
		AutoBindSA     *bool  `json:"auto_bind_sa"`
	}
	if err := c.ShouldBindJSON(&req); err != nil || strings.TrimSpace(req.Name) == "" || strings.TrimSpace(req.SecretValue) == "" {
		api.Error(c, http.StatusBadRequest, 40001, "Secret 名称与密钥内容不能为空")
		return
	}
	if req.Namespace != "" && req.Namespace != "ALL_NAMESPACES" && req.Namespace != "*" {
		ns = req.Namespace
	}

	claims := s.extractClaimsFromRequest(c)
	uName, gRole := "admin", "admin"
	if claims != nil {
		uName, gRole = claims.Username, claims.Role
	}
	effRole, roleName, _, _ := s.resolveUserNamespaceRole(c.Request.Context(), uName, gRole, ns)
	matrix := s.getRolePermissionsByName(c.Request.Context(), roleName, effRole)
	if effRole == "readonly" || effRole == "none" || !matrix.Secrets.Create {
		api.Error(c, http.StatusForbidden, 40301, fmt.Sprintf("RBAC 权限不足：当前角色 (%s) 在命名空间 '%s' 下无权创建 Secret", roleName, ns))
		return
	}

	name := strings.ToLower(strings.TrimSpace(req.Name))
	secType := strings.TrimSpace(req.SecretType)
	if secType == "" {
		secType = "kubernetes.io/basic-auth"
	}
	targetURL := strings.TrimSpace(req.TargetURL)
	annotationType := "tekton.dev/git-0"
	switch secType {
	case "kubernetes.io/basic-auth", "git_basic":
		secType = "kubernetes.io/basic-auth"
		annotationType = "tekton.dev/git-0"
		if targetURL == "" {
			targetURL = "https://github.com"
		}
	case "kubernetes.io/ssh-auth", "git_ssh":
		secType = "kubernetes.io/ssh-auth"
		annotationType = "tekton.dev/git-0"
		if targetURL == "" {
			targetURL = "github.com"
		}
	case "kubernetes.io/dockerconfigjson", "docker_registry":
		secType = "kubernetes.io/dockerconfigjson"
		annotationType = "tekton.dev/docker-0"
		if targetURL == "" {
			targetURL = "https://index.docker.io/v1/"
		}
	default:
		secType = "Opaque"
		annotationType = "none"
	}

	saName := strings.TrimSpace(req.ServiceAccount)
	if saName == "" {
		saName = "default"
	}
	autoBind := true
	if req.AutoBindSA != nil {
		autoBind = *req.AutoBindSA
	}

	masked := "******"
	val := strings.TrimSpace(req.SecretValue)
	if len(val) > 6 {
		masked = val[:3] + "****" + val[len(val)-2:]
	}

	if s.k8sBundle != nil && s.k8sBundle.Clientset != nil {
		annotations := map[string]string{
			"dashboard.tekton.dev/service-account": saName,
		}
		if annotationType != "none" && targetURL != "" {
			annotations[annotationType] = targetURL
		}

		stringData := map[string]string{}
		var k8sSecType corev1.SecretType
		switch secType {
		case "kubernetes.io/basic-auth":
			k8sSecType = corev1.SecretTypeBasicAuth
			stringData["username"] = req.Username
			stringData["password"] = val
		case "kubernetes.io/ssh-auth":
			k8sSecType = corev1.SecretTypeSSHAuth
			stringData["ssh-privatekey"] = val
		case "kubernetes.io/dockerconfigjson":
			k8sSecType = corev1.SecretTypeBasicAuth
			stringData["username"] = req.Username
			stringData["password"] = val
		default:
			k8sSecType = corev1.SecretTypeOpaque
			stringData["value"] = val
		}

		k8sSec := &corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      name,
				Namespace: ns,
				Labels: map[string]string{
					"app.kubernetes.io/managed-by": "tekton-devops-dashboard",
					PlatformRBACLabelKey:           PlatformRBACLabelVal,
				},
				Annotations: annotations,
			},
			Type:       k8sSecType,
			StringData: stringData,
		}
		_ = s.k8sBundle.Clientset.CoreV1().Secrets(ns).Delete(c.Request.Context(), name, metav1.DeleteOptions{})
		_, _ = s.k8sBundle.Clientset.CoreV1().Secrets(ns).Create(c.Request.Context(), k8sSec, metav1.CreateOptions{})

		if autoBind {
			if sa, saErr := s.k8sBundle.Clientset.CoreV1().ServiceAccounts(ns).Get(c.Request.Context(), saName, metav1.GetOptions{}); saErr == nil && sa != nil {
				found := false
				for _, ref := range sa.Secrets {
					if ref.Name == name {
						found = true
						break
					}
				}
				if !found {
					sa.Secrets = append(sa.Secrets, corev1.ObjectReference{Name: name})
					_, _ = s.k8sBundle.Clientset.CoreV1().ServiceAccounts(ns).Update(c.Request.Context(), sa, metav1.UpdateOptions{})
				}
			}
		}
	}

	autoBindInt := 0
	if autoBind {
		autoBindInt = 1
	}
	id := uuid.New().String()
	_, _ = s.db.ExecContext(c.Request.Context(), repository.Rebind(s.cfg.Database.Driver,
		`DELETE FROM k8s_secrets WHERE namespace = ? AND name = ?`), ns, name)
	_, err := s.db.ExecContext(c.Request.Context(), repository.Rebind(s.cfg.Database.Driver,
		`INSERT INTO k8s_secrets (id, name, namespace, secret_type, annotation_type, target_url, username, service_account, auto_bind_sa, masked_value, created_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`),
		id, name, ns, secType, annotationType, targetURL, req.Username, saName, autoBindInt, masked, time.Now().UTC())
	if err != nil {
		api.Error(c, http.StatusInternalServerError, 50001, "保存 K8s Secret 失败: "+err.Error())
		return
	}

	api.Success(c, K8sSecretItem{
		ID:             id,
		Name:           name,
		Namespace:      ns,
		SecretType:     secType,
		AnnotationType: annotationType,
		TargetURL:      targetURL,
		Username:       req.Username,
		ServiceAccount: saName,
		AutoBindSA:     autoBind,
		MaskedValue:    masked,
		CreatedAt:      time.Now().UTC(),
	})
}

// handleDeleteK8sSecret removes a Kubernetes Secret
func (s *Server) handleDeleteK8sSecret(c *gin.Context) {
	ns := strings.TrimSpace(c.Param("namespace"))
	name := strings.TrimSpace(c.Param("name"))

	claims := s.extractClaimsFromRequest(c)
	uName, gRole := "admin", "admin"
	if claims != nil {
		uName, gRole = claims.Username, claims.Role
	}
	effRole, roleName, _, _ := s.resolveUserNamespaceRole(c.Request.Context(), uName, gRole, ns)
	matrix := s.getRolePermissionsByName(c.Request.Context(), roleName, effRole)
	if effRole == "readonly" || effRole == "none" || !matrix.Secrets.Delete {
		api.Error(c, http.StatusForbidden, 40301, "RBAC 权限不足：当前角色无法删除 Secret")
		return
	}

	if s.k8sBundle != nil && s.k8sBundle.Clientset != nil && ns != "" && ns != "*" {
		_ = s.k8sBundle.Clientset.CoreV1().Secrets(ns).Delete(c.Request.Context(), name, metav1.DeleteOptions{})
	}

	if ns == "" || ns == "*" || ns == "ALL_NAMESPACES" {
		_, _ = s.db.ExecContext(c.Request.Context(), repository.Rebind(s.cfg.Database.Driver,
			`DELETE FROM k8s_secrets WHERE name = ? OR id = ?`), name, name)
	} else {
		_, _ = s.db.ExecContext(c.Request.Context(), repository.Rebind(s.cfg.Database.Driver,
			`DELETE FROM k8s_secrets WHERE (namespace = ? AND name = ?) OR id = ?`), ns, name, name)
	}
	api.Success(c, gin.H{"deleted": name})
}

// requirePlatformAdmin verifies that the caller holds `tekton-devops-admin` permissions
func (s *Server) requirePlatformAdmin(c *gin.Context) bool {
	claims := s.extractClaimsFromRequest(c)
	if claims != nil && normalizeRoleType(claims.Role) != "admin" && claims.Username != "admin" {
		api.Error(c, http.StatusForbidden, 40302, "权限不足：仅「平台管理员」允许访问与修改带有 tekton-devops-dashboard 标签的 RBAC 配置")
		return false
	}
	return true
}

// handleRBACRoles returns ONLY K8s ClusterRoles carrying label `tekton-devops-dashboard: "true"`
func (s *Server) handleRBACRoles(c *gin.Context) {
	items := s.listLabeledRolesInternal(c.Request.Context())
	api.Success(c, gin.H{
		"label_selector": PlatformRBACLabelSelector,
		"items":          items,
	})
}

// handleSaveRBACRole creates or updates a K8s ClusterRole carrying `tekton-devops-dashboard: "true"`
// Rejects any attempt to modify an existing cluster role that lacks the `tekton-devops-dashboard: "true"` label.
func (s *Server) handleSaveRBACRole(c *gin.Context) {
	if !s.requirePlatformAdmin(c) {
		return
	}

	var req struct {
		Name            string               `json:"name"`
		DisplayName     string               `json:"display_name"`
		RoleType        string               `json:"role_type"` // admin, worker, readonly
		Description     string               `json:"description"`
		Permissions     RolePermissionMatrix `json:"permissions"`
		BoundNamespaces []string             `json:"bound_namespaces"`
		Namespaces      []string             `json:"namespaces"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		api.Error(c, http.StatusBadRequest, 40001, "无效的 RBAC 角色配置参数")
		return
	}

	roleName := strings.TrimSpace(c.Param("name"))
	if roleName == "" {
		roleName = strings.ToLower(strings.TrimSpace(req.Name))
	}
	if roleName == "" {
		api.Error(c, http.StatusBadRequest, 40001, "角色名称不能为空")
		return
	}

	// Guard: If connected to real K8s cluster, verify we never overwrite a non-labeled system/cluster role
	if s.k8sBundle != nil && s.k8sBundle.Clientset != nil {
		if existing, err := s.k8sBundle.Clientset.RbacV1().ClusterRoles().Get(c.Request.Context(), roleName, metav1.GetOptions{}); err == nil && existing != nil {
			if existing.Labels[PlatformRBACLabelKey] != PlatformRBACLabelVal {
				api.Error(c, http.StatusForbidden, 40305, fmt.Sprintf("RBAC 标签边界拦截：集群中已存在角色 '%s' 但未携带 '%s: %s' 标签，严禁修改集群非平台角色", roleName, PlatformRBACLabelKey, PlatformRBACLabelVal))
				return
			}
		}
	}

	rType := normalizeRoleType(req.RoleType)
	if roleName == RoleNameAdmin {
		rType = "admin"
	} else if roleName == RoleNameWorker {
		rType = "worker"
	} else if roleName == RoleNameReadonly {
		rType = "readonly"
	}

	displayName := strings.TrimSpace(req.DisplayName)
	if displayName == "" {
		displayName = roleTypeToDisplayName(rType)
	}
	desc := strings.TrimSpace(req.Description)
	if desc == "" {
		desc = "自定义角色"
	}

	targetNamespaces := req.BoundNamespaces
	if len(targetNamespaces) == 0 {
		targetNamespaces = req.Namespaces
	}
	if len(targetNamespaces) == 0 {
		targetNamespaces = s.getRoleBoundNamespaces(c.Request.Context(), roleName, rType, nil)
	}

	// Ensure Namespaces remain strictly read-only and RBAC label scope is locked
	req.Permissions.Namespaces = ModulePermissionRule{Get: true, List: true}
	req.Permissions.RBAC.RequiredLabelKey = PlatformRBACLabelKey
	req.Permissions.RBAC.RequiredLabelVal = PlatformRBACLabelVal
	if rType == "admin" {
		req.Permissions.RBAC.Manage = true
	}

	// 1. Reconcile ClusterRole + per-Namespace RoleBindings in Kubernetes cluster
	boundNS := s.reconcileRoleNamespaceBindings(c.Request.Context(), roleName, rType, targetNamespaces)

	permBytes, _ := json.Marshal(req.Permissions)
	isBuiltin := 0
	if roleName == RoleNameAdmin || roleName == RoleNameWorker || roleName == RoleNameReadonly {
		isBuiltin = 1
	}

	policyRules, verbs, resources := compileMatrixToK8sPolicyRules(req.Permissions)

	// 2. Write directly to Kubernetes ClusterRole API with mandatory label `tekton-devops-dashboard: "true"`
	if s.k8sBundle != nil && s.k8sBundle.Clientset != nil {
		crClient := s.k8sBundle.Clientset.RbacV1().ClusterRoles()
		cr := &rbacv1.ClusterRole{
			ObjectMeta: metav1.ObjectMeta{
				Name: roleName,
				Labels: map[string]string{
					"app.kubernetes.io/part-of":       "tekton-dashboard",
					"app.kubernetes.io/managed-by":    "tekton-devops-dashboard",
					PlatformRBACLabelKey:              PlatformRBACLabelVal,
					"rbac.tekton-devops.io/role-type": rType,
				},
				Annotations: map[string]string{
					"rbac.tekton-devops.io/display-name":     displayName,
					"rbac.tekton-devops.io/description":      desc,
					"rbac.tekton-devops.io/permissions":      string(permBytes),
					"rbac.tekton-devops.io/bound-namespaces": strings.Join(boundNS, ","),
				},
			},
			Rules: policyRules,
		}
		if existing, gErr := crClient.Get(c.Request.Context(), roleName, metav1.GetOptions{}); gErr == nil && existing != nil {
			cr.ResourceVersion = existing.ResourceVersion
			if _, uErr := crClient.Update(c.Request.Context(), cr, metav1.UpdateOptions{}); uErr != nil {
				api.Error(c, http.StatusInternalServerError, 50001, "直接更新 K8s ClusterRole 失败: "+uErr.Error())
				return
			}
		} else {
			if _, cErr := crClient.Create(c.Request.Context(), cr, metav1.CreateOptions{}); cErr != nil {
				api.Error(c, http.StatusInternalServerError, 50001, "直接创建 K8s ClusterRole 失败: "+cErr.Error())
				return
			}
		}
	}

	_, _ = s.db.ExecContext(c.Request.Context(), repository.Rebind(s.cfg.Database.Driver,
		`DELETE FROM k8s_labeled_roles WHERE name = ?`), roleName)
	_, _ = s.db.ExecContext(c.Request.Context(), repository.Rebind(s.cfg.Database.Driver,
		`INSERT INTO k8s_labeled_roles (name, display_name, role_type, description, label_key, label_value, permissions_json, is_builtin, updated_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`),
		roleName, displayName, rType, desc, PlatformRBACLabelKey, PlatformRBACLabelVal, string(permBytes), isBuiltin, time.Now().UTC())

	api.Success(c, LabeledRoleItem{
		Name:            roleName,
		DisplayName:     displayName,
		RoleType:        rType,
		Description:     desc,
		K8sKind:         "ClusterRole",
		Labels: map[string]string{
			PlatformRBACLabelKey:           PlatformRBACLabelVal,
			"app.kubernetes.io/managed-by": "tekton-devops-dashboard",
		},
		Verbs:           verbs,
		Resources:       resources,
		Permissions:     req.Permissions,
		BoundNamespaces: boundNS,
		IsBuiltin:       isBuiltin == 1,
		UpdatedAt:       time.Now().UTC(),
	})
}

// handleDeleteRBACRole deletes a custom labeled K8s ClusterRole and its Namespace RoleBindings (built-in 3 roles cannot be deleted)
func (s *Server) handleDeleteRBACRole(c *gin.Context) {
	if !s.requirePlatformAdmin(c) {
		return
	}
	name := strings.TrimSpace(c.Param("name"))
	if name == RoleNameAdmin || name == RoleNameWorker || name == RoleNameReadonly {
		api.Error(c, http.StatusBadRequest, 40003, "release.yaml 预置的三大基础角色禁止删除，您可以直接修改其权限矩阵")
		return
	}
	if s.k8sBundle != nil && s.k8sBundle.Clientset != nil {
		if existing, err := s.k8sBundle.Clientset.RbacV1().ClusterRoles().Get(c.Request.Context(), name, metav1.GetOptions{}); err == nil && existing != nil {
			if existing.Labels[PlatformRBACLabelKey] != PlatformRBACLabelVal {
				api.Error(c, http.StatusForbidden, 40305, "RBAC 标签边界拦截：严禁删除不带 tekton-devops-dashboard 标签的集群角色")
				return
			}
			_ = s.k8sBundle.Clientset.RbacV1().ClusterRoles().Delete(c.Request.Context(), name, metav1.DeleteOptions{})
		}
		if rbList, lErr := s.k8sBundle.Clientset.RbacV1().RoleBindings("").List(c.Request.Context(), metav1.ListOptions{
			LabelSelector: fmt.Sprintf("%s=%s,rbac.tekton-devops.io/role-name=%s", PlatformRBACLabelKey, PlatformRBACLabelVal, name),
		}); lErr == nil && rbList != nil {
			for _, rb := range rbList.Items {
				_ = s.k8sBundle.Clientset.RbacV1().RoleBindings(rb.Namespace).Delete(c.Request.Context(), rb.Name, metav1.DeleteOptions{})
			}
		}
		if crbList, lErr := s.k8sBundle.Clientset.RbacV1().ClusterRoleBindings().List(c.Request.Context(), metav1.ListOptions{
			LabelSelector: fmt.Sprintf("%s=%s,rbac.tekton-devops.io/role-name=%s", PlatformRBACLabelKey, PlatformRBACLabelVal, name),
		}); lErr == nil && crbList != nil {
			for _, crb := range crbList.Items {
				_ = s.k8sBundle.Clientset.RbacV1().ClusterRoleBindings().Delete(c.Request.Context(), crb.Name, metav1.DeleteOptions{})
			}
		}
	}
	_, _ = s.db.ExecContext(c.Request.Context(), repository.Rebind(s.cfg.Database.Driver,
		`DELETE FROM k8s_labeled_roles WHERE name = ? AND is_builtin = 0`), name)
	_, _ = s.db.ExecContext(c.Request.Context(), repository.Rebind(s.cfg.Database.Driver,
		`DELETE FROM k8s_role_bindings WHERE role_name = ?`), name)
	api.Success(c, gin.H{"deleted": name})
}

// handleListRBACUsers lists platform users enriched with their assigned K8s ClusterRole and its RoleBinding namespaces
func (s *Server) handleListRBACUsers(c *gin.Context) {
	users, err := s.authService.ListUsers(c.Request.Context())
	if err != nil {
		api.Error(c, http.StatusInternalServerError, 50001, err.Error())
		return
	}

	enriched := make([]gin.H, 0, len(users))
	for _, u := range users {
		normRole := normalizeRoleType(u.Role)
		effRole, roleName, _, boundNS := s.resolveUserNamespaceRole(c.Request.Context(), u.Username, normRole, "*")
		roleNS := s.getRoleBoundNamespaces(c.Request.Context(), roleName, effRole, nil)
		if len(roleNS) > 0 {
			boundNS = roleNS
		}
		enriched = append(enriched, gin.H{
			"id":               u.ID,
			"username":         u.Username,
			"role":             normRole,
			"role_display":     roleTypeToDisplayName(effRole),
			"k8s_role_name":    roleName,
			"bound_namespaces": boundNS,
			"created_at":       u.CreatedAt,
		})
	}
	api.Success(c, gin.H{"items": enriched})
}

// handleCreateRBACUser creates a new user and binds them to the specified role (inheriting the role's Namespace RoleBindings)
func (s *Server) handleCreateRBACUser(c *gin.Context) {
	if !s.requirePlatformAdmin(c) {
		return
	}
	var req struct {
		Username   string   `json:"username"`
		Password   string   `json:"password"`
		Role       string   `json:"role"`       // admin, worker, readonly
		RoleName   string   `json:"role_name"`  // specific labeled ClusterRole name
		Namespace  string   `json:"namespace"`  // optional override
		Namespaces []string `json:"namespaces"` // optional override
	}
	if err := c.ShouldBindJSON(&req); err != nil || strings.TrimSpace(req.Username) == "" || strings.TrimSpace(req.Password) == "" {
		api.Error(c, http.StatusBadRequest, 40001, "用户名与登录密码不能为空")
		return
	}
	normRole := normalizeRoleType(req.Role)
	targetRoleName := strings.TrimSpace(req.RoleName)
	if targetRoleName == "" {
		targetRoleName = roleTypeToClusterRoleName(normRole)
	} else if req.Role == "" {
		normRole = normalizeRoleType(targetRoleName)
	}

	u, err := s.authService.CreateUser(c.Request.Context(), strings.TrimSpace(req.Username), req.Password, normRole)
	if err != nil {
		api.Error(c, http.StatusBadRequest, 40002, "创建用户失败: "+err.Error())
		return
	}

	// Inherit the target role's configured Namespace RoleBindings
	nsList := s.getRoleBoundNamespaces(c.Request.Context(), targetRoleName, normRole, nil)
	if len(req.Namespaces) > 0 {
		nsList = req.Namespaces
	} else if strings.TrimSpace(req.Namespace) != "" {
		nsList = []string{strings.TrimSpace(req.Namespace)}
	}

	s.syncUserRoleBindingsInternal(c.Request.Context(), u.Username, normRole, targetRoleName, nsList)
	api.Success(c, gin.H{
		"id":               u.ID,
		"username":         u.Username,
		"role":             normRole,
		"role_display":     roleTypeToDisplayName(normRole),
		"k8s_role_name":    targetRoleName,
		"bound_namespaces": nsList,
	})
}

// handleUpdateRBACUser updates an existing user's bound role (inheriting the role's Namespace RoleBindings)
func (s *Server) handleUpdateRBACUser(c *gin.Context) {
	if !s.requirePlatformAdmin(c) {
		return
	}
	id := strings.TrimSpace(c.Param("id"))
	u, err := s.authService.GetUserByID(c.Request.Context(), id)
	if err != nil || u == nil {
		api.Error(c, http.StatusNotFound, 40401, "未找到该用户")
		return
	}

	var req struct {
		Role       string   `json:"role"`
		RoleName   string   `json:"role_name"`
		Namespaces []string `json:"namespaces"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		api.Error(c, http.StatusBadRequest, 40001, "无效的请求参数")
		return
	}

	targetRoleName := strings.TrimSpace(req.RoleName)
	normRole := normalizeRoleType(req.Role)
	if targetRoleName == "" {
		targetRoleName = roleTypeToClusterRoleName(normRole)
	} else if req.Role == "" {
		normRole = normalizeRoleType(targetRoleName)
	}
	if u.Username == "admin" {
		normRole = "admin"
		targetRoleName = RoleNameAdmin
	}
	_ = s.authService.UpdateUserRole(c.Request.Context(), u.ID, normRole)

	nsList := s.getRoleBoundNamespaces(c.Request.Context(), targetRoleName, normRole, nil)
	if len(req.Namespaces) > 0 {
		nsList = req.Namespaces
	}

	s.syncUserRoleBindingsInternal(c.Request.Context(), u.Username, normRole, targetRoleName, nsList)
	api.Success(c, gin.H{
		"id":               u.ID,
		"username":         u.Username,
		"role":             normRole,
		"role_display":     roleTypeToDisplayName(normRole),
		"k8s_role_name":    targetRoleName,
		"bound_namespaces": nsList,
	})
}

func (s *Server) deleteUserK8sBindings(ctx context.Context, username string) {
	if s.k8sBundle == nil || s.k8sBundle.Clientset == nil || username == "" {
		return
	}
	if crbList, err := s.k8sBundle.Clientset.RbacV1().ClusterRoleBindings().List(ctx, metav1.ListOptions{
		LabelSelector: PlatformRBACLabelSelector,
	}); err == nil && crbList != nil {
		for _, crb := range crbList.Items {
			for _, subj := range crb.Subjects {
				if subj.Kind == "User" && subj.Name == username {
					_ = s.k8sBundle.Clientset.RbacV1().ClusterRoleBindings().Delete(ctx, crb.Name, metav1.DeleteOptions{})
					break
				}
			}
		}
	}
	if rbList, err := s.k8sBundle.Clientset.RbacV1().RoleBindings("").List(ctx, metav1.ListOptions{
		LabelSelector: PlatformRBACLabelSelector,
	}); err == nil && rbList != nil {
		for _, rb := range rbList.Items {
			for _, subj := range rb.Subjects {
				if subj.Kind == "User" && subj.Name == username {
					_ = s.k8sBundle.Clientset.RbacV1().RoleBindings(rb.Namespace).Delete(ctx, rb.Name, metav1.DeleteOptions{})
					break
				}
			}
		}
	}
}

func (s *Server) syncUserRoleBindingsInternal(ctx context.Context, username, roleType, roleName string, namespaces []string) {
	s.deleteUserK8sBindings(ctx, username)
	_, _ = s.db.ExecContext(ctx, repository.Rebind(s.cfg.Database.Driver,
		`DELETE FROM k8s_role_bindings WHERE username = ?`), username)

	k8sGroup := "tekton-devops:" + roleName
	for _, rawNS := range namespaces {
		ns := strings.TrimSpace(rawNS)
		if ns == "" {
			continue
		}
		_, _ = s.db.ExecContext(ctx, repository.Rebind(s.cfg.Database.Driver,
			`INSERT INTO k8s_role_bindings (id, username, namespace, role_name, k8s_group, created_at) VALUES (?, ?, ?, ?, ?, ?)`),
			uuid.New().String(), username, ns, roleName, k8sGroup, time.Now().UTC())
	}

	// Reconcile the target role's Namespace RoleBindings in Kubernetes so this user is included in their Subjects
	s.reconcileRoleNamespaceBindings(ctx, roleName, roleType, namespaces)
}

// handleDeleteRBACUser deletes a user and their associated K8s RoleBindings
func (s *Server) handleDeleteRBACUser(c *gin.Context) {
	if !s.requirePlatformAdmin(c) {
		return
	}
	id := strings.TrimSpace(c.Param("id"))
	u, err := s.authService.GetUserByID(c.Request.Context(), id)
	if err == nil && u != nil {
		if u.Username == "admin" {
			api.Error(c, http.StatusBadRequest, 40003, "默认平台管理员账号 admin 禁止删除")
			return
		}
		s.deleteUserK8sBindings(c.Request.Context(), u.Username)
		_, _ = s.db.ExecContext(c.Request.Context(), repository.Rebind(s.cfg.Database.Driver,
			`DELETE FROM k8s_role_bindings WHERE username = ?`), u.Username)
	}
	_ = s.authService.DeleteUser(c.Request.Context(), id)
	api.Success(c, gin.H{"deleted": id})
}

// handleListRBACBindings lists ONLY K8s RoleBindings / ClusterRoleBindings carrying `tekton-devops-dashboard: "true"` directly from the K8s cluster
func (s *Server) handleListRBACBindings(c *gin.Context) {
	ctx := c.Request.Context()
	if s.k8sBundle != nil && s.k8sBundle.Clientset != nil {
		items := make([]K8sRoleBindingItem, 0)
		if crbList, err := s.k8sBundle.Clientset.RbacV1().ClusterRoleBindings().List(ctx, metav1.ListOptions{
			LabelSelector: PlatformRBACLabelSelector,
		}); err == nil && crbList != nil {
			for _, crb := range crbList.Items {
				if crb.Labels[PlatformRBACLabelKey] != PlatformRBACLabelVal {
					continue
				}
				for _, subj := range crb.Subjects {
					if subj.Kind == "User" {
						items = append(items, K8sRoleBindingItem{
							ID:        string(crb.UID),
							Username:  subj.Name,
							Namespace: "*",
							RoleName:  crb.RoleRef.Name,
							K8sGroup:  "tekton-devops:" + normalizeRoleType(crb.RoleRef.Name),
							Labels:    crb.Labels,
							CreatedAt: crb.CreationTimestamp.Time,
						})
					}
				}
			}
		}
		if rbList, err := s.k8sBundle.Clientset.RbacV1().RoleBindings("").List(ctx, metav1.ListOptions{
			LabelSelector: PlatformRBACLabelSelector,
		}); err == nil && rbList != nil {
			for _, rb := range rbList.Items {
				if rb.Labels[PlatformRBACLabelKey] != PlatformRBACLabelVal {
					continue
				}
				for _, subj := range rb.Subjects {
					if subj.Kind == "User" {
						items = append(items, K8sRoleBindingItem{
							ID:        string(rb.UID),
							Username:  subj.Name,
							Namespace: rb.Namespace,
							RoleName:  rb.RoleRef.Name,
							K8sGroup:  "tekton-devops:" + normalizeRoleType(rb.RoleRef.Name),
							Labels:    rb.Labels,
							CreatedAt: rb.CreationTimestamp.Time,
						})
					}
				}
			}
		}
		if len(items) > 0 {
			api.Success(c, gin.H{
				"label_selector": PlatformRBACLabelSelector,
				"items":          items,
			})
			return
		}
	}

	rows, err := s.db.QueryContext(ctx,
		`SELECT id, username, namespace, role_name, k8s_group, created_at FROM k8s_role_bindings ORDER BY created_at DESC`)
	if err != nil {
		api.Error(c, http.StatusInternalServerError, 50001, err.Error())
		return
	}
	defer rows.Close()

	items := make([]K8sRoleBindingItem, 0)
	for rows.Next() {
		var b K8sRoleBindingItem
		if rows.Scan(&b.ID, &b.Username, &b.Namespace, &b.RoleName, &b.K8sGroup, &b.CreatedAt) == nil {
			b.Labels = map[string]string{
				PlatformRBACLabelKey:           PlatformRBACLabelVal,
				"app.kubernetes.io/managed-by": "tekton-devops-dashboard",
			}
			items = append(items, b)
		}
	}
	api.Success(c, gin.H{
		"label_selector": PlatformRBACLabelSelector,
		"items":          items,
	})
}

// handleCreateRBACBinding creates or updates a K8s RoleBinding / ClusterRoleBinding carrying `tekton-devops-dashboard: "true"`
func (s *Server) handleCreateRBACBinding(c *gin.Context) {
	if !s.requirePlatformAdmin(c) {
		return
	}
	var req struct {
		Username  string `json:"username"`
		Namespace string `json:"namespace"`
		RoleName  string `json:"role_name"`
	}
	if err := c.ShouldBindJSON(&req); err != nil || strings.TrimSpace(req.Username) == "" {
		api.Error(c, http.StatusBadRequest, 40001, "必须指定授权用户名")
		return
	}
	username := strings.TrimSpace(req.Username)
	ns := strings.TrimSpace(req.Namespace)
	if ns == "" || ns == "ALL_NAMESPACES" {
		ns = "*"
	}
	roleName := strings.TrimSpace(req.RoleName)
	if roleName == "" {
		roleName = RoleNameWorker
	}

	// Verify target role is a `tekton-devops-dashboard: "true"` labeled role
	var labeledExists int
	_ = s.db.QueryRowContext(c.Request.Context(), repository.Rebind(s.cfg.Database.Driver,
		`SELECT count(*) FROM k8s_labeled_roles WHERE name = ? AND label_key = ? AND label_value = ?`),
		roleName, PlatformRBACLabelKey, PlatformRBACLabelVal).Scan(&labeledExists)
	if labeledExists == 0 {
		roleType := normalizeRoleType(roleName)
		roleName = roleTypeToClusterRoleName(roleType)
	}

	rType := normalizeRoleType(roleName)
	k8sGroup := "tekton-devops:" + rType

	id := uuid.New().String()
	_, _ = s.db.ExecContext(c.Request.Context(), repository.Rebind(s.cfg.Database.Driver,
		`DELETE FROM k8s_role_bindings WHERE username = ? AND namespace = ?`), username, ns)
	_, err := s.db.ExecContext(c.Request.Context(), repository.Rebind(s.cfg.Database.Driver,
		`INSERT INTO k8s_role_bindings (id, username, namespace, role_name, k8s_group, created_at) VALUES (?, ?, ?, ?, ?, ?)`),
		id, username, ns, roleName, k8sGroup, time.Now().UTC())
	if err != nil {
		api.Error(c, http.StatusInternalServerError, 50001, "创建带标签 K8s RoleBinding 失败: "+err.Error())
		return
	}

	api.Success(c, K8sRoleBindingItem{
		ID:        id,
		Username:  username,
		Namespace: ns,
		RoleName:  roleName,
		K8sGroup:  k8sGroup,
		Labels: map[string]string{
			PlatformRBACLabelKey:           PlatformRBACLabelVal,
			"app.kubernetes.io/managed-by": "tekton-devops-dashboard",
		},
		CreatedAt: time.Now().UTC(),
	})
}

// handleDeleteRBACBinding deletes a labeled K8s RoleBinding
func (s *Server) handleDeleteRBACBinding(c *gin.Context) {
	if !s.requirePlatformAdmin(c) {
		return
	}
	id := strings.TrimSpace(c.Param("id"))
	_, _ = s.db.ExecContext(c.Request.Context(), repository.Rebind(s.cfg.Database.Driver,
		`DELETE FROM k8s_role_bindings WHERE id = ?`), id)
	api.Success(c, gin.H{"deleted": id})
}

// ensureDefaultTriggersResources seeds default Tekton Triggers CRDs for all 6 pages
func (s *Server) ensureDefaultTriggersResources(ctx context.Context) {
	var count int
	_ = s.db.QueryRowContext(ctx, `SELECT count(*) FROM k8s_triggers_resources`).Scan(&count)
	if count > 0 {
		return
	}

	seeds := []struct {
		kind, apiVer, name, ns string
		spec                   gin.H
	}{
		{
			"eventlisteners", "triggers.tekton.dev/v1beta1", "github-webhook-listener", "default",
			gin.H{
				"serviceAccountName": "default",
				"triggers": []gin.H{
					{"name": "github-push-trigger", "triggerRef": "github-push-trigger"},
				},
			},
		},
		{
			"triggers", "triggers.tekton.dev/v1beta1", "github-push-trigger", "default",
			gin.H{
				"serviceAccountName": "default",
				"bindings":           []gin.H{{"ref": "github-push-binding"}},
				"template":           gin.H{"ref": "ci-pipeline-template"},
				"interceptors":       []gin.H{{"ref": gin.H{"name": "github-cel-filter"}}},
			},
		},
		{
			"triggerbindings", "triggers.tekton.dev/v1beta1", "github-push-binding", "default",
			gin.H{
				"params": []gin.H{
					{"name": "gitrevision", "value": "$(body.head_commit.id)"},
					{"name": "gitrepositoryurl", "value": "$(body.repository.clone_url)"},
				},
			},
		},
		{
			"clustertriggerbindings", "triggers.tekton.dev/v1beta1", "cluster-git-commit-binding", "",
			gin.H{
				"params": []gin.H{
					{"name": "gitbranch", "value": "$(body.ref)"},
					{"name": "pusher", "value": "$(body.pusher.name)"},
				},
			},
		},
		{
			"triggertemplates", "triggers.tekton.dev/v1beta1", "ci-pipeline-template", "default",
			gin.H{
				"params": []gin.H{
					{"name": "gitrevision", "default": "main"},
					{"name": "gitrepositoryurl", "default": "https://github.com/tektoncd/pipeline"},
				},
				"resourcetemplates": []gin.H{
					{
						"apiVersion": "tekton.dev/v1",
						"kind":       "PipelineRun",
						"metadata":   gin.H{"generateName": "webhook-ci-run-"},
						"spec": gin.H{
							"pipelineRef": gin.H{"name": "fullstack-ci-pipeline"},
						},
					},
				},
			},
		},
		{
			"interceptors", "triggers.tekton.dev/v1alpha1", "github-cel-filter", "default",
			gin.H{
				"clientConfig": gin.H{
					"service": gin.H{"name": "tekton-triggers-core-interceptors", "namespace": "tekton-pipelines"},
				},
				"params": []gin.H{
					{"name": "filter", "value": "header.match('X-GitHub-Event', 'push')"},
				},
			},
		},
		{
			"clusterinterceptors", "triggers.tekton.dev/v1alpha1", "cel", "",
			gin.H{
				"clientConfig": gin.H{
					"service": gin.H{"name": "tekton-triggers-core-interceptors", "namespace": "tekton-pipelines"},
				},
			},
		},
	}

	for _, item := range seeds {
		specBytes, _ := json.Marshal(item.spec)
		_, _ = s.db.ExecContext(ctx, repository.Rebind(s.cfg.Database.Driver,
			`INSERT INTO k8s_triggers_resources (id, kind, api_version, name, namespace, spec_json, created_at)
			 VALUES (?, ?, ?, ?, ?, ?, ?)`),
			uuid.New().String(), item.kind, item.apiVer, item.name, item.ns, string(specBytes), time.Now().UTC())
	}
}

func triggersKindToSingularAndListKind(resourceType string) (string, string, string, bool) {
	switch resourceType {
	case "eventlisteners":
		return "EventListener", "EventListenerList", "triggers.tekton.dev/v1beta1", true
	case "triggers":
		return "Trigger", "TriggerList", "triggers.tekton.dev/v1beta1", true
	case "triggerbindings":
		return "TriggerBinding", "TriggerBindingList", "triggers.tekton.dev/v1beta1", true
	case "clustertriggerbindings":
		return "ClusterTriggerBinding", "ClusterTriggerBindingList", "triggers.tekton.dev/v1beta1", false
	case "triggertemplates":
		return "TriggerTemplate", "TriggerTemplateList", "triggers.tekton.dev/v1beta1", true
	case "interceptors":
		return "Interceptor", "InterceptorList", "triggers.tekton.dev/v1alpha1", true
	case "clusterinterceptors":
		return "ClusterInterceptor", "ClusterInterceptorList", "triggers.tekton.dev/v1alpha1", false
	default:
		return "Trigger", "TriggerList", "triggers.tekton.dev/v1beta1", true
	}
}

// handleTektonTriggersFallback implements full CRUD for all 6 Tekton Triggers resource types
func (s *Server) handleTektonTriggersFallback(c *gin.Context, resourceType, reqNamespace string, hasExplicitNamespace bool, parts []string) {
	singularKind, listKind, defaultAPIVersion, isNamespaced := triggersKindToSingularAndListKind(resourceType)

	isDetail := false
	var resourceName string
	for i, p := range parts {
		if p == resourceType && i+1 < len(parts) && parts[i+1] != "" {
			isDetail = true
			resourceName = parts[i+1]
			break
		}
	}

	formatItem := func(id, name, ns, apiVer, specJSON string, createdAt time.Time) gin.H {
		var spec gin.H
		if specJSON != "" {
			_ = json.Unmarshal([]byte(specJSON), &spec)
		}
		if spec == nil {
			spec = gin.H{}
		}
		meta := gin.H{
			"name":              name,
			"uid":               id,
			"creationTimestamp": createdAt.Format(time.RFC3339),
			"resourceVersion":   "1",
			"labels": gin.H{
				"app.kubernetes.io/managed-by": "tekton-devops-dashboard",
				PlatformRBACLabelKey:           PlatformRBACLabelVal,
			},
		}
		if isNamespaced {
			if ns == "" {
				ns = "default"
			}
			meta["namespace"] = ns
		}
		item := gin.H{
			"apiVersion": apiVer,
			"kind":       singularKind,
			"metadata":   meta,
			"spec":       spec,
		}
		if resourceType == "eventlisteners" {
			item["status"] = gin.H{
				"configuration": gin.H{"generatedName": "el-" + name},
				"address":       gin.H{"url": fmt.Sprintf("http://el-%s.%s.svc.cluster.local:8080", name, ns)},
				"conditions": []gin.H{
					{"type": "Ready", "status": "True", "reason": "MinimumReplicasAvailable", "message": "EventListener is ready"},
				},
			}
		}
		return item
	}

	// POST / PUT: Create or update Triggers resource
	if c.Request.Method == http.MethodPost || c.Request.Method == http.MethodPut {
		var payload struct {
			APIVersion string `json:"apiVersion"`
			Metadata   struct {
				Name      string `json:"name"`
				Namespace string `json:"namespace"`
			} `json:"metadata"`
			Spec map[string]interface{} `json:"spec"`
		}
		if err := c.ShouldBindJSON(&payload); err != nil || strings.TrimSpace(payload.Metadata.Name) == "" {
			c.JSON(http.StatusBadRequest, gin.H{"message": "metadata.name 不能为空"})
			return
		}
		name := strings.TrimSpace(payload.Metadata.Name)
		ns := strings.TrimSpace(payload.Metadata.Namespace)
		if !isNamespaced {
			ns = ""
		} else if ns == "" {
			ns = reqNamespace
			if ns == "" {
				ns = "default"
			}
		}
		apiVer := payload.APIVersion
		if apiVer == "" {
			apiVer = defaultAPIVersion
		}
		specBytes, _ := json.Marshal(payload.Spec)
		id := uuid.New().String()
		now := time.Now().UTC()

		_, _ = s.db.ExecContext(c.Request.Context(), repository.Rebind(s.cfg.Database.Driver,
			`DELETE FROM k8s_triggers_resources WHERE kind = ? AND namespace = ? AND name = ?`),
			resourceType, ns, name)
		_, err := s.db.ExecContext(c.Request.Context(), repository.Rebind(s.cfg.Database.Driver,
			`INSERT INTO k8s_triggers_resources (id, kind, api_version, name, namespace, spec_json, created_at)
			 VALUES (?, ?, ?, ?, ?, ?, ?)`),
			id, resourceType, apiVer, name, ns, string(specBytes), now)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"message": err.Error()})
			return
		}
		c.JSON(http.StatusCreated, formatItem(id, name, ns, apiVer, string(specBytes), now))
		return
	}

	// DELETE
	if c.Request.Method == http.MethodDelete && isDetail {
		if isNamespaced && hasExplicitNamespace {
			_, _ = s.db.ExecContext(c.Request.Context(), repository.Rebind(s.cfg.Database.Driver,
				`DELETE FROM k8s_triggers_resources WHERE kind = ? AND namespace = ? AND name = ?`),
				resourceType, reqNamespace, resourceName)
		} else {
			_, _ = s.db.ExecContext(c.Request.Context(), repository.Rebind(s.cfg.Database.Driver,
				`DELETE FROM k8s_triggers_resources WHERE kind = ? AND name = ?`),
				resourceType, resourceName)
		}
		c.JSON(http.StatusOK, gin.H{"apiVersion": defaultAPIVersion, "kind": "Status", "status": "Success"})
		return
	}

	// GET Detail
	if isDetail {
		var id, name, ns, apiVer, specJSON string
		var createdAt time.Time
		var err error
		if isNamespaced && hasExplicitNamespace {
			err = s.db.QueryRowContext(c.Request.Context(), repository.Rebind(s.cfg.Database.Driver,
				`SELECT id, name, namespace, api_version, spec_json, created_at FROM k8s_triggers_resources WHERE kind = ? AND namespace = ? AND name = ?`),
				resourceType, reqNamespace, resourceName).Scan(&id, &name, &ns, &apiVer, &specJSON, &createdAt)
		}
		if err != nil || id == "" {
			err = s.db.QueryRowContext(c.Request.Context(), repository.Rebind(s.cfg.Database.Driver,
				`SELECT id, name, namespace, api_version, spec_json, created_at FROM k8s_triggers_resources WHERE kind = ? AND name = ? LIMIT 1`),
				resourceType, resourceName).Scan(&id, &name, &ns, &apiVer, &specJSON, &createdAt)
		}
		if err != nil || id == "" {
			c.JSON(http.StatusNotFound, gin.H{"message": fmt.Sprintf("%s '%s' not found", singularKind, resourceName)})
			return
		}
		c.JSON(http.StatusOK, formatItem(id, name, ns, apiVer, specJSON, createdAt))
		return
	}

	// GET List
	rows, err := s.db.QueryContext(c.Request.Context(), repository.Rebind(s.cfg.Database.Driver,
		`SELECT id, name, namespace, api_version, spec_json, created_at FROM k8s_triggers_resources WHERE kind = ? ORDER BY created_at DESC`),
		resourceType)
	items := make([]gin.H, 0)
	if err == nil {
		defer rows.Close()
		for rows.Next() {
			var id, name, ns, apiVer, specJSON string
			var createdAt time.Time
			if rows.Scan(&id, &name, &ns, &apiVer, &specJSON, &createdAt) == nil {
				if isNamespaced && hasExplicitNamespace && ns != reqNamespace {
					continue
				}
				items = append(items, formatItem(id, name, ns, apiVer, specJSON, createdAt))
			}
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"apiVersion": defaultAPIVersion,
		"kind":       listKind,
		"metadata":   gin.H{"resourceVersion": "1"},
		"items":      items,
	})
}

// handleNativeDashboardMetrics returns real-time Kubernetes & Tekton native resource counts and recent PipelineRuns
func (s *Server) handleNativeDashboardMetrics(c *gin.Context) {
	reqNS := strings.TrimSpace(c.Query("namespace"))
	filterByNS := reqNS != "" && reqNS != "*" && reqNS != "ALL_NAMESPACES"

	claims := s.extractClaimsFromRequest(c)
	uName, gRole := "admin", "admin"
	if claims != nil {
		uName, gRole = claims.Username, claims.Role
	}

	nsList := s.listNamespacesInternal(c.Request.Context(), uName, gRole)
	totalNamespaces := len(nsList)

	var totalLegacyProjects int
	_ = s.db.QueryRowContext(c.Request.Context(), `SELECT count(*) FROM projects`).Scan(&totalLegacyProjects)
	if totalLegacyProjects == 0 {
		totalLegacyProjects = totalNamespaces
	}

	projectNames := make(map[string]string)
	if projs, _, pErr := s.projectRepo.List(c.Request.Context(), 1, 100); pErr == nil {
		for _, pr := range projs {
			projectNames[pr.ID] = pr.Name
		}
	}

	totalPipelines := 0
	pipelineNames := make(map[string]string)
	pipelineNS := make(map[string]string)
	pipelineProjNames := make(map[string]string)
	if pipes, err := s.pipelineRepo.ListAll(c.Request.Context()); err == nil {
		for _, p := range pipes {
			pNS := extractPipelineNamespace(p)
			pipelineNames[p.ID] = p.Name
			pipelineNS[p.ID] = pNS
			if pName, ok := projectNames[p.ProjectID]; ok && pName != "" && pName != "default-workspace" {
				pipelineProjNames[p.ID] = pName
			} else {
				pipelineProjNames[p.ID] = pNS
			}
			if !filterByNS || pNS == reqNS {
				totalPipelines++
			}
		}
	}

	totalTasks := 0
	if tasks, err := s.taskRepo.ListAll(c.Request.Context()); err == nil {
		for _, t := range tasks {
			tNS := t.Namespace
			if tNS == "" {
				tNS = "default"
			}
			if !filterByNS || tNS == reqNS {
				totalTasks++
			}
		}
	}

	totalSecrets := 0
	if filterByNS {
		_ = s.db.QueryRowContext(c.Request.Context(), repository.Rebind(s.cfg.Database.Driver,
			`SELECT count(*) FROM k8s_secrets WHERE namespace = ?`), reqNS).Scan(&totalSecrets)
	} else {
		_ = s.db.QueryRowContext(c.Request.Context(), `SELECT count(*) FROM k8s_secrets`).Scan(&totalSecrets)
	}
	var legacyCreds int
	_ = s.db.QueryRowContext(c.Request.Context(), `SELECT count(*) FROM credentials`).Scan(&legacyCreds)

	statusCounts := make(map[string]int)
	var totalRuns, succeededRuns, failedRuns, runningRuns, cancelledRuns int
	recentRuns := make([]gin.H, 0)

	if runs, err := s.runRepo.ListAll(c.Request.Context()); err == nil {
		for _, r := range runs {
			rNS := r.Namespace
			if rNS == "" {
				rNS = pipelineNS[r.PipelineID]
			}
			if rNS == "" {
				rNS = "default"
			}
			if filterByNS && rNS != reqNS {
				continue
			}
			totalRuns++
			statusCounts[r.Status]++
			switch r.Status {
			case "Succeeded":
				succeededRuns++
			case "Failed":
				failedRuns++
			case "Running":
				runningRuns++
			case "Cancelled":
				cancelledRuns++
			}

			if len(recentRuns) < 12 {
				var dur int64
				if r.StartedAt != nil && r.FinishedAt != nil && !r.FinishedAt.Before(*r.StartedAt) {
					dur = int64(r.FinishedAt.Sub(*r.StartedAt).Seconds())
				}
				pName := pipelineNames[r.PipelineID]
				if pName == "" {
					pName = r.PipelineID
				}
				projName := pipelineProjNames[r.PipelineID]
				if projName == "" {
					projName = rNS
				}
				runName := r.K8sPipelineRunName
				if runName == "" {
					if len(r.ID) >= 8 {
						runName = pName + "-run-" + r.ID[:6]
					} else {
						runName = pName + "-run"
					}
				}
				recentRuns = append(recentRuns, gin.H{
					"id":               r.ID,
					"k8s_name":         runName,
					"pipeline_id":      r.PipelineID,
					"pipeline_name":    pName,
					"namespace":        rNS,
					"project_name":     projName,
					"status":           r.Status,
					"triggered_by":     r.TriggeredBy,
					"duration_seconds": dur,
					"started_at":       r.StartedAt,
					"finished_at":      r.FinishedAt,
					"created_at":       r.CreatedAt,
				})
			}
		}
	}

	api.Success(c, gin.H{
		"namespace":         reqNS,
		"total_namespaces":  totalNamespaces,
		"total_pipelines":   totalPipelines,
		"total_tasks":       totalTasks,
		"total_secrets":     totalSecrets,
		"total_projects":    totalLegacyProjects,
		"total_credentials": totalSecrets + legacyCreds,
		"total_runs":        totalRuns,
		"succeeded_runs":    succeededRuns,
		"failed_runs":       failedRuns,
		"running_runs":      runningRuns,
		"cancelled_runs":    cancelledRuns,
		"status_counts":     statusCounts,
		"recent_runs":       recentRuns,
	})
}

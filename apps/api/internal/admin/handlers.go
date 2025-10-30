package admin

import (
	"encoding/json"
	"net/http"

	"api/internal/auth"
	"api/internal/database"
)

func requireOrgManager(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		orgID := r.URL.Query().Get("organization_id")
		if orgID == "" { http.Error(w, "organization_id required", http.StatusBadRequest); return }
		if !auth.HasOrgRole(r.Context(), orgID, auth.RoleOrgManager) && !auth.HasOrgRole(r.Context(), orgID, auth.RoleOrgAdmin) {
			http.Error(w, "forbidden", http.StatusForbidden); return
		}
		next(w, r)
	}
}

func UpsertQuota(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost && r.Method != http.MethodPut { http.Error(w, "method not allowed", http.StatusMethodNotAllowed); return }
	var q database.OrgQuota
	if err := json.NewDecoder(r.Body).Decode(&q); err != nil { http.Error(w, err.Error(), http.StatusBadRequest); return }
	if q.OrganizationID == "" { http.Error(w, "organization_id required", http.StatusBadRequest); return }
	if err := database.DB.Save(&q).Error; err != nil { http.Error(w, err.Error(), http.StatusInternalServerError); return }
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(q)
}

func CreateRole(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost { http.Error(w, "method not allowed", http.StatusMethodNotAllowed); return }
	var role database.OrgRole
	if err := json.NewDecoder(r.Body).Decode(&role); err != nil { http.Error(w, err.Error(), http.StatusBadRequest); return }
	if role.OrganizationID == "" || role.Name == "" { http.Error(w, "missing fields", http.StatusBadRequest); return }
	if err := database.DB.Create(&role).Error; err != nil { http.Error(w, err.Error(), http.StatusInternalServerError); return }
	w.WriteHeader(http.StatusCreated)
	_ = json.NewEncoder(w).Encode(role)
}

func BindRole(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost { http.Error(w, "method not allowed", http.StatusMethodNotAllowed); return }
	var b database.OrgRoleBinding
	if err := json.NewDecoder(r.Body).Decode(&b); err != nil { http.Error(w, err.Error(), http.StatusBadRequest); return }
	if b.OrganizationID == "" || b.UserID == "" || b.RoleID == "" { http.Error(w, "missing fields", http.StatusBadRequest); return }
	if err := database.DB.Create(&b).Error; err != nil { http.Error(w, err.Error(), http.StatusInternalServerError); return }
	w.WriteHeader(http.StatusCreated)
	_ = json.NewEncoder(w).Encode(b)
}

func ListRoles(w http.ResponseWriter, r *http.Request) {
	orgID := r.URL.Query().Get("organization_id")
	var roles []database.OrgRole
	_ = database.DB.Where("organization_id = ?", orgID).Find(&roles)
	_ = json.NewEncoder(w).Encode(roles)
}

func ListBindings(w http.ResponseWriter, r *http.Request) {
	orgID := r.URL.Query().Get("organization_id")
	var bindings []database.OrgRoleBinding
	_ = database.DB.Where("organization_id = ?", orgID).Find(&bindings)
	_ = json.NewEncoder(w).Encode(bindings)
}

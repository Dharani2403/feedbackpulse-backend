package handler

import (
	"net/http"
)

func handleListTenants(d Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		tenants, err := d.Tenants.List()
		if err != nil {
			jsonError(w, "failed to list tenants", http.StatusInternalServerError)
			return
		}
		jsonOK(w, tenants)
	}
}

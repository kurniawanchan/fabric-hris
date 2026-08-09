package pipeline

import (
	"net/http"
	"time"
)

// RouteConfig is what differs between AD-5's five profile-section routes:
// which literal ProfileSection value they anchor under, and which Hooks
// write-path method [dispatch] calls. Every route shares the exact same
// pipeline shape below -- adding a route (Stories 1.3-1.6) means adding a
// RouteConfig value and a mux registration, never a new handler body.
type RouteConfig struct {
	ProfileSection string
	Dispatch       DispatchFunc
}

// RegisterProfileSectionRoute wires one AD-5 route through the shared
// 5-stage pipeline: [auth] -> [validate] -> [dispatch] -> [map-error] ->
// [respond]. Every failure path -- including [auth]/[validate] rejections
// -- funnels through classify() before any response is written (AD-1; the
// architecture reviewer gate's own Finding 3 required this explicitly: an
// early stage must never bypass error-classification on its own failure
// path, so [respond] is always reached through classify(), never directly).
//
// path must already be a method-qualified pattern (e.g.
// "POST /v1/profile-sections/PERSONAL") -- stdlib net/http's ServeMux
// handles the 405+Allow response for a mismatched method itself (AD-6).
func RegisterProfileSectionRoute(mux *http.ServeMux, path string, route RouteConfig, authCfg AuthConfig, tenantID string, timeout time.Duration) {
	mux.HandleFunc(path, func(w http.ResponseWriter, r *http.Request) {
		bc := &bridgectx{ProfileSection: route.ProfileSection, TenantID: tenantID}
		ctx := withBridgeCtx(r.Context(), bc)

		if err := authenticate(r, authCfg); err != nil {
			bc.Err = err
			respond(w, classify(bc.Err), bc.Result, bc.Err)
			return
		}

		employeeInternalID, userID, newValue, document, err := validateRequest(w, r)
		if err != nil {
			bc.Err = err
			respond(w, classify(bc.Err), bc.Result, bc.Err)
			return
		}
		bc.EmployeeInternalID = employeeInternalID

		bc.Result, bc.Err = dispatch(ctx, timeout, route.Dispatch, employeeInternalID, userID, newValue, document)
		respond(w, classify(bc.Err), bc.Result, bc.Err)
	})
}

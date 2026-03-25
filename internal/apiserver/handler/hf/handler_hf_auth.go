package hf

import (
	"net/http"

	"github.com/matrixhub-ai/hfd/pkg/authenticate"
)

// handleWhoami handles GET /api/whoami-v2
func (h *Handler) handleWhoami(w http.ResponseWriter, r *http.Request) {
	userInfo, ok := authenticate.GetUserInfo(r.Context())
	if !ok || userInfo.User == authenticate.Anonymous {
		responseJSON(w, map[string]string{"error": "Unauthorized"}, http.StatusUnauthorized)
		return
	}

	resp := whoamiResponse{
		Type:          "user",
		ID:            userInfo.User,
		Name:          userInfo.User,
		Fullname:      userInfo.User,
		Email:         userInfo.Email,
		EmailVerified: false,
		IsPro:         false,
		CanPay:        false,
		Orgs:          []any{},
		Auth: authInfo{
			AccessToken: accessToken{
				DisplayName: "token",
				Role:        "write",
			},
		},
	}

	responseJSON(w, resp, http.StatusOK)
}

package resolver

import (
	pklHTTP "github.com/kdeps/schema/gen/http"
	pklResource "github.com/kdeps/schema/gen/resource"
)

// HTTPResourceHandler extends the generic resource handler for HTTP-specific operations
type HTTPResourceHandler struct {
	*ResourceHandler[pklHTTP.ResourceHTTPClient]
}

// NewHTTPResourceHandler creates a new HTTP-specific resource handler
func NewHTTPResourceHandler(dr *DependencyResolver) *HTTPResourceHandler {
	base := NewResourceHandler[pklHTTP.ResourceHTTPClient](dr, "HTTP")
	return &HTTPResourceHandler{
		ResourceHandler: base,
	}
}

// updateResourceFromReloaded implements HTTP-specific resource updating
func (h *HTTPResourceHandler) updateResourceFromReloaded(resource *pklHTTP.ResourceHTTPClient, reloadedGenericResource *pklResource.Resource) {
	// Extract the HTTP block from the reloaded resource
	if reloadedGenericResource.Run != nil && reloadedGenericResource.Run.HTTPClient != nil {
		reloadedHTTP := reloadedGenericResource.Run.HTTPClient

		// Update the resource with the reloaded values that contain fresh template evaluation
		if reloadedHTTP.Url != "" {
			resource.Url = reloadedHTTP.Url
			h.dr.Logger.Debug("updateResourceFromReloaded: updated URL from reloaded resource")
		}

		if reloadedHTTP.Method != "" {
			resource.Method = reloadedHTTP.Method
			h.dr.Logger.Debug("updateResourceFromReloaded: updated method from reloaded resource")
		}

		if reloadedHTTP.Headers != nil {
			resource.Headers = reloadedHTTP.Headers
			h.dr.Logger.Debug("updateResourceFromReloaded: updated headers from reloaded resource")
		}

		if reloadedHTTP.Params != nil {
			resource.Params = reloadedHTTP.Params
			h.dr.Logger.Debug("updateResourceFromReloaded: updated params from reloaded resource")
		}

		if reloadedHTTP.Data != nil {
			resource.Data = reloadedHTTP.Data
			h.dr.Logger.Debug("updateResourceFromReloaded: updated data from reloaded resource")
		}
	}
}

// HandleHTTPClientV2 handles HTTP client resources using the specialized handler
func (dr *DependencyResolver) HandleHTTPClientV2(actionID string, httpBlock *pklHTTP.ResourceHTTPClient) error {
	handler := NewHTTPResourceHandler(dr)

	// Use the specialized handler (method override handled in the handler itself)
	return handler.HandleResource(actionID, httpBlock, func(resource *pklHTTP.ResourceHTTPClient) error {
		return dr.processHTTPBlock(actionID, resource)
	})
}

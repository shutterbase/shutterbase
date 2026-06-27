package server

// registerAPIRoutes is the router-registration seam: every CRUD controller is
// wired onto the /api/v1 group here, AFTER authentication.Setup has installed
// RequireAuth (engine.Use), so all of these inherit the auth middleware. Per-role
// authorization is S8 — each handler carries an `authz (S8)` seam comment marking
// where the role check will plug in; for now routes require authentication only.
func (s *Server) registerAPIRoutes() {
	api := s.Engine.Group(s.options.ApiBaseURL)

	s.registerImageRoutes(api)
	s.registerImageTagRoutes(api)
	s.registerImageTagAssignmentRoutes(api)
	s.registerProjectRoutes(api)
	s.registerProjectAssignmentRoutes(api)
	s.registerCameraRoutes(api)
	s.registerUploadRoutes(api)
	s.registerTimeOffsetRoutes(api)
	s.registerRoleRoutes(api)
	s.registerUserRoutes(api)
	s.registerCustomRoutes(api)
}

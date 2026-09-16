package api

import "nofx/logger"

func (s *Server) planStructuralGeometry(traderID, planID string, version int) any {
	rows, err := s.store.StructuralGeometryFor(traderID, planID, version)
	if err != nil {
		logger.Warnf("plan geometry read unavailable: %v", err)
		return nil
	}
	return rows
}

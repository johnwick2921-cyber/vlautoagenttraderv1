package agent

// stateOwner keeps sync.Map/mutex values in their original allocation. Never
// shallow-copy Agent: that would copy locks while sharing their underlying data.
func (a *Agent) stateOwner() *Agent {
	if a.runtimeOwner != nil {
		return a.runtimeOwner
	}
	return a
}

// requestRuntime fixes the selected model/client for the entire request,
// including tool follow-ups and memory summaries. Concurrent users cannot
// replace each other's model credentials or the background agent's client.
//
// F20 (ported fresh from dev, 2026-09-25): HandleMessage /
// HandleMessageForStoreUser used to call ensureAIClientForStoreUser on the
// SHARED agent, so two authenticated chats overwrote each other's model/key
// mid-request. The request-local runtime shares conversation, flow locks,
// setup state and all owner plumbing, but owns its own aiClient.
func (a *Agent) requestRuntime(storeUserID string) *Agent {
	owner := a.stateOwner()
	owner.ensureHistory()
	local := &Agent{
		runtimeOwner:  owner,
		traderManager: owner.traderManager,
		store:         owner.store,
		config:        owner.config,
		sentinel:      owner.sentinel,
		brain:         owner.brain,
		scheduler:     owner.scheduler,
		logger:        owner.logger,
		history:       owner.history,
		pending:       owner.pending,
		stopCh:        owner.stopCh,
		NotifyFunc:    owner.NotifyFunc,
	}
	local.ensureAIClientForStoreUser(storeUserID)
	return local
}

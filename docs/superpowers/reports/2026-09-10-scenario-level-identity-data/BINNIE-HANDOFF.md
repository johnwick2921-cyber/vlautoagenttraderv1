# Binnie lane: dispatch 105 partner patch handoff

Owner routing: hand this patch to Binnie; 105 must not apply it.

Patch: /tmp/identity-build/partner-identity.patch
SHA256: 59c582b9684173f8cc3caf29cf8b56f9d4b2f0bb8418c37b44e418d0bdedc138
Source branch: fix/scenario-level-identity, PR #101
Base: de26d1e4867b2b1e356b7c90d7b93305060cff41 (W2)
Code candidate: b30afc6570936a628c8cef1f462dc9e8dd8216d9

Partner checkout observed at f6ae7597fb3bc9caeaaedb25ce8c3c48bca72247,
missing kernel/plan_doc.go, kernel/scenario_state.go, store/touch_outcomes.go.
Existing dirty files: agent/planner_runtime_state_test.go,
agent/skill_dispatcher_test.go, agent/trader_scope_test.go.
Do not overwrite existing work. Synchronize prerequisites/history first;
canon requires fresh re-clone after origin history rewrite.
105 has not applied the patch or changed/pushed the partner checkout.
Local nofx full Go suite and frontend 421 tests passed; partner tests have
not run. CI setup failures are pre-existing, owed to cleanup batch 2.

package agent

import (
	"context"
	"log/slog"
	"nofx/mcp"
	"strings"
	"sync"
	"testing"
	"time"
)

// F20 (W117-E) — request-local model selection, pinned at the PRODUCTION call
// sites. The behavioral mutation that must fail these tests is:
//
//	agent.go: handleMessageForStoreUser / handleMessageStreamForStoreUser
//	`a = a.requestRuntime(storeUserID)` → `a.ensureAIClientForStoreUser(storeUserID)`
//	(requestRuntime left defined)
//
// Under that mutation every LLM call in BOTH users' flows is served by whichever
// user's client was loaded onto the SHARED agent last, so at least one reply is
// wrong and the shared aiClient is replaced. Nothing here calls requestRuntime
// directly for the pin; everything goes through HandleMessageForStoreUser /
// HandleMessageStreamForStoreUser.

type requestIsolationCall struct {
	model  string
	system string
	user   string
}

type requestIsolationLog struct {
	mu    sync.Mutex
	calls []requestIsolationCall
}

func (l *requestIsolationLog) add(c requestIsolationCall) {
	l.mu.Lock()
	l.calls = append(l.calls, c)
	l.mu.Unlock()
}

func (l *requestIsolationLog) snapshot() []requestIsolationCall {
	l.mu.Lock()
	defer l.mu.Unlock()
	return append([]requestIsolationCall(nil), l.calls...)
}

// recordingModelClient is bound to ONE store user's model: loadAIClientFromStoreUser
// calls SetAPIKey with that user's custom model name. It records every call it
// serves and scripts the LLM stages of a planned-agent request so a single
// production entry-point call exercises
//
//	router LLM call → step-selector LLM call → tool execution (get_exchange_configs)
//	→ step-selector LLM FOLLOW-UP (sees the tool observation) → final-response LLM call.
type recordingModelClient struct {
	model         string
	log           *requestIsolationLog
	selectorMu    sync.Mutex
	selectorCalls int
}

func (c *recordingModelClient) SetAPIKey(apiKey string, customURL string, customModel string) {
	c.model = customModel
}
func (c *recordingModelClient) SetTimeout(timeout time.Duration) {}
func (c *recordingModelClient) ResolvedModel() string            { return c.model }

func (c *recordingModelClient) scripted(systemPrompt string) string {
	switch {
	case strings.Contains(systemPrompt, "unified turn router"):
		return `{"topic_intent":"start_new","business_action":"planned_agent","target_skill":"","tasks":[],"target_snapshot_id":"","context_mode":"fresh_context","extracted_data":{},"reply_to_user":"","confidence":0.9}`
	case strings.Contains(systemPrompt, "step selector"):
		c.selectorMu.Lock()
		c.selectorCalls++
		n := c.selectorCalls
		c.selectorMu.Unlock()
		if n == 1 {
			return `{"goal":"list my traders","steps":[{"id":"step_1","type":"tool","title":"read exchange configs","tool_name":"get_exchange_configs","tool_args":{},"instruction":"","requires_confirmation":false}]}`
		}
		return `{"goal":"list my traders","steps":[{"id":"step_2","type":"respond","title":"","tool_name":"","tool_args":{},"instruction":"","requires_confirmation":false}]}`
	case strings.Contains(systemPrompt, "trading partner") || strings.Contains(systemPrompt, "交易伙伴"):
		return "served-by-" + c.model + " after tool"
	default:
		return "unrecognized-stage"
	}
}

func (c *recordingModelClient) CallWithMessages(systemPrompt, userPrompt string) (string, error) {
	c.log.add(requestIsolationCall{model: c.model, system: systemPrompt, user: userPrompt})
	return c.scripted(systemPrompt), nil
}

func (c *recordingModelClient) CallWithRequest(req *mcp.Request) (string, error) {
	system, user := firstSystemLastUser(req)
	c.log.add(requestIsolationCall{model: c.model, system: system, user: user})
	return c.scripted(system), nil
}

func (c *recordingModelClient) CallWithRequestStream(req *mcp.Request, onChunk func(string)) (string, error) {
	system, user := firstSystemLastUser(req)
	c.log.add(requestIsolationCall{model: c.model, system: system, user: user})
	reply := c.scripted(system)
	if onChunk != nil {
		onChunk(reply)
	}
	return reply, nil
}

func (c *recordingModelClient) CallWithRequestFull(req *mcp.Request) (*mcp.LLMResponse, error) {
	system, user := firstSystemLastUser(req)
	c.log.add(requestIsolationCall{model: c.model, system: system, user: user})
	return &mcp.LLMResponse{Content: c.scripted(system)}, nil
}

func firstSystemLastUser(req *mcp.Request) (string, string) {
	if req == nil {
		return "", ""
	}
	system, user := "", ""
	for _, m := range req.Messages {
		if m.Role == "system" && system == "" {
			system = m.Content
		}
		if m.Role == "user" {
			user = m.Content
		}
	}
	return system, user
}

// TestConcurrentStoreUserRequestsKeepTheirOwnModelClients — F20 RED-first at the
// production call sites: two store users with different enabled models run the
// same planned-agent flow concurrently (alice via HandleMessageForStoreUser, bob
// via HandleMessageStreamForStoreUser). Each flow makes a router LLM call, a
// step-selector LLM call, executes the get_exchange_configs tool, then makes a
// post-tool step-selector follow-up and a final-response LLM call. Every call in
// a flow must be served by that user's own client, and the shared agent's
// aiClient must be unchanged afterwards.
func TestConcurrentStoreUserRequestsKeepTheirOwnModelClients(t *testing.T) {
	log := &requestIsolationLog{}
	mcp.RegisterProvider("w117etest", func(...mcp.ClientOption) mcp.AIClient {
		return &recordingModelClient{log: log}
	})

	a := newTestAgentWithStore(t)
	a.config = DefaultConfig()
	a.logger = slog.Default()
	for _, user := range []string{"alice", "bob"} {
		if err := a.store.AIModel().Update(user, "w117etest", true, "test-key", "", "model-"+user); err != nil {
			t.Fatalf("seed model for %s: %v", user, err)
		}
	}
	sentinel := &recordingModelClient{model: "sentinel", log: log}
	a.SetAIClient(sentinel)

	var aliceReply, bobReply string
	var wg sync.WaitGroup
	wg.Add(2)
	go func() {
		defer wg.Done()
		reply, err := a.HandleMessageForStoreUser(context.Background(), "alice", 1, "list my traders")
		if err != nil {
			t.Errorf("alice request: %v", err)
			return
		}
		aliceReply = reply
	}()
	go func() {
		defer wg.Done()
		reply, err := a.HandleMessageStreamForStoreUser(context.Background(), "bob", 2, "list my traders", func(event, data string) {})
		if err != nil {
			t.Errorf("bob request: %v", err)
			return
		}
		bobReply = reply
	}()
	wg.Wait()

	if aliceReply != "served-by-model-alice after tool" {
		t.Errorf("alice reply = %q; her request was not served by her own client", aliceReply)
	}
	if bobReply != "served-by-model-bob after tool" {
		t.Errorf("bob reply = %q; his request was not served by his own client", bobReply)
	}
	if a.aiClient != sentinel {
		t.Fatal("authenticated request replaced shared/background AI client")
	}

	byModel := map[string]int{}
	selectors := map[string]int{}
	followUps := map[string]int{}
	finals := map[string]int{}
	for _, call := range log.snapshot() {
		byModel[call.model]++
		if strings.Contains(call.system, "step selector") {
			selectors[call.model]++
			if strings.Contains(call.user, `"kind":"tool_result"`) {
				followUps[call.model]++
			}
		}
		if strings.Contains(call.system, "trading partner") || strings.Contains(call.system, "交易伙伴") {
			finals[call.model]++
		}
	}
	if byModel["model-alice"] == 0 || byModel["model-bob"] == 0 {
		t.Fatalf("each user's own client must serve their calls; got %v", byModel)
	}
	if byModel["sentinel"] != 0 || byModel[""] != 0 {
		t.Fatalf("shared/sentinel client served request calls: %v", byModel)
	}
	if selectors["model-alice"] < 2 || selectors["model-bob"] < 2 {
		t.Fatalf("step-selector LLM calls missing for a user: %v", selectors)
	}
	if followUps["model-alice"] == 0 || followUps["model-bob"] == 0 {
		t.Fatalf("post-tool step-selector follow-up missing for a user: %v", followUps)
	}
	if finals["model-alice"] != 1 || finals["model-bob"] != 1 {
		t.Fatalf("final-response LLM call missing for a user: %v", finals)
	}
}

// TestUnconfiguredStoreUserGetsNoInheritedClient — F20 fallback removal at the
// production call site: an authenticated user with no enabled model must not be
// served by another user's credentials, and must not trigger any LLM call.
func TestUnconfiguredStoreUserGetsNoInheritedClient(t *testing.T) {
	log := &requestIsolationLog{}
	mcp.RegisterProvider("w117etestunconf", func(...mcp.ClientOption) mcp.AIClient {
		return &recordingModelClient{log: log}
	})

	a := newTestAgentWithStore(t)
	a.config = DefaultConfig()
	a.logger = slog.Default()
	if err := a.store.AIModel().Update("alice", "w117etestunconf", true, "test-key", "", "model-alice"); err != nil {
		t.Fatal(err)
	}
	if rt := a.requestRuntime("stranger"); rt.aiClient != nil {
		t.Fatal("unconfigured user inherited another user's client")
	}

	before := len(log.snapshot())
	reply, err := a.HandleMessageForStoreUser(context.Background(), "stranger", 99, "hi")
	if err != nil {
		t.Fatal(err)
	}
	if len(log.snapshot()) != before {
		t.Fatal("unconfigured user's request made LLM calls (inherited a client)")
	}
	if reply == "" {
		t.Fatal("expected a direct reply for the unconfigured user")
	}
}

// TestRequestRuntimeKeepsClientAndConversationOwnership — request runtimes own
// their model client but share conversation/setup/flow-lock state with the
// shared agent (stateOwner routing).
func TestRequestRuntimeKeepsClientAndConversationOwnership(t *testing.T) {
	log := &requestIsolationLog{}
	mcp.RegisterProvider("w117etestowner", func(...mcp.ClientOption) mcp.AIClient {
		return &recordingModelClient{log: log}
	})

	a := newTestAgentWithStore(t)
	a.config = DefaultConfig()
	a.logger = slog.Default()
	for _, user := range []string{"alice", "bob", "default"} {
		if err := a.store.AIModel().Update(user, "w117etestowner", true, "test-key", "", "model-"+user); err != nil {
			t.Fatal(err)
		}
	}
	first := a.requestRuntime("alice")
	second := a.requestRuntime("bob")
	if first == a || second == a || first.aiClient == second.aiClient {
		t.Fatal("request clients were shared")
	}
	if first.aiClient.ResolvedModel() != "model-alice" || second.aiClient.ResolvedModel() != "model-bob" {
		t.Fatal("selected model changed across requests")
	}
	if first.flowLock(42) != second.flowLock(42) {
		t.Fatal("request runtimes lost shared conversation serialization")
	}
	first.history.Add(42, "user", "retained")
	if len(a.history.Get(42)) != 1 {
		t.Fatal("conversation history detached")
	}
	first.saveSetupState(42, &SetupState{Step: "test-step"})
	if second.getSetupState(42).Step != "test-step" {
		t.Fatal("setup state detached")
	}
	second.clearSetupState(42)
	if first.getSetupState(42).Step != "" {
		t.Fatal("setup clear detached")
	}
}

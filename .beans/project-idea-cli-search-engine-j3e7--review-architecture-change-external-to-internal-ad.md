---
# project-idea-cli-search-engine-j3e7
title: 'Review architecture change: External to Internal Adapters'
status: completed
type: task
priority: normal
created_at: 2026-04-18T00:48:03Z
updated_at: 2026-04-18T00:48:20Z
parent: project-idea-cli-search-engine-irmz
---

Apply philosophy of software design to evaluate moving from external adapter binaries to internal Go packages



## Architecture Review: Moving from External to Internal Adapters

### Principle: Information Hiding & Module Boundaries
Currently, adapters are decoupled processes communicating via JSON over stdin/stdout. This is the ultimate form of information hiding: the adapter binary knows nothing about qry's internals, and qry knows nothing about the adapter's implementation.

**Impact of moving internal**:
*   *Risk:* Implementation details might leak. If adapters are just Go packages, they might start depending on 's internal packages (like  or ), creating tight coupling.
*   *Mitigation:* Define a strict interface in an  package (or  interface type). The router should only interact with this interface. The  and  types should ideally be decoupled from internal config representations.

### Principle: Complexity and Cognitive Load
*   *Current state:* Managing separate binaries, timeouts, subprocess execution (), and parsing JSON over pipes adds operational complexity to the core  module. Deployment requires distributing multiple binaries or a complex build process.
*   *Proposed state:* Invoking a Go function is significantly simpler than .  will shrink drastically. Error handling becomes native Go  types rather than parsing stderr JSON.
*   *Benefit:* Lowers cognitive load for anyone reading the  package.

### Principle: Change Amplification
*   *Current state:* Changing the adapter protocol (e.g., adding a new field to ) requires updating  and *every* external adapter binary independently, and ensuring they are deployed together.
*   *Proposed state:* Refactoring the adapter interface is a simple compiler-checked change within the same repository. Change amplification is reduced because the compiler enforces the contract across all adapters simultaneously.

### Module Depth
*   *Router Module:* Currently, the router is fairly deep. It hides the complexity of subprocess management and concurrent execution behind a simple  method. Moving adapters internal will make the implementation of  simpler, slightly reducing the depth by reducing the internal complexity, but it makes the overall system more cohesive.

### Recommendations for the Refactoring
1.  **Define a clear Adapter interface:** e.g., 
2.  **Avoid passing full  to adapters:** Pass only the specific parameters they need (e.g., API keys, result limits). This prevents adapters from relying on global state or unrelated config values.
3.  **Registry pattern:** Implement a way to register internal adapters (e.g., a map of string names to  implementations) so the  can look them up without hardcoding switch statements for every adapter, preserving the data-driven routing logic.

**Conclusion**: The move to internal adapters is a net positive for software design. It trades the extreme (but costly) isolation of subprocesses for the simplicity and compile-time safety of internal interfaces. It will reduce the complexity of the  module significantly.



## Summary of Changes
Evaluated the proposed architectural refactoring (external adapter binaries -> internal Go packages) through the lens of 'A Philosophy of Software Design' principles. Identified risks around information hiding (leaking internal config to adapters) and recommended defining a strict, simple interface to maintain the boundary. Also noted the significant complexity reduction in the router (no subprocesses or JSON parsing over pipes) and the decrease in change amplification for cross-cutting changes. Outlined actionable recommendations for the implementation.

// Package acp provides Go types and connection plumbing for the
// Agent Client Protocol (ACP). It contains generated dispatchers,
// outbound helpers, shared request/response types, and related
// utilities used by agents and clients to communicate over ACP.
//
// # Versioning
//
// This module's version tracks the ACP schema release it is generated
// from, not Go API compatibility: vX.Y.Z is generated from schema
// X.Y.Z. A minor bump can remove exported types or add methods to the
// Client and Agent interfaces, breaking compilation for code that
// implements them. Pin an exact version rather than @latest or @v1.
//
// The wire protocol is more stable than the Go API; a bump that breaks
// compilation usually leaves interoperability with older peers intact.
package acp

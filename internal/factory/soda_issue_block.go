package factory

import (
	"bytes"
	"encoding/json"
	"errors"
	"strings"
)

// MaxSodaIssueBlockBytes bounds one native issue Soda block: the fenced
// JSON contract is tiny by design, and anything larger is prose that
// belongs outside the fence.
const MaxSodaIssueBlockBytes = 4 * 1024

// Admitted issue work roles for the native Soda block. The author
// proposes; dispatch admits. Anything else refuses instead of starting
// a default coding run on an unknown route.
const (
	SodaRoleCoder         = "coder"
	SodaRoleLibraryAdmin  = "library_admin"
	SodaRoleResearch      = "research"
	SodaBlockWorkflowCode = "code_change"
	SodaBlockWorkflowLib  = "library_maintenance"
	SodaBlockWorkflowRes  = "research_report"
)

// SodaIssueBlock is the canonical native issue work contract (R13.2):
// the single strict ```soda fenced JSON block naming the requested
// work role. Human prose stays outside the fence and is never parsed
// for authority. A missing block is the zero value and routes by the
// ordinary default; a present block must name an admitted role.
type SodaIssueBlock struct {
	Role string `json:"role,omitempty"`
}

// ParseSodaIssueBlock extracts the authoritative Soda block from one
// native issue body. Multiple blocks, an unclosed fence, oversize or
// malformed JSON, unknown fields and unknown roles all refuse: one
// ambiguous contract never routes work. A body without the fence
// returns the zero block and no error; ordinary prose is preserved
// untouched, never classified.
func ParseSodaIssueBlock(body string) (SodaIssueBlock, error) {
	var block SodaIssueBlock
	chunks, unclosed := splitSodaFences(body)
	if unclosed {
		return SodaIssueBlock{}, errors.New("unclosed soda metadata block")
	}
	found := false
	for _, chunk := range chunks {
		if found {
			return SodaIssueBlock{}, errors.New("multiple soda metadata blocks")
		}
		found = true
		if len(chunk) > MaxSodaIssueBlockBytes {
			return SodaIssueBlock{}, errors.New("soda metadata block exceeds its bound")
		}
		d := json.NewDecoder(bytes.NewReader([]byte(chunk)))
		d.DisallowUnknownFields()
		if err := d.Decode(&block); err != nil {
			return SodaIssueBlock{}, errors.New("invalid soda issue metadata")
		}
		if strings.TrimSpace(block.Role) == "" {
			return SodaIssueBlock{}, errors.New("soda metadata block names no work role")
		}
		switch block.Role {
		case SodaRoleCoder, SodaRoleLibraryAdmin, SodaRoleResearch:
		default:
			return SodaIssueBlock{}, errors.New("unknown soda issue work role")
		}
	}
	return block, nil
}

// splitSodaFences yields the raw contents of each ```soda fenced block.
// An unclosed fence yields no chunks and reports unclosed so the caller
// refuses instead of decoding a fence that never ended.
func splitSodaFences(body string) ([]string, bool) {
	var out []string
	unclosed := false
	lines := strings.Split(body, "\n")
	for i := 0; i < len(lines); i++ {
		if strings.TrimSpace(lines[i]) != "```soda" {
			continue
		}
		var block []string
		i++
		for ; i < len(lines) && strings.TrimSpace(lines[i]) != "```"; i++ {
			block = append(block, lines[i])
		}
		if i == len(lines) {
			unclosed = true
			break
		}
		out = append(out, strings.Join(block, "\n"))
	}
	return out, unclosed
}

// HasWork reports whether the body carried an explicit work contract.
func (b SodaIssueBlock) HasWork() bool { return b.Role != "" }

// Workflow resolves the block to its R10.1 registry workflow key. The
// zero block carries no classification claim; callers keep their
// ordinary routing default instead of reading a workflow here.
func (b SodaIssueBlock) Workflow() string {
	switch b.Role {
	case SodaRoleLibraryAdmin:
		return SodaBlockWorkflowLib
	case SodaRoleResearch:
		return SodaBlockWorkflowRes
	default:
		return ""
	}
}

// Labels derives the stable navigation aids for one explicit contract:
// its registry workflow and work role. The zero block derives nothing;
// ordinary issues without metadata are never auto-classified.
func (b SodaIssueBlock) Labels() []string {
	if !b.HasWork() {
		return nil
	}
	workflow := b.Workflow()
	if workflow == "" {
		workflow = SodaBlockWorkflowCode
	}
	return []string{"workflow:" + workflow, "role:" + b.Role}
}

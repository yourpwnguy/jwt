package output

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/yourpwnguy/jwt/internal/audit"
	"github.com/yourpwnguy/jwt/internal/jwt"
	"github.com/yourpwnguy/jwt/internal/verify"
)

// Audit/verify rendering tests guard the machine contracts:
// JSON must stay jq-parseable, human must stay greppable, and every
// named format must render distinctly (pretty != table).

func TestFormatAuditModes(t *testing.T) {
	tok, err := jwt.Parse("eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJzdWIiOiIxMjM0NTY3ODkwIiwibmFtZSI6IkpvaG4gRG9lIiwiaWF0IjoxNTE2MjM5MDIyfQ.SflKxwRJSMeKKF2QT4fwpMeJf36POk6yJV_adQssw5c")
	if err != nil {
		t.Fatal(err)
	}
	rep := audit.Run(context.Background(), tok, time.Now())
	human, err := FormatAudit(rep, FormatPretty, true)
	if err != nil || !strings.Contains(human, "missing-exp") {
		t.Errorf("human: %v %s", err, human)
	}
	machine, err := FormatAudit(rep, FormatJSON, true)
	if err != nil || !strings.Contains(machine, `"id"`) {
		t.Errorf("json: %v %s", err, machine)
	}
}

func TestFormatAuditTableDistinct(t *testing.T) {
	tok, err := jwt.Parse("eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJzdWIiOiIxMjM0NTY3ODkwIiwibmFtZSI6IkpvaG4gRG9lIiwiaWF0IjoxNTE2MjM5MDIyfQ.SflKxwRJSMeKKF2QT4fwpMeJf36POk6yJV_adQssw5c")
	if err != nil {
		t.Fatal(err)
	}
	rep := audit.Run(context.Background(), tok, time.Now())
	pretty, _ := FormatAudit(rep, FormatPretty, true)
	table, err := FormatAudit(rep, FormatTable, true)
	if err != nil {
		t.Fatal(err)
	}
	if pretty == table {
		t.Error("pretty and table must render distinctly")
	}
	if !strings.Contains(table, "STATUS") || !strings.Contains(table, "missing-exp") {
		t.Errorf("table should list findings in columns: %s", table)
	}
	// Batch table is a merged per-token overview.
	overview, err := FormatAuditBatch([]audit.Report{rep, rep}, Options{Format: FormatTable, NoColor: true})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(overview, "TOKEN") || !strings.Contains(overview, "missing-exp") {
		t.Errorf("overview should summarize per token: %s", overview)
	}
}

func TestFormatVerifyModes(t *testing.T) {
	tok, err := jwt.Parse("eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJzdWIiOiIxMjM0NTY3ODkwIiwibmFtZSI6IkpvaG4gRG9lIiwiaWF0IjoxNTE2MjM5MDIyfQ.SflKxwRJSMeKKF2QT4fwpMeJf36POk6yJV_adQssw5c")
	if err != nil {
		t.Fatal(err)
	}
	res := verify.Result{Alg: "HS256", SignatureValid: true, ClaimsValid: true}
	human, err := FormatVerify(tok, res, FormatPretty, true)
	if err != nil || !strings.Contains(human, "Signature: VALID") {
		t.Errorf("human: %v %s", err, human)
	}
	machine, err := FormatVerify(tok, res, FormatJSON, true)
	if err != nil || !strings.Contains(machine, `"signatureValid"`) {
		t.Errorf("json: %v %s", err, machine)
	}
	bad := verify.Result{Alg: "HS256", SignatureValid: false, ClaimsValid: false, ClaimsReason: "expired 1h ago"}
	human, _ = FormatVerify(tok, bad, FormatPretty, true)
	if !strings.Contains(human, "INVALID") {
		t.Errorf("invalid human: %s", human)
	}
}

func TestFormatVerifyTableDistinct(t *testing.T) {
	tok, err := jwt.Parse("eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJzdWIiOiIxMjM0NTY3ODkwIiwibmFtZSI6IkpvaG4gRG9lIiwiaWF0IjoxNTE2MjM5MDIyfQ.SflKxwRJSMeKKF2QT4fwpMeJf36POk6yJV_adQssw5c")
	if err != nil {
		t.Fatal(err)
	}
	res := verify.Result{Alg: "HS256", SignatureValid: true, ClaimsValid: true}
	pretty, _ := FormatVerify(tok, res, FormatPretty, true)
	table, err := FormatVerify(tok, res, FormatTable, true)
	if err != nil {
		t.Fatal(err)
	}
	if pretty == table {
		t.Error("pretty and table must render distinctly")
	}
	if !strings.Contains(table, "Signature: VALID") {
		t.Errorf("table keeps greppable verdicts: %s", table)
	}
	if strings.Contains(table, `"header"`) {
		t.Errorf("table omits the JSON dump: %s", table)
	}
}

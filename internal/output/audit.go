package output

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/yourpwnguy/jwt/internal/audit"
)

// audit.go renders static-analysis reports. JSON is for CI gates
// (`jwt audit -o json | jq`), table is for eyes. Both list every rule —
// passing checks included — so "8 passed" is visible trust, not silence.

// FormatAudit renders one token's report.
// pretty = triage view (summary + failures + collapsed passes).
// table = findings table (one row per check, nothing collapsed).
// json/compact = machine shapes for CI gates.
func FormatAudit(rep audit.Report, format Format, noColor bool) (string, error) {
	switch format {
	case FormatJSON, FormatCompact:
		data, err := marshalAuditJSON(rep, format)
		if err != nil {
			return "", err
		}
		return data, nil
	case FormatTable:
		return formatAuditFindingsTable(rep, noColor), nil
	default:
		return formatAuditTable(rep, noColor), nil
	}
}

// FormatAuditBatch numbers multiple reports like decode does.
// pretty = triage blocks with Token headers. table = merged overview
// (one row per token), so no per-token headers are needed.
func FormatAuditBatch(reps []audit.Report, opts Options) (string, error) {
	format := opts.Format
	if format == "" {
		format = FormatPretty
	}
	if len(reps) == 1 {
		if format == FormatJSON || format == FormatCompact {
			return FormatAudit(reps[0], format, opts.NoColor)
		}
		if format == FormatTable {
			return formatAuditFindingsTable(reps[0], opts.NoColor), nil
		}
		return formatAuditTableWithFailOn(reps[0], opts.NoColor, opts.FailOn), nil
	}
	var b strings.Builder
	showIndex := len(reps) > 1 && !opts.Quiet
	p := newPalette(opts.NoColor)
	if format == FormatTable {
		return formatAuditOverviewTable(reps, opts), nil
	}
	for i, rep := range reps {
		var one string
		if format == FormatJSON || format == FormatCompact {
			var err error
			one, err = FormatAudit(rep, format, opts.NoColor)
			if err != nil {
				return "", err
			}
		} else {
			one = formatAuditTableWithFailOn(rep, opts.NoColor, opts.FailOn)
		}
		if showIndex {
			fmt.Fprintf(&b, "%s %s\n", p.gray(symDot), p.bold(fmt.Sprintf("Token %d", i+1)))
		}
		b.WriteString(one)
		if i < len(reps)-1 {
			b.WriteString("\n\n")
		}
	}
	return b.String(), nil
}

func marshalAuditJSON(rep audit.Report, format Format) (string, error) {
	if format == FormatCompact {
		data, err := json.Marshal(rep)
		return string(data), err
	}
	data, err := json.MarshalIndent(rep, "", "  ")
	return string(data), err
}

// formatAuditTable leads with a summary, lists failures first (sorted by
// severity), then collapses passes to one dim line. Triage reads top-down:
// verdict → what's broken → what passed. Pass IDs stay visible for trust.
func formatAuditTable(rep audit.Report, noColor bool) string {
	return formatAuditTableWithFailOn(rep, noColor, "")
}

// formatAuditTableWithFailOn adds the gate threshold to the header.
func formatAuditTableWithFailOn(rep audit.Report, noColor bool, failOn string) string {
	p := newPalette(noColor)
	var b strings.Builder
	failed := rep.FailedCount()
	total := len(rep.Findings)
	if failed == 0 {
		b.WriteString(p.green(symOK + " PASS " + itoa(total) + "/" + itoa(total)))
	} else {
		b.WriteString(p.red(symFail + " FAIL " + itoa(failed) + "/" + itoa(total)))
	}
	if failOn != "" {
		b.WriteString(p.gray(" · fail-on " + failOn))
	}
	b.WriteByte('\n')
	b.WriteString(p.rule(52) + "\n")

	// Failures first, sorted by severity (critical on top). Passes keep
	// rule order — stable and predictable.
	var fails, passes []audit.Finding
	for _, f := range rep.Findings {
		if f.Pass {
			passes = append(passes, f)
		} else {
			fails = append(fails, f)
		}
	}
	for i := 0; i < len(fails); i++ {
		for j := i + 1; j < len(fails); j++ {
			if fails[j].Severity.Rank() > fails[i].Severity.Rank() {
				fails[i], fails[j] = fails[j], fails[i]
			}
		}
	}
	for _, f := range fails {
		sev := p.red(strings.ToUpper(string(f.Severity)))
		if f.Severity == audit.SeverityMedium || f.Severity == audit.SeverityLow {
			sev = p.yellow(strings.ToUpper(string(f.Severity)))
		}
		fmt.Fprintf(&b, "%s %s  %-16s %s\n", p.red(symFail), sev, f.ID, f.Detail)
	}
	if len(fails) > 0 {
		b.WriteString(p.rule(52) + "\n")
	}
	if len(passes) > 0 {
		ids := make([]string, 0, len(passes))
		for _, f := range passes {
			ids = append(ids, f.ID)
		}
		fmt.Fprintf(&b, "%s\n", p.gray("  "+symOK+" "+strings.Join(ids, " · ")+fmt.Sprintf(" (%d passed)", len(passes))))
	}
	return strings.TrimSuffix(b.String(), "\n")
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	var buf [8]byte
	pos := len(buf)
	for n > 0 {
		pos--
		buf[pos] = byte('0' + n%10)
		n /= 10
	}
	return string(buf[pos:])
}

// formatAuditFindingsTable renders every check as a columnar row.
// Unlike pretty (failures first, passes collapsed), nothing is hidden:
// STATUS  SEVERITY  ID  DETAIL. This is the `-o table` view.
func formatAuditFindingsTable(rep audit.Report, noColor bool) string {
	p := newPalette(noColor)
	type row struct{ status, sev, id, detail string }
	rows := make([]row, 0, len(rep.Findings))
	for _, f := range rep.Findings {
		status := p.green(symOK + " pass")
		if !f.Pass {
			status = p.red(symFail + " FAIL")
		}
		sev := strings.ToUpper(string(f.Severity))
		rows = append(rows, row{status, sev, f.ID, f.Detail})
	}
	widths := [4]int{len("STATUS"), len("SEVERITY"), len("ID"), len("DETAIL")}
	for _, r := range rows {
		if l := len(stripANSI(r.status)); l > widths[0] {
			widths[0] = l
		}
		if len(r.sev) > widths[1] {
			widths[1] = len(r.sev)
		}
		if len(r.id) > widths[2] {
			widths[2] = len(r.id)
		}
	}
	var b strings.Builder
	fmt.Fprintf(&b, "%-*s  %-*s  %-*s  %s\n",
		widths[0], "STATUS", widths[1], "SEVERITY", widths[2], "ID", "DETAIL")
	b.WriteString(p.rule(60) + "\n")
	for _, r := range rows {
		fmt.Fprintf(&b, "%-*s  %-*s  %-*s  %s\n",
			widths[0], r.status, widths[1], r.sev, widths[2], r.id, r.detail)
	}
	return strings.TrimSuffix(b.String(), "\n")
}

// formatAuditOverviewTable renders batch triage: one row per token.
// TOKEN  VERDICT  FAILS, where FAILS lists breached check IDs or "-" when clean.
func formatAuditOverviewTable(reps []audit.Report, opts Options) string {
	p := newPalette(opts.NoColor)
	var b strings.Builder
	b.WriteString("TOKEN  VERDICT   FAILS\n")
	b.WriteString(p.rule(60) + "\n")
	for i, rep := range reps {
		verdict := p.green(symOK + " pass")
		fails := "-"
		if rep.FailedCount() > 0 {
			verdict = p.red(symFail + " FAIL")
			ids := make([]string, 0, rep.FailedCount())
			for _, f := range rep.Findings {
				if !f.Pass {
					ids = append(ids, f.ID)
				}
			}
			fails = strings.Join(ids, ", ")
		}
		fmt.Fprintf(&b, "%-5d  %-12s  %s\n", i+1, verdict, fails)
	}
	return strings.TrimSuffix(b.String(), "\n")
}

package main

import (
	"encoding/json"
	"fmt"
	"sort"
	"strconv"
	"strings"
	"text/tabwriter"
)

// prettyQuery renders a query result's JSON as an aligned, human-readable table for the
// supervised (non-agent) workflow (`query <kind> --pretty`). It is a PRESENTATION layer only:
// the underlying JSON — the agent/scripting contract — is unchanged and stays the default.
// Returns (rendered, true) for the kinds it formats; (_, false) for anything it does not
// recognize (or malformed JSON), so the caller falls back to the raw JSON and --pretty never
// drops data.
func prettyQuery(kind string, raw json.RawMessage) (string, bool) {
	var doc map[string]any
	if err := json.Unmarshal(raw, &doc); err != nil {
		return "", false
	}
	switch kind {
	case "summary":
		return prettySummary(doc), true
	case "findings":
		return prettyFindings(doc), true
	case "live_files", "recent", "orphans", "uncollapsed", "adoption":
		return prettyList(kind, doc), true
	default:
		return "", false
	}
}

// arrayKey maps a list-kind to the JSON field holding its rows.
var arrayKey = map[string]string{
	"live_files":  "files",
	"recent":      "files",
	"orphans":     "files",
	"uncollapsed": "findings",
	"adoption":    "rows",
}

// colOrder is the stable left-to-right column order; only columns present in the rows print.
var colOrder = []string{"date", "event", "path", "dir_kind", "entity_type", "status", "graduated_to", "git_last_commit", "detail"}

func prettyList(kind string, doc map[string]any) string {
	rows, _ := doc[arrayKey[kind]].([]any)
	var b strings.Builder
	fmt.Fprintf(&b, "%s: %s\n", kind, numStr(doc["count"]))
	if len(rows) == 0 {
		return strings.TrimRight(b.String(), "\n")
	}
	cols := presentColumns(rows)
	tw := tabwriter.NewWriter(&b, 0, 2, 2, ' ', 0)
	fmt.Fprintln(tw, strings.Join(cols, "\t"))
	for _, r := range rows {
		obj, _ := r.(map[string]any)
		cells := make([]string, len(cols))
		for i, c := range cols {
			cells[i] = str(obj[c])
		}
		fmt.Fprintln(tw, strings.Join(cells, "\t"))
	}
	_ = tw.Flush()
	return strings.TrimRight(b.String(), "\n")
}

// presentColumns returns colOrder filtered to the keys that appear on any row (so an all-empty
// optional column like graduated_to is dropped rather than printed as a blank column).
func presentColumns(rows []any) []string {
	present := map[string]bool{}
	for _, r := range rows {
		if obj, ok := r.(map[string]any); ok {
			for k, v := range obj {
				if s := str(v); s != "" {
					present[k] = true
				}
			}
		}
	}
	var cols []string
	for _, c := range colOrder {
		if present[c] {
			cols = append(cols, c)
		}
	}
	return cols
}

func prettyFindings(doc map[string]any) string {
	byRule, _ := doc["by_rule"].(map[string]any)
	var b strings.Builder
	fmt.Fprintf(&b, "findings: %s\n", numStr(doc["count"]))
	rules := make([]string, 0, len(byRule))
	for r := range byRule {
		rules = append(rules, r)
	}
	sort.Strings(rules)
	for _, rule := range rules {
		items, _ := byRule[rule].([]any)
		fmt.Fprintf(&b, "\n%s (%d)\n", rule, len(items))
		tw := tabwriter.NewWriter(&b, 0, 2, 2, ' ', 0)
		for _, it := range items {
			obj, _ := it.(map[string]any)
			fmt.Fprintf(tw, "  %s\t%s\n", str(obj["path"]), str(obj["detail"]))
		}
		_ = tw.Flush()
	}
	return strings.TrimRight(b.String(), "\n")
}

func prettySummary(doc map[string]any) string {
	var b strings.Builder
	fmt.Fprintf(&b, "files: %s total\n", numStr(doc["files_total"]))
	writeCountMap(&b, "by dir_kind", doc["files_by_dir_kind"])
	fmt.Fprintf(&b, "open findings: %s total\n", numStr(doc["open_findings_total"]))
	writeCountMap(&b, "by rule", doc["open_findings_by_rule"])
	writeCountMap(&b, "by severity", doc["open_findings_by_severity"])
	return strings.TrimRight(b.String(), "\n")
}

// writeCountMap prints a string→number aggregate as sorted, aligned "  key  n" lines.
func writeCountMap(b *strings.Builder, label string, v any) {
	m, _ := v.(map[string]any)
	if len(m) == 0 {
		return
	}
	fmt.Fprintf(b, "  %s:\n", label)
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	tw := tabwriter.NewWriter(b, 0, 2, 2, ' ', 0)
	for _, k := range keys {
		fmt.Fprintf(tw, "    %s\t%s\n", k, numStr(m[k]))
	}
	_ = tw.Flush()
}

// numStr formats a JSON number (decoded as float64) as an integer string; passes non-numbers
// through via str.
func numStr(v any) string {
	if f, ok := v.(float64); ok {
		return strconv.FormatInt(int64(f), 10)
	}
	return str(v)
}

func str(v any) string {
	if v == nil {
		return ""
	}
	if s, ok := v.(string); ok {
		return s
	}
	return fmt.Sprintf("%v", v)
}

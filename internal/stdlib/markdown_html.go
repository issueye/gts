package stdlib

import (
	"html"
	"regexp"
	"strings"
)

var (
	htmlDropBlocksRE = regexp.MustCompile(`(?is)<script[^>]*>.*?</script>|<style[^>]*>.*?</style>|<nav[^>]*>.*?</nav>|<noscript[^>]*>.*?</noscript>`)
	htmlHeadingRE    = regexp.MustCompile(`(?is)<h([1-6])[^>]*>(.*?)</h[1-6]>`)
	htmlPreCodeRE    = regexp.MustCompile(`(?is)<pre[^>]*>\s*<code[^>]*>(.*?)</code>\s*</pre>`)
	htmlPreRE        = regexp.MustCompile(`(?is)<pre[^>]*>(.*?)</pre>`)
	htmlCodeRE       = regexp.MustCompile(`(?is)<code[^>]*>(.*?)</code>`)
	htmlLinkRE       = regexp.MustCompile(`(?is)<a\s+[^>]*href=["']([^"']+)["'][^>]*>(.*?)</a>`)
	htmlListItemRE   = regexp.MustCompile(`(?is)<li[^>]*>(.*?)</li>`)
	htmlTableRowRE   = regexp.MustCompile(`(?is)<tr[^>]*>(.*?)</tr>`)
	htmlTableCellRE  = regexp.MustCompile(`(?is)<t[hd][^>]*>(.*?)</t[hd]>`)
	htmlTagRE        = regexp.MustCompile(`(?is)<[^>]+>`)
	htmlBlankLinesRE = regexp.MustCompile(`\n{3,}`)
)

func htmlToMarkdown(source string, opts markdownHTMLOptions) string {
	if opts.maxChars <= 0 {
		opts.maxChars = 20000
	}
	s := htmlDropBlocksRE.ReplaceAllString(source, "")
	s = htmlPreCodeRE.ReplaceAllStringFunc(s, func(match string) string {
		inner := htmlPreCodeRE.FindStringSubmatch(match)
		if len(inner) < 2 {
			return ""
		}
		return "\n\n```\n" + htmlText(inner[1]) + "\n```\n\n"
	})
	s = htmlPreRE.ReplaceAllStringFunc(s, func(match string) string {
		inner := htmlPreRE.FindStringSubmatch(match)
		if len(inner) < 2 {
			return ""
		}
		return "\n\n```\n" + htmlText(inner[1]) + "\n```\n\n"
	})
	s = htmlHeadingRE.ReplaceAllStringFunc(s, func(match string) string {
		inner := htmlHeadingRE.FindStringSubmatch(match)
		if len(inner) < 3 {
			return ""
		}
		level := int(inner[1][0] - '0')
		return "\n\n" + strings.Repeat("#", level) + " " + htmlText(inner[2]) + "\n\n"
	})
	s = replaceHTMLTables(s)
	s = htmlListItemRE.ReplaceAllStringFunc(s, func(match string) string {
		inner := htmlListItemRE.FindStringSubmatch(match)
		if len(inner) < 2 {
			return ""
		}
		return "\n- " + htmlText(inner[1])
	})
	if opts.includeLinks {
		s = htmlLinkRE.ReplaceAllStringFunc(s, func(match string) string {
			inner := htmlLinkRE.FindStringSubmatch(match)
			if len(inner) < 3 {
				return ""
			}
			url := strings.TrimSpace(inner[1])
			if opts.baseURL != "" && strings.HasPrefix(url, "/") {
				url = strings.TrimRight(opts.baseURL, "/") + url
			}
			return "[" + htmlText(inner[2]) + "](" + url + ")"
		})
	} else {
		s = htmlLinkRE.ReplaceAllString(s, "$2")
	}
	replacements := []struct {
		old string
		new string
	}{
		{"</p>", "\n\n"},
		{"<br>", "\n"},
		{"<br/>", "\n"},
		{"<br />", "\n"},
		{"</div>", "\n\n"},
		{"</section>", "\n\n"},
		{"</article>", "\n\n"},
		{"</ul>", "\n\n"},
		{"</ol>", "\n\n"},
	}
	for _, repl := range replacements {
		s = strings.ReplaceAll(s, repl.old, repl.new)
	}
	s = htmlCodeRE.ReplaceAllStringFunc(s, func(match string) string {
		inner := htmlCodeRE.FindStringSubmatch(match)
		if len(inner) < 2 {
			return ""
		}
		return "`" + htmlText(inner[1]) + "`"
	})
	s = htmlText(s)
	s = htmlBlankLinesRE.ReplaceAllString(s, "\n\n")
	s = strings.TrimSpace(s)
	if len([]rune(s)) > opts.maxChars {
		runes := []rune(s)
		s = string(runes[:opts.maxChars])
	}
	return s
}

func replaceHTMLTables(source string) string {
	tableRE := regexp.MustCompile(`(?is)<table[^>]*>(.*?)</table>`)
	return tableRE.ReplaceAllStringFunc(source, func(match string) string {
		rows := htmlTableRowRE.FindAllStringSubmatch(match, -1)
		var out []string
		for i, row := range rows {
			cells := htmlTableCellRE.FindAllStringSubmatch(row[1], -1)
			values := make([]string, 0, len(cells))
			for _, cell := range cells {
				values = append(values, htmlText(cell[1]))
			}
			if len(values) == 0 {
				continue
			}
			out = append(out, "| "+strings.Join(values, " | ")+" |")
			if i == 0 {
				sep := make([]string, len(values))
				for j := range sep {
					sep[j] = "---"
				}
				out = append(out, "| "+strings.Join(sep, " | ")+" |")
			}
		}
		return "\n\n" + strings.Join(out, "\n") + "\n\n"
	})
}

func htmlText(source string) string {
	s := htmlTagRE.ReplaceAllString(source, "")
	s = html.UnescapeString(s)
	lines := strings.Split(s, "\n")
	for i, line := range lines {
		lines[i] = strings.Join(strings.Fields(line), " ")
	}
	return strings.TrimSpace(strings.Join(lines, "\n"))
}

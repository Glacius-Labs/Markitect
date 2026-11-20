package state

// This file ports Git's wildmatch.c, which matches the patterns of .gitignore
// files: byte by byte, with case.

const (
	wmMatch = iota
	wmNoMatch
	wmAbortAll
	wmAbortToStarStar
)

// wildmatch says whether text matches pattern as Git's wildmatch does.
// pathname is Git's WM_PATHNAME: "*" and "?" then do not match "/", and "**"
// between slashes matches any number of folders.
func wildmatch(pattern, text string, pathname bool) bool {
	return dowild(pattern, text, pathname) == wmMatch
}

// at returns s[i], or 0 past its end, as a C string reads.
func at(s string, i int) byte {
	if i < len(s) {
		return s[i]
	}
	return 0
}

func dowild(pat, text string, pathname bool) int {
	p, t := 0, 0
	for ; p < len(pat); p, t = p+1, t+1 {
		pc := pat[p]
		tc := at(text, t)
		if t >= len(text) && pc != '*' {
			return wmAbortAll
		}
		switch pc {
		case '\\':
			// A literal match with the next character; a backslash at the
			// end matches nothing.
			p++
			if p >= len(pat) || tc != pat[p] {
				return wmNoMatch
			}
		default:
			if tc != pc {
				return wmNoMatch
			}
		case '?':
			if pathname && tc == '/' {
				return wmNoMatch
			}
		case '*':
			var matchSlash bool
			p++
			if at(pat, p) == '*' {
				prev := p - 2
				for p++; at(pat, p) == '*'; p++ {
				}
				if (prev < 0 || pat[prev] == '/') &&
					(p >= len(pat) || pat[p] == '/' || at(pat, p) == '\\' && at(pat, p+1) == '/') {
					if at(pat, p) == '/' && dowild(pat[p+1:], text[t:], pathname) == wmMatch {
						return wmMatch
					}
					matchSlash = true
				} else {
					matchSlash = !pathname
				}
			} else {
				matchSlash = !pathname
			}
			if p >= len(pat) {
				// A trailing "**" matches everything, a trailing "*" only
				// what holds no more slash.
				if !matchSlash && containsSlash(text[t:]) {
					return wmNoMatch
				}
				return wmMatch
			}
			if !matchSlash && pat[p] == '/' {
				// One star and a slash match the next folder.
				i := indexSlash(text[t:])
				if i < 0 {
					return wmNoMatch
				}
				t += i
				// The slash is matched by the loop.
				continue
			}
			for {
				if t >= len(text) {
					break
				}
				if !globSpecial(pat[p]) {
					// The text up to the next literal belongs to the star.
					for t < len(text) && (matchSlash || text[t] != '/') && text[t] != pat[p] {
						t++
					}
					if at(text, t) != pat[p] {
						return wmNoMatch
					}
				}
				m := dowild(pat[p:], text[t:], pathname)
				if m != wmNoMatch {
					if !matchSlash || m != wmAbortToStarStar {
						return m
					}
				} else if !matchSlash && text[t] == '/' {
					return wmAbortToStarStar
				}
				t++
			}
			return wmAbortAll
		case '[':
			p++
			c := at(pat, p)
			if c == '^' {
				c = '!'
			}
			negated := c == '!'
			if negated {
				p++
				c = at(pat, p)
			}
			var prev byte
			matched := false
			for {
				if p >= len(pat) {
					return wmAbortAll
				}
				switch {
				case c == '\\':
					p++
					if p >= len(pat) {
						return wmAbortAll
					}
					c = pat[p]
					if tc == c {
						matched = true
					}
				case c == '-' && prev != 0 && p+1 < len(pat) && pat[p+1] != ']':
					p++
					c = pat[p]
					if c == '\\' {
						p++
						if p >= len(pat) {
							return wmAbortAll
						}
						c = pat[p]
					}
					if tc <= c && tc >= prev {
						matched = true
					}
					c = 0
				case c == '[' && at(pat, p+1) == ':':
					s := p + 2
					e := s
					for e < len(pat) && pat[e] != ']' {
						e++
					}
					if e >= len(pat) {
						return wmAbortAll
					}
					if e-s-1 < 0 || pat[e-1] != ':' {
						// No ":]": a plain "[".
						c = '['
						if tc == c {
							matched = true
						}
						break
					}
					class, ok := classes[pat[s:e-1]]
					if !ok {
						return wmAbortAll
					}
					if class(tc) {
						matched = true
					}
					p = e
					c = 0
				default:
					if tc == c {
						matched = true
					}
				}
				prev = c
				p++
				c = at(pat, p)
				if c == ']' {
					break
				}
			}
			if matched == negated || pathname && tc == '/' {
				return wmNoMatch
			}
		}
	}
	if t < len(text) {
		return wmNoMatch
	}
	return wmMatch
}

func containsSlash(s string) bool { return indexSlash(s) >= 0 }

func indexSlash(s string) int {
	for i := 0; i < len(s); i++ {
		if s[i] == '/' {
			return i
		}
	}
	return -1
}

// globSpecial says whether a pattern byte has a meaning of its own.
func globSpecial(c byte) bool { return c == '*' || c == '?' || c == '[' || c == '\\' }

// classes are the POSIX character classes, over ASCII as Git defines them.
var classes = map[string]func(byte) bool{
	"alnum":  func(c byte) bool { return isAlpha(c) || isDigit(c) },
	"alpha":  isAlpha,
	"blank":  func(c byte) bool { return c == ' ' || c == '\t' },
	"cntrl":  func(c byte) bool { return c < 0x20 || c == 0x7f },
	"digit":  isDigit,
	"graph":  func(c byte) bool { return c > 0x20 && c < 0x7f },
	"lower":  func(c byte) bool { return c >= 'a' && c <= 'z' },
	"print":  func(c byte) bool { return c >= 0x20 && c < 0x7f },
	"punct":  func(c byte) bool { return c > 0x20 && c < 0x7f && !isAlpha(c) && !isDigit(c) },
	"space":  func(c byte) bool { return c == ' ' || c == '\t' || c == '\n' || c == '\r' },
	"upper":  func(c byte) bool { return c >= 'A' && c <= 'Z' },
	"xdigit": func(c byte) bool { return isDigit(c) || c >= 'a' && c <= 'f' || c >= 'A' && c <= 'F' },
}

func isAlpha(c byte) bool { return c >= 'a' && c <= 'z' || c >= 'A' && c <= 'Z' }
func isDigit(c byte) bool { return c >= '0' && c <= '9' }

// Package slug derives short, human-readable, URL-safe ids from Bengali names.
//
// A name is romanized with a best-effort phonetic map, then the shortest unique
// form is chosen: given name -> given-surname -> full -> given-parent -> numeric.
// Ids are opaque once assigned; readability is a bonus, not a guarantee.
package slug

import (
	"strconv"
	"strings"
)

// independent vowels
var vowel = map[rune]string{
	'অ': "o", 'আ': "a", 'ই': "i", 'ঈ': "i", 'উ': "u", 'ঊ': "u",
	'ঋ': "ri", 'এ': "e", 'ঐ': "oi", 'ও': "o", 'ঔ': "ou",
}

// dependent vowel signs (kar)
var kar = map[rune]string{
	'া': "a", 'ি': "i", 'ী': "i", 'ু': "u", 'ূ': "u",
	'ৃ': "ri", 'ে': "e", 'ৈ': "oi", 'ো': "o", 'ৌ': "ou",
}

// consonants (incl. precomposed nukta glyphs and our fold placeholders)
var cons = map[rune]string{
	'ক': "k", 'খ': "kh", 'গ': "g", 'ঘ': "gh", 'ঙ': "ng",
	'চ': "ch", 'ছ': "chh", 'জ': "j", 'ঝ': "jh", 'ঞ': "n",
	'ট': "t", 'ঠ': "th", 'ড': "d", 'ঢ': "dh", 'ণ': "n",
	'ত': "t", 'থ': "th", 'দ': "d", 'ধ': "dh", 'ন': "n",
	'প': "p", 'ফ': "f", 'ব': "b", 'ভ': "bh", 'ম': "m",
	'য': "j", 'র': "r", 'ল': "l", 'শ': "sh", 'ষ': "sh", 'স': "s", 'হ': "h",
	'ৎ': "t",
	'য়': "y", 'ড়': "r", 'ঢ়': "rh", // precomposed য় ড় ঢ়
	'Ɏ': "y", 'Ɍ': "r", 'Ʀ': "rh", // fold placeholders
}

const (
	virama   = '্' // ্ hasanta
	nukta    = '়' // ় combining nukta
	inherent = "o"      // inherent vowel for a bare medial consonant
)

// base consonant -> placeholder when followed by a nukta
var nukFold = map[rune]rune{'য': 'Ɏ', 'জ': 'Ɏ', 'ড': 'Ɍ', 'ঢ': 'Ʀ'}

func skip(r rune) bool {
	switch r {
	case 'ঁ', 'ং', 'ঃ', '‌', '‍':
		return true
	}
	return false
}

// token-level overrides for names that have a conventional romanization
var override = map[string]string{"কাজী": "kazi", "আলী": "ali", "আলি": "ali"}

// romanizeToken transliterates a single whitespace-delimited token.
func romanizeToken(tok string) string {
	if o, ok := override[tok]; ok {
		return o
	}
	// fold base-consonant + nukta into a single placeholder rune
	in := []rune(tok)
	ch := make([]rune, 0, len(in))
	for _, r := range in {
		if r == nukta && len(ch) > 0 {
			if p, ok := nukFold[ch[len(ch)-1]]; ok {
				ch[len(ch)-1] = p
				continue
			}
		}
		ch = append(ch, r)
	}

	var b strings.Builder
	for i := 0; i < len(ch); i++ {
		c := ch[i]
		if skip(c) {
			continue
		}
		if v, ok := vowel[c]; ok {
			b.WriteString(v)
			continue
		}
		if v, ok := kar[c]; ok {
			b.WriteString(v)
			continue
		}
		if cn, ok := cons[c]; ok {
			b.WriteString(cn)
			if i+1 < len(ch) {
				nxt := ch[i+1]
				if nxt == virama { // conjunct: suppress vowel, consume virama
					i++
					continue
				}
				if _, isKar := kar[nxt]; isKar { // explicit vowel follows
					continue
				}
			}
			// bare consonant: inherent vowel unless it's the last sounded rune
			last := true
			for j := i + 1; j < len(ch); j++ {
				if !skip(ch[j]) {
					last = false
					break
				}
			}
			if !last {
				b.WriteString(inherent)
			}
		}
	}
	return b.String()
}

// Tokens returns the romanized, lower-cased tokens of a name.
func Tokens(name string) []string {
	var out []string
	for _, t := range strings.Fields(name) {
		if s := strings.ToLower(romanizeToken(t)); s != "" {
			out = append(out, s)
		}
	}
	return out
}

// Generate returns the shortest id for name that is not already in taken, then
// records it in taken. parentName may be "" (root). The candidate ladder is:
// given -> given-last -> full -> given-parentgiven -> given-N.
func Generate(name, parentName string, taken map[string]bool) string {
	toks := Tokens(name)
	if len(toks) == 0 {
		toks = []string{"x"}
	}
	first := toks[0]
	last := toks[len(toks)-1]

	cands := []string{first}
	if last != first {
		cands = append(cands, first+"-"+last)
	}
	if len(toks) > 2 {
		cands = append(cands, strings.Join(toks, "-"))
	}
	if pt := Tokens(parentName); len(pt) > 0 {
		cands = append(cands, first+"-"+pt[0])
	}
	for _, c := range cands {
		if !taken[c] {
			taken[c] = true
			return c
		}
	}
	for n := 2; ; n++ {
		c := first + "-" + strconv.Itoa(n)
		if !taken[c] {
			taken[c] = true
			return c
		}
	}
}

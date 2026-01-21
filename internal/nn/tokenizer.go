package nn

import (
	"regexp"
	"strings"
	"unicode"
)

// Tokenizer handles text tokenization for the neural network
type Tokenizer struct {
	wordToIdx    map[string]int
	idxToWord    map[int]string
	vocabSize    int
	maxSeqLen    int
	specialTokens map[string]int
}

// Special tokens
const (
	PadToken   = "<PAD>"
	UnkToken   = "<UNK>"
	StartToken = "<START>"
	EndToken   = "<END>"
)

// NewTokenizer creates a new tokenizer
func NewTokenizer(maxSeqLen int) *Tokenizer {
	t := &Tokenizer{
		wordToIdx: make(map[string]int),
		idxToWord: make(map[int]string),
		maxSeqLen: maxSeqLen,
		specialTokens: map[string]int{
			PadToken:   0,
			UnkToken:   1,
			StartToken: 2,
			EndToken:   3,
		},
	}

	// Initialize special tokens
	for token, idx := range t.specialTokens {
		t.wordToIdx[token] = idx
		t.idxToWord[idx] = token
	}
	t.vocabSize = len(t.specialTokens)

	return t
}

// Tokenize splits text into tokens
func (t *Tokenizer) Tokenize(text string) []string {
	// Convert to lowercase
	text = strings.ToLower(text)

	// Normalize SQL keywords
	text = normalizeSQLKeywords(text)

	// Split on whitespace and punctuation while preserving SQL operators
	var tokens []string
	var current strings.Builder

	for i, r := range text {
		if unicode.IsLetter(r) || unicode.IsDigit(r) || r == '_' {
			current.WriteRune(r)
		} else if r == '.' {
			// Check if part of a qualified name (schema.table)
			if current.Len() > 0 {
				tokens = append(tokens, current.String())
				current.Reset()
			}
			tokens = append(tokens, ".")
		} else if isSQLOperator(text, i) {
			if current.Len() > 0 {
				tokens = append(tokens, current.String())
				current.Reset()
			}
			// Get the operator
			op := getSQLOperator(text, i)
			if op != "" {
				tokens = append(tokens, op)
			}
		} else if unicode.IsSpace(r) {
			if current.Len() > 0 {
				tokens = append(tokens, current.String())
				current.Reset()
			}
		} else {
			// Other punctuation
			if current.Len() > 0 {
				tokens = append(tokens, current.String())
				current.Reset()
			}
			tokens = append(tokens, string(r))
		}
	}

	if current.Len() > 0 {
		tokens = append(tokens, current.String())
	}

	return tokens
}

// normalizeSQLKeywords normalizes common SQL keywords
func normalizeSQLKeywords(text string) string {
	keywords := []string{
		"SELECT", "FROM", "WHERE", "AND", "OR", "NOT",
		"INSERT", "UPDATE", "DELETE", "INTO", "VALUES",
		"JOIN", "LEFT", "RIGHT", "INNER", "OUTER", "ON",
		"GROUP", "BY", "ORDER", "HAVING", "LIMIT", "OFFSET",
		"AS", "IN", "IS", "NULL", "LIKE", "BETWEEN",
		"COUNT", "SUM", "AVG", "MIN", "MAX", "DISTINCT",
		"CREATE", "TABLE", "INDEX", "ALTER", "DROP",
		"PRIMARY", "KEY", "FOREIGN", "REFERENCES",
		"ST_DISTANCE", "ST_WITHIN", "ST_CONTAINS", "ST_INTERSECTS",
		"ST_AREA", "ST_LENGTH", "ST_BUFFER", "ST_TRANSFORM",
	}

	for _, kw := range keywords {
		// Case-insensitive replacement with lowercase
		re := regexp.MustCompile("(?i)" + regexp.QuoteMeta(kw))
		text = re.ReplaceAllString(text, strings.ToLower(kw))
	}

	return text
}

// isSQLOperator checks if position starts an SQL operator
func isSQLOperator(text string, i int) bool {
	if i >= len(text) {
		return false
	}
	r := rune(text[i])
	return r == '=' || r == '<' || r == '>' || r == '!' ||
		r == '(' || r == ')' || r == ',' || r == ';' ||
		r == '*' || r == '+' || r == '-' || r == '/'
}

// getSQLOperator extracts SQL operator at position
func getSQLOperator(text string, i int) string {
	if i >= len(text) {
		return ""
	}

	// Check for two-character operators
	if i+1 < len(text) {
		twoChar := text[i : i+2]
		switch twoChar {
		case "<=", ">=", "<>", "!=", "::":
			return twoChar
		}
	}

	return string(text[i])
}

// BuildVocabulary builds vocabulary from training data
func (t *Tokenizer) BuildVocabulary(texts []string) {
	wordFreq := make(map[string]int)

	for _, text := range texts {
		tokens := t.Tokenize(text)
		for _, token := range tokens {
			wordFreq[token]++
		}
	}

	// Add words to vocabulary (filter by minimum frequency)
	minFreq := 1
	for word, freq := range wordFreq {
		if freq >= minFreq && t.wordToIdx[word] == 0 && word != PadToken {
			idx := t.vocabSize
			t.wordToIdx[word] = idx
			t.idxToWord[idx] = word
			t.vocabSize++
		}
	}
}

// Encode converts tokens to indices
func (t *Tokenizer) Encode(text string) []int {
	tokens := t.Tokenize(text)
	encoded := make([]int, 0, len(tokens)+2) // +2 for START and END

	// Add START token
	encoded = append(encoded, t.specialTokens[StartToken])

	for _, token := range tokens {
		if idx, ok := t.wordToIdx[token]; ok {
			encoded = append(encoded, idx)
		} else {
			encoded = append(encoded, t.specialTokens[UnkToken])
		}
	}

	// Add END token
	encoded = append(encoded, t.specialTokens[EndToken])

	// Pad or truncate to maxSeqLen
	return t.PadSequence(encoded)
}

// PadSequence pads or truncates sequence to maxSeqLen
func (t *Tokenizer) PadSequence(seq []int) []int {
	if len(seq) >= t.maxSeqLen {
		return seq[:t.maxSeqLen]
	}

	padded := make([]int, t.maxSeqLen)
	copy(padded, seq)
	// Rest is already 0 (PAD token)
	return padded
}

// Decode converts indices back to text
func (t *Tokenizer) Decode(indices []int) string {
	var tokens []string

	for _, idx := range indices {
		if word, ok := t.idxToWord[idx]; ok {
			// Skip special tokens in output
			if word == PadToken || word == StartToken {
				continue
			}
			if word == EndToken {
				break
			}
			tokens = append(tokens, word)
		}
	}

	return joinTokens(tokens)
}

// joinTokens joins tokens back into text with proper spacing
func joinTokens(tokens []string) string {
	var result strings.Builder

	for i, token := range tokens {
		// Don't add space before punctuation or after opening paren
		if i > 0 && !isPunctuation(token) && !isClosingParen(tokens[i-1]) {
			if !isOpeningParen(token) {
				result.WriteString(" ")
			}
		}
		result.WriteString(token)
	}

	return result.String()
}

func isPunctuation(s string) bool {
	if len(s) != 1 {
		return false
	}
	return s == "," || s == ";" || s == "." || s == ")" || s == ":"
}

func isOpeningParen(s string) bool {
	return s == "("
}

func isClosingParen(s string) bool {
	return s == ")"
}

// VocabSize returns the vocabulary size
func (t *Tokenizer) VocabSize() int {
	return t.vocabSize
}

// MaxSeqLen returns the maximum sequence length
func (t *Tokenizer) MaxSeqLen() int {
	return t.maxSeqLen
}

// GetWordIndex returns the index for a word
func (t *Tokenizer) GetWordIndex(word string) int {
	if idx, ok := t.wordToIdx[word]; ok {
		return idx
	}
	return t.specialTokens[UnkToken]
}

// GetIndexWord returns the word for an index
func (t *Tokenizer) GetIndexWord(idx int) string {
	if word, ok := t.idxToWord[idx]; ok {
		return word
	}
	return UnkToken
}

// Export exports the tokenizer vocabulary for persistence
func (t *Tokenizer) Export() map[string]interface{} {
	return map[string]interface{}{
		"word_to_idx": t.wordToIdx,
		"vocab_size":  t.vocabSize,
		"max_seq_len": t.maxSeqLen,
	}
}

// Import imports a tokenizer vocabulary
func (t *Tokenizer) Import(data map[string]interface{}) error {
	if wordToIdx, ok := data["word_to_idx"].(map[string]interface{}); ok {
		t.wordToIdx = make(map[string]int)
		t.idxToWord = make(map[int]string)
		for word, idxVal := range wordToIdx {
			if idx, ok := idxVal.(float64); ok {
				t.wordToIdx[word] = int(idx)
				t.idxToWord[int(idx)] = word
			}
		}
	}

	if vocabSize, ok := data["vocab_size"].(float64); ok {
		t.vocabSize = int(vocabSize)
	}

	if maxSeqLen, ok := data["max_seq_len"].(float64); ok {
		t.maxSeqLen = int(maxSeqLen)
	}

	return nil
}

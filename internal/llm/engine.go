package llm

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/kartoza/kartoza-pg-ai/internal/config"
	"github.com/kartoza/kartoza-pg-ai/internal/nn"
)

// QueryEngine handles natural language to SQL conversion
// This is a simplified rule-based engine augmented with neural network predictions.
type QueryEngine struct {
	schema    *config.SchemaCache
	nnTrainer *nn.QueryTrainer
	useNN     bool // Whether to use NN predictions when available
}

// NewQueryEngine creates a new query engine
func NewQueryEngine(schema *config.SchemaCache) *QueryEngine {
	engine := &QueryEngine{
		schema: schema,
		useNN:  true, // Enable NN by default
	}

	// Initialize neural network trainer
	trainer, err := nn.NewQueryTrainer()
	if err == nil {
		engine.nnTrainer = trainer
	}

	return engine
}

// GenerateSQL converts natural language to SQL
func (e *QueryEngine) GenerateSQL(query string, context string) (string, error) {
	if e.schema == nil {
		return "", fmt.Errorf("no schema loaded")
	}

	// Try neural network prediction first if enabled and trained
	if e.useNN && e.nnTrainer != nil && e.nnTrainer.IsTrained() {
		if nnSQL, confidence, err := e.nnTrainer.Predict(query); err == nil && confidence > 0.6 {
			// Validate the NN-generated SQL is syntactically reasonable
			if isValidSQLStructure(nnSQL) {
				return nnSQL, nil
			}
		}
	}

	// Fall back to rule-based matching
	query = strings.TrimSpace(strings.ToLower(query))

	// Simple pattern matching for common queries
	// In production, replace with actual LLM integration

	// Count queries
	if countMatch := e.matchCountQuery(query); countMatch != "" {
		return countMatch, nil
	}

	// Show/list queries
	if showMatch := e.matchShowQuery(query); showMatch != "" {
		return showMatch, nil
	}

	// Table info queries
	if tableMatch := e.matchTableQuery(query); tableMatch != "" {
		return tableMatch, nil
	}

	// Spatial queries (if PostGIS available)
	if e.schema.HasPostGIS {
		if spatialMatch := e.matchSpatialQuery(query); spatialMatch != "" {
			return spatialMatch, nil
		}
	}

	// Generic select with limit
	if selectMatch := e.matchSelectQuery(query); selectMatch != "" {
		return selectMatch, nil
	}

	// Search/find queries - look for tables/columns matching keywords
	if searchMatch := e.matchSearchQuery(query); searchMatch != "" {
		return searchMatch, nil
	}

	// Fallback: try to extract table name and do basic select
	for _, table := range e.schema.Tables {
		tableName := strings.ToLower(table.Name)
		if strings.Contains(query, tableName) {
			return fmt.Sprintf("SELECT * FROM \"%s\".\"%s\" LIMIT 50", table.Schema, table.Name), nil
		}
	}

	return "", fmt.Errorf("could not understand query: %s", query)
}

func (e *QueryEngine) matchCountQuery(query string) string {
	countPatterns := []string{
		`how many (?:rows|records|entries) (?:are )?in (?:the )?(?:table )?(\w+)`,
		`count (?:of |all )?(?:rows |records )?(?:in )?(?:the )?(\w+)`,
		`(\w+) (?:row |record )?count`,
		`how many (\w+)`,
	}

	// Check for "all tables" or "each table" pattern
	if strings.Contains(query, "each table") || strings.Contains(query, "all tables") {
		var queries []string
		for _, table := range e.schema.Tables {
			queries = append(queries, fmt.Sprintf(
				"SELECT '%s.%s' as table_name, COUNT(*) as row_count FROM \"%s\".\"%s\"",
				table.Schema, table.Name, table.Schema, table.Name,
			))
		}
		if len(queries) > 0 {
			return strings.Join(queries, " UNION ALL ") + " ORDER BY row_count DESC"
		}
	}

	for _, pattern := range countPatterns {
		re := regexp.MustCompile(pattern)
		if matches := re.FindStringSubmatch(query); len(matches) > 1 {
			tableName := matches[1]
			if table := e.findTable(tableName); table != nil {
				return fmt.Sprintf("SELECT COUNT(*) as count FROM \"%s\".\"%s\"", table.Schema, table.Name)
			}
		}
	}

	return ""
}

func (e *QueryEngine) matchShowQuery(query string) string {
	showPatterns := []string{
		`show (?:me )?(?:the )?(?:first )?(\d+)? ?(?:rows |records )?(?:from |of )?(?:the )?(\w+)`,
		`list (?:the )?(?:first )?(\d+)? ?(\w+)`,
		`get (?:the )?(?:first )?(\d+)? ?(\w+)`,
		`display (?:the )?(?:first )?(\d+)? ?(\w+)`,
	}

	for _, pattern := range showPatterns {
		re := regexp.MustCompile(pattern)
		if matches := re.FindStringSubmatch(query); len(matches) > 1 {
			limit := "50"
			tableName := ""

			for i, m := range matches[1:] {
				if m == "" {
					continue
				}
				// Check if it's a number (limit) or table name
				if regexp.MustCompile(`^\d+$`).MatchString(m) {
					limit = m
				} else if i > 0 || !regexp.MustCompile(`^\d+$`).MatchString(matches[1]) {
					tableName = m
				}
			}

			if tableName != "" {
				if table := e.findTable(tableName); table != nil {
					return fmt.Sprintf("SELECT * FROM \"%s\".\"%s\" LIMIT %s", table.Schema, table.Name, limit)
				}
			}
		}
	}

	return ""
}

func (e *QueryEngine) matchTableQuery(query string) string {
	// Table structure queries
	if strings.Contains(query, "tables") && (strings.Contains(query, "list") || strings.Contains(query, "show") || strings.Contains(query, "what")) {
		return `SELECT table_schema, table_name,
			(SELECT COUNT(*) FROM information_schema.columns c WHERE c.table_schema = t.table_schema AND c.table_name = t.table_name) as column_count
			FROM information_schema.tables t
			WHERE table_type = 'BASE TABLE' AND table_schema NOT IN ('pg_catalog', 'information_schema')
			ORDER BY table_schema, table_name`
	}

	// Column info
	colPatterns := []string{
		`what (?:are )?(?:the )?columns (?:in |of )?(?:the )?(\w+)`,
		`describe (?:the )?(\w+)`,
		`schema (?:of |for )?(?:the )?(\w+)`,
		`(\w+) structure`,
	}

	for _, pattern := range colPatterns {
		re := regexp.MustCompile(pattern)
		if matches := re.FindStringSubmatch(query); len(matches) > 1 {
			tableName := matches[1]
			if table := e.findTable(tableName); table != nil {
				return fmt.Sprintf(`SELECT column_name, data_type, is_nullable, column_default
					FROM information_schema.columns
					WHERE table_schema = '%s' AND table_name = '%s'
					ORDER BY ordinal_position`, table.Schema, table.Name)
			}
		}
	}

	// Largest tables
	if strings.Contains(query, "largest") && strings.Contains(query, "table") {
		var queries []string
		for _, table := range e.schema.Tables {
			queries = append(queries, fmt.Sprintf(
				"SELECT '%s.%s' as table_name, COUNT(*) as row_count FROM \"%s\".\"%s\"",
				table.Schema, table.Name, table.Schema, table.Name,
			))
		}
		if len(queries) > 0 {
			return strings.Join(queries, " UNION ALL ") + " ORDER BY row_count DESC LIMIT 10"
		}
	}

	return ""
}

func (e *QueryEngine) matchSpatialQuery(query string) string {
	// Find geometry columns
	var geomTables []config.TableInfo
	for _, table := range e.schema.Tables {
		for _, col := range table.Columns {
			if col.IsGeometry {
				geomTables = append(geomTables, table)
				break
			}
		}
	}

	if len(geomTables) == 0 {
		return ""
	}

	// Distance queries
	distancePatterns := []string{
		`within (\d+(?:\.\d+)?)\s*(?:km|kilometers?|m|meters?|mi|miles?)`,
		`(\d+(?:\.\d+)?)\s*(?:km|kilometers?|m|meters?|mi|miles?) (?:from|of|away)`,
	}

	for _, pattern := range distancePatterns {
		re := regexp.MustCompile(pattern)
		if matches := re.FindStringSubmatch(query); len(matches) > 1 {
			// Simple spatial query template
			table := geomTables[0]
			geomCol := ""
			for _, col := range table.Columns {
				if col.IsGeometry {
					geomCol = col.Name
					break
				}
			}

			if geomCol != "" {
				// Convert distance to meters
				dist := matches[1]
				distMeters := dist
				if strings.Contains(query, "km") || strings.Contains(query, "kilometer") {
					distMeters = dist + " * 1000"
				} else if strings.Contains(query, "mi") || strings.Contains(query, "mile") {
					distMeters = dist + " * 1609.34"
				}

				return fmt.Sprintf(`SELECT * FROM "%s"."%s"
					WHERE ST_DWithin("%s"::geography, ST_MakePoint(0, 0)::geography, %s)
					LIMIT 50`, table.Schema, table.Name, geomCol, distMeters)
			}
		}
	}

	// Area queries
	if strings.Contains(query, "area") || strings.Contains(query, "size") {
		for _, table := range geomTables {
			for _, col := range table.Columns {
				if col.IsGeometry && (col.GeomType == "POLYGON" || col.GeomType == "MULTIPOLYGON") {
					return fmt.Sprintf(`SELECT *, ST_Area("%s"::geography) as area_sqm
						FROM "%s"."%s"
						ORDER BY ST_Area("%s"::geography) DESC
						LIMIT 50`, col.Name, table.Schema, table.Name, col.Name)
				}
			}
		}
	}

	// Length queries
	if strings.Contains(query, "length") || strings.Contains(query, "meters of road") || strings.Contains(query, "distance") {
		for _, table := range geomTables {
			for _, col := range table.Columns {
				if col.IsGeometry && (col.GeomType == "LINESTRING" || col.GeomType == "MULTILINESTRING") {
					if strings.Contains(query, "total") || strings.Contains(query, "sum") {
						return fmt.Sprintf(`SELECT SUM(ST_Length("%s"::geography)) as total_length_meters
							FROM "%s"."%s"`, col.Name, table.Schema, table.Name)
					}
					return fmt.Sprintf(`SELECT *, ST_Length("%s"::geography) as length_meters
						FROM "%s"."%s"
						ORDER BY ST_Length("%s"::geography) DESC
						LIMIT 50`, col.Name, table.Schema, table.Name, col.Name)
				}
			}
		}
	}

	return ""
}

func (e *QueryEngine) matchSelectQuery(query string) string {
	selectPatterns := []string{
		`select (?:all )?(?:from )?(\w+)`,
		`fetch (?:all )?(?:from )?(\w+)`,
		`retrieve (?:all )?(?:from )?(\w+)`,
	}

	for _, pattern := range selectPatterns {
		re := regexp.MustCompile(pattern)
		if matches := re.FindStringSubmatch(query); len(matches) > 1 {
			tableName := matches[1]
			if table := e.findTable(tableName); table != nil {
				return fmt.Sprintf("SELECT * FROM \"%s\".\"%s\" LIMIT 50", table.Schema, table.Name)
			}
		}
	}

	return ""
}

// SemanticMatcher provides intelligent fuzzy matching between keywords and database entities
type SemanticMatcher struct{}

// MatchScore represents how well a keyword matches an entity name
type MatchScore struct {
	Score      float64
	MatchType  string // "exact", "contains", "prefix", "suffix", "stem", "fuzzy", "ngram"
	Keyword    string
	EntityName string
}

// stemWord returns a simplified stem of a word (basic Porter-like stemming)
func stemWord(word string) string {
	word = strings.ToLower(word)

	// Common suffixes to remove
	suffixes := []string{
		"ology", "ological", "ation", "ations", "ment", "ments",
		"ness", "ity", "ies", "ing", "ed", "er", "ers", "est",
		"ly", "al", "ial", "ous", "ive", "able", "ible", "ful",
		"less", "ship", "ward", "wards", "wise",
	}

	// Handle plurals first
	if strings.HasSuffix(word, "ies") && len(word) > 4 {
		word = word[:len(word)-3] + "y"
	} else if strings.HasSuffix(word, "es") && len(word) > 3 {
		word = word[:len(word)-2]
	} else if strings.HasSuffix(word, "s") && len(word) > 2 && !strings.HasSuffix(word, "ss") {
		word = word[:len(word)-1]
	}

	// Remove common suffixes
	for _, suffix := range suffixes {
		if strings.HasSuffix(word, suffix) && len(word)-len(suffix) >= 3 {
			word = word[:len(word)-len(suffix)]
			break
		}
	}

	return word
}

// levenshteinDistance calculates the edit distance between two strings
func levenshteinDistance(s1, s2 string) int {
	if len(s1) == 0 {
		return len(s2)
	}
	if len(s2) == 0 {
		return len(s1)
	}

	// Create matrix
	matrix := make([][]int, len(s1)+1)
	for i := range matrix {
		matrix[i] = make([]int, len(s2)+1)
		matrix[i][0] = i
	}
	for j := range matrix[0] {
		matrix[0][j] = j
	}

	// Fill matrix
	for i := 1; i <= len(s1); i++ {
		for j := 1; j <= len(s2); j++ {
			cost := 1
			if s1[i-1] == s2[j-1] {
				cost = 0
			}
			matrix[i][j] = minInt(
				matrix[i-1][j]+1,      // deletion
				matrix[i][j-1]+1,      // insertion
				matrix[i-1][j-1]+cost, // substitution
			)
		}
	}

	return matrix[len(s1)][len(s2)]
}

func minInt(nums ...int) int {
	min := nums[0]
	for _, n := range nums[1:] {
		if n < min {
			min = n
		}
	}
	return min
}

// generateNgrams generates character n-grams from a string
func generateNgrams(s string, n int) map[string]bool {
	ngrams := make(map[string]bool)
	s = strings.ToLower(s)
	if len(s) < n {
		ngrams[s] = true
		return ngrams
	}
	for i := 0; i <= len(s)-n; i++ {
		ngrams[s[i:i+n]] = true
	}
	return ngrams
}

// ngramSimilarity calculates Jaccard similarity between n-gram sets
func ngramSimilarity(s1, s2 string, n int) float64 {
	ngrams1 := generateNgrams(s1, n)
	ngrams2 := generateNgrams(s2, n)

	if len(ngrams1) == 0 || len(ngrams2) == 0 {
		return 0
	}

	// Calculate intersection
	intersection := 0
	for ng := range ngrams1 {
		if ngrams2[ng] {
			intersection++
		}
	}

	// Calculate union
	union := len(ngrams1) + len(ngrams2) - intersection

	if union == 0 {
		return 0
	}

	return float64(intersection) / float64(union)
}

// splitCamelCase splits camelCase and snake_case into words
func splitEntityName(name string) []string {
	// Replace underscores and hyphens with spaces
	name = strings.ReplaceAll(name, "_", " ")
	name = strings.ReplaceAll(name, "-", " ")

	// Split camelCase
	var result []string
	var current strings.Builder
	for i, r := range name {
		if r == ' ' {
			if current.Len() > 0 {
				result = append(result, strings.ToLower(current.String()))
				current.Reset()
			}
			continue
		}
		// Check for camelCase boundary
		if i > 0 && r >= 'A' && r <= 'Z' {
			if current.Len() > 0 {
				result = append(result, strings.ToLower(current.String()))
				current.Reset()
			}
		}
		current.WriteRune(r)
	}
	if current.Len() > 0 {
		result = append(result, strings.ToLower(current.String()))
	}

	return result
}

// calculateMatchScore calculates how well a keyword matches an entity name
func (sm *SemanticMatcher) calculateMatchScore(keyword, entityName string) MatchScore {
	keyword = strings.ToLower(keyword)
	entityLower := strings.ToLower(entityName)

	// Split entity name into component words
	entityWords := splitEntityName(entityName)

	bestScore := MatchScore{
		Score:      0,
		Keyword:    keyword,
		EntityName: entityName,
	}

	// 1. Exact match (highest score)
	if keyword == entityLower {
		return MatchScore{Score: 1.0, MatchType: "exact", Keyword: keyword, EntityName: entityName}
	}

	// Check against each word in entity name
	for _, word := range entityWords {
		if keyword == word {
			return MatchScore{Score: 0.95, MatchType: "exact_word", Keyword: keyword, EntityName: entityName}
		}
	}

	// 2. Contains match
	if strings.Contains(entityLower, keyword) {
		score := 0.85 * (float64(len(keyword)) / float64(len(entityLower)))
		if score > bestScore.Score {
			bestScore = MatchScore{Score: score + 0.1, MatchType: "contains", Keyword: keyword, EntityName: entityName}
		}
	}
	if strings.Contains(keyword, entityLower) && len(entityLower) >= 3 {
		score := 0.8 * (float64(len(entityLower)) / float64(len(keyword)))
		if score > bestScore.Score {
			bestScore = MatchScore{Score: score + 0.1, MatchType: "contains_reverse", Keyword: keyword, EntityName: entityName}
		}
	}

	// 3. Prefix/suffix match
	if strings.HasPrefix(entityLower, keyword) || strings.HasPrefix(keyword, entityLower) {
		minLen := len(keyword)
		if len(entityLower) < minLen {
			minLen = len(entityLower)
		}
		score := 0.75 * (float64(minLen) / float64(len(entityLower)+len(keyword)-minLen))
		if score > bestScore.Score {
			bestScore = MatchScore{Score: score + 0.15, MatchType: "prefix", Keyword: keyword, EntityName: entityName}
		}
	}

	// 4. Stem matching
	keywordStem := stemWord(keyword)
	entityStem := stemWord(entityLower)
	for _, word := range entityWords {
		wordStem := stemWord(word)
		if keywordStem == wordStem && len(keywordStem) >= 3 {
			return MatchScore{Score: 0.85, MatchType: "stem", Keyword: keyword, EntityName: entityName}
		}
		// Partial stem match
		if strings.HasPrefix(wordStem, keywordStem) || strings.HasPrefix(keywordStem, wordStem) {
			minLen := len(keywordStem)
			if len(wordStem) < minLen {
				minLen = len(wordStem)
			}
			if minLen >= 3 {
				score := 0.7 * (float64(minLen) / float64(len(keywordStem)))
				if score > bestScore.Score {
					bestScore = MatchScore{Score: score, MatchType: "stem_partial", Keyword: keyword, EntityName: entityName}
				}
			}
		}
	}
	if keywordStem == entityStem && len(keywordStem) >= 3 {
		if bestScore.Score < 0.85 {
			bestScore = MatchScore{Score: 0.85, MatchType: "stem", Keyword: keyword, EntityName: entityName}
		}
	}

	// 5. N-gram similarity (trigrams)
	ngramScore := ngramSimilarity(keyword, entityLower, 3)
	if ngramScore > 0.3 && ngramScore > bestScore.Score {
		bestScore = MatchScore{Score: ngramScore * 0.8, MatchType: "ngram", Keyword: keyword, EntityName: entityName}
	}

	// 6. Levenshtein distance (fuzzy match) - only for similar length strings
	lenDiff := len(keyword) - len(entityLower)
	if lenDiff < 0 {
		lenDiff = -lenDiff
	}
	if lenDiff <= 3 && len(keyword) >= 4 {
		distance := levenshteinDistance(keyword, entityLower)
		maxLen := len(keyword)
		if len(entityLower) > maxLen {
			maxLen = len(entityLower)
		}
		if distance <= 2 {
			score := 1.0 - (float64(distance) / float64(maxLen))
			score *= 0.7 // Fuzzy matches get lower weight
			if score > bestScore.Score {
				bestScore = MatchScore{Score: score, MatchType: "fuzzy", Keyword: keyword, EntityName: entityName}
			}
		}
	}

	return bestScore
}

// FindMatches finds all tables/columns that match the given keywords
func (e *QueryEngine) findSemanticMatches(keywords []string) []struct {
	Table     config.TableInfo
	Score     float64
	MatchType string
	MatchedOn string
	Keyword   string
} {
	matcher := &SemanticMatcher{}
	var matches []struct {
		Table     config.TableInfo
		Score     float64
		MatchType string
		MatchedOn string
		Keyword   string
	}

	// Minimum score threshold for a match
	const minScore = 0.35

	for _, table := range e.schema.Tables {
		var bestTableScore float64
		var bestMatch struct {
			Score     float64
			MatchType string
			MatchedOn string
			Keyword   string
		}

		for _, keyword := range keywords {
			// Check table name
			score := matcher.calculateMatchScore(keyword, table.Name)
			if score.Score > bestTableScore && score.Score >= minScore {
				bestTableScore = score.Score
				bestMatch = struct {
					Score     float64
					MatchType string
					MatchedOn string
					Keyword   string
				}{score.Score, score.MatchType + "_table", table.Name, keyword}
			}

			// Check table comment
			if table.Comment != "" {
				commentWords := strings.Fields(strings.ToLower(table.Comment))
				for _, word := range commentWords {
					score := matcher.calculateMatchScore(keyword, word)
					if score.Score > bestTableScore && score.Score >= minScore {
						bestTableScore = score.Score
						bestMatch = struct {
							Score     float64
							MatchType string
							MatchedOn string
							Keyword   string
						}{score.Score * 0.9, score.MatchType + "_comment", table.Comment, keyword}
					}
				}
			}

			// Check column names
			for _, col := range table.Columns {
				score := matcher.calculateMatchScore(keyword, col.Name)
				if score.Score >= minScore {
					// Column matches boost the table score
					colScore := score.Score * 0.85 // Slightly lower weight for column matches
					if colScore > bestTableScore {
						bestTableScore = colScore
						bestMatch = struct {
							Score     float64
							MatchType string
							MatchedOn string
							Keyword   string
						}{colScore, score.MatchType + "_column", col.Name, keyword}
					}
				}
			}
		}

		if bestTableScore >= minScore {
			matches = append(matches, struct {
				Table     config.TableInfo
				Score     float64
				MatchType string
				MatchedOn string
				Keyword   string
			}{table, bestMatch.Score, bestMatch.MatchType, bestMatch.MatchedOn, bestMatch.Keyword})
		}
	}

	// Sort by score descending
	for i := 0; i < len(matches)-1; i++ {
		for j := i + 1; j < len(matches); j++ {
			if matches[j].Score > matches[i].Score {
				matches[i], matches[j] = matches[j], matches[i]
			}
		}
	}

	return matches
}

func (e *QueryEngine) matchSearchQuery(query string) string {
	// Check for search/find/have patterns with keywords
	searchPatterns := []string{
		`(?:do i have|is there|are there|find|search for|look for|any) (?:any )?(.+?)(?:\s+(?:related\s+)?data|\s+tables?|\s+information)?$`,
		`(?:what|which) (?:tables?|data) (?:contain|have|include|relate to|about) (.+)`,
		`(.+?)(?:\s+related)?\s+(?:tables?|data)`,
	}

	// Common words to exclude from keyword extraction
	excludeWords := map[string]bool{
		"the": true, "a": true, "an": true, "any": true, "some": true,
		"data": true, "table": true, "tables": true, "related": true,
		"information": true, "do": true, "i": true, "have": true, "is": true,
		"there": true, "are": true, "find": true, "search": true, "for": true,
		"look": true, "what": true, "which": true, "contain": true, "about": true,
		"include": true, "with": true, "my": true, "in": true, "to": true,
	}

	// Extract keywords from query
	var keywords []string
	for _, pattern := range searchPatterns {
		re := regexp.MustCompile(pattern)
		if matches := re.FindStringSubmatch(query); len(matches) > 1 {
			// Split the matched group into words
			words := strings.Fields(matches[1])
			for _, word := range words {
				word = strings.ToLower(strings.Trim(word, ".,?!"))
				if len(word) > 2 && !excludeWords[word] {
					keywords = append(keywords, word)
				}
			}
			break
		}
	}

	// If no keywords from patterns, try to extract meaningful words from whole query
	if len(keywords) == 0 {
		words := strings.Fields(query)
		for _, word := range words {
			word = strings.ToLower(strings.Trim(word, ".,?!"))
			if len(word) > 3 && !excludeWords[word] {
				keywords = append(keywords, word)
			}
		}
	}

	if len(keywords) == 0 {
		return ""
	}

	// Use semantic matching to find related tables/columns
	matches := e.findSemanticMatches(keywords)

	if len(matches) == 0 {
		// Return a query showing what's available
		return `SELECT table_schema, table_name,
			(SELECT string_agg(column_name, ', ') FROM information_schema.columns c
			 WHERE c.table_schema = t.table_schema AND c.table_name = t.table_name) as columns
			FROM information_schema.tables t
			WHERE table_type = 'BASE TABLE' AND table_schema NOT IN ('pg_catalog', 'information_schema')
			ORDER BY table_schema, table_name`
	}

	// If only one table matches with high confidence, show its data
	if len(matches) == 1 {
		t := matches[0].Table
		return fmt.Sprintf("SELECT * FROM \"%s\".\"%s\" LIMIT 50", t.Schema, t.Name)
	}

	// Check if top matches have similar scores (within 0.2 of each other)
	// If so, combine their results instead of just showing counts
	var similarMatches []struct {
		Table     config.TableInfo
		Score     float64
		MatchType string
		MatchedOn string
		Keyword   string
	}

	topScore := matches[0].Score
	for _, m := range matches {
		// Include matches within 0.25 of top score, up to 5 tables
		if topScore-m.Score <= 0.25 && len(similarMatches) < 5 {
			similarMatches = append(similarMatches, m)
		}
	}

	// If we have multiple similar matches, combine their data with UNION ALL
	if len(similarMatches) > 1 {
		// Find common columns across all matching tables
		commonCols := e.findCommonColumns(similarMatches)

		if len(commonCols) > 0 {
			// Build UNION query with common columns + source table identifier
			var unionQueries []string
			for _, m := range similarMatches {
				colList := make([]string, len(commonCols))
				for i, col := range commonCols {
					colList[i] = fmt.Sprintf("\"%s\"", col)
				}
				unionQueries = append(unionQueries, fmt.Sprintf(
					"SELECT '%s.%s' as _source_table, %s FROM \"%s\".\"%s\"",
					m.Table.Schema, m.Table.Name,
					strings.Join(colList, ", "),
					m.Table.Schema, m.Table.Name,
				))
			}
			return strings.Join(unionQueries, " UNION ALL ") + " LIMIT 100"
		}

		// No common columns - just select * from each with source identifier
		var unionQueries []string
		rowsPerTable := 100 / len(similarMatches)
		if rowsPerTable < 10 {
			rowsPerTable = 10
		}
		for _, m := range similarMatches {
			unionQueries = append(unionQueries, fmt.Sprintf(
				"(SELECT '%s.%s' as _source_table, * FROM \"%s\".\"%s\" LIMIT %d)",
				m.Table.Schema, m.Table.Name,
				m.Table.Schema, m.Table.Name,
				rowsPerTable,
			))
		}
		return strings.Join(unionQueries, " UNION ALL ")
	}

	// Single high-confidence match
	if matches[0].Score > 0.6 {
		t := matches[0].Table
		return fmt.Sprintf("SELECT * FROM \"%s\".\"%s\" LIMIT 50", t.Schema, t.Name)
	}

	// Lower confidence matches - show a summary of what was found
	var queries []string
	for i, m := range matches {
		if i >= 10 { // Limit to 10 tables
			break
		}
		queries = append(queries, fmt.Sprintf(
			"SELECT '%s.%s' as table_name, '%.0f%%' as match_score, '%s' as matched_on, COUNT(*) as row_count FROM \"%s\".\"%s\"",
			m.Table.Schema, m.Table.Name, m.Score*100, m.MatchedOn, m.Table.Schema, m.Table.Name,
		))
	}

	if len(queries) > 0 {
		return strings.Join(queries, " UNION ALL ") + " ORDER BY row_count DESC"
	}

	return ""
}

// findCommonColumns finds columns that exist in all the given tables
func (e *QueryEngine) findCommonColumns(matches []struct {
	Table     config.TableInfo
	Score     float64
	MatchType string
	MatchedOn string
	Keyword   string
}) []string {
	if len(matches) == 0 {
		return nil
	}

	// Start with columns from first table
	commonCols := make(map[string]int)
	for _, col := range matches[0].Table.Columns {
		commonCols[col.Name] = 1
	}

	// Keep only columns that exist in all tables
	for _, m := range matches[1:] {
		tableCols := make(map[string]bool)
		for _, col := range m.Table.Columns {
			tableCols[col.Name] = true
		}
		for colName := range commonCols {
			if tableCols[colName] {
				commonCols[colName]++
			}
		}
	}

	// Filter to columns present in ALL tables
	var result []string
	targetCount := len(matches)
	for colName, count := range commonCols {
		if count == targetCount {
			result = append(result, colName)
		}
	}

	// Sort for consistent output
	for i := 0; i < len(result)-1; i++ {
		for j := i + 1; j < len(result); j++ {
			if result[i] > result[j] {
				result[i], result[j] = result[j], result[i]
			}
		}
	}

	return result
}

func (e *QueryEngine) findTable(name string) *config.TableInfo {
	name = strings.ToLower(name)

	// Exact match
	for _, table := range e.schema.Tables {
		if strings.ToLower(table.Name) == name {
			return &table
		}
	}

	// Partial match
	for _, table := range e.schema.Tables {
		if strings.Contains(strings.ToLower(table.Name), name) {
			return &table
		}
	}

	// Singular/plural handling
	singular := strings.TrimSuffix(name, "s")
	for _, table := range e.schema.Tables {
		tableLower := strings.ToLower(table.Name)
		if tableLower == singular || strings.TrimSuffix(tableLower, "s") == singular {
			return &table
		}
	}

	return nil
}

// SetSchema updates the schema for the query engine
func (e *QueryEngine) SetSchema(schema *config.SchemaCache) {
	e.schema = schema
}

// TrainFromHistory trains the neural network from query history
func (e *QueryEngine) TrainFromHistory(history []config.QueryHistoryEntry, epochs int) error {
	if e.nnTrainer == nil {
		return fmt.Errorf("neural network trainer not initialized")
	}
	return e.nnTrainer.Train(history, epochs)
}

// TrainFromHistoryAsync trains the neural network asynchronously
func (e *QueryEngine) TrainFromHistoryAsync(history []config.QueryHistoryEntry, epochs int, callback func(error)) {
	if e.nnTrainer == nil {
		if callback != nil {
			callback(fmt.Errorf("neural network trainer not initialized"))
		}
		return
	}
	e.nnTrainer.TrainAsync(history, epochs, callback)
}

// IsNNTrained returns whether the neural network has been trained
func (e *QueryEngine) IsNNTrained() bool {
	if e.nnTrainer == nil {
		return false
	}
	return e.nnTrainer.IsTrained()
}

// SetUseNN enables or disables neural network predictions
func (e *QueryEngine) SetUseNN(use bool) {
	e.useNN = use
}

// GetNNStats returns statistics about the neural network
func (e *QueryEngine) GetNNStats() map[string]interface{} {
	if e.nnTrainer == nil {
		return map[string]interface{}{"available": false}
	}
	stats := e.nnTrainer.GetTrainingStats()
	stats["available"] = true
	stats["enabled"] = e.useNN
	return stats
}

// isValidSQLStructure checks if a string looks like valid SQL
func isValidSQLStructure(sql string) bool {
	sql = strings.TrimSpace(strings.ToLower(sql))
	if sql == "" {
		return false
	}

	// Must start with a SQL keyword
	validStarts := []string{"select", "insert", "update", "delete", "with", "create", "alter", "drop"}
	hasValidStart := false
	for _, start := range validStarts {
		if strings.HasPrefix(sql, start) {
			hasValidStart = true
			break
		}
	}
	if !hasValidStart {
		return false
	}

	// For SELECT, must have FROM
	if strings.HasPrefix(sql, "select") && !strings.Contains(sql, "from") {
		return false
	}

	// Basic balanced parentheses check
	openCount := strings.Count(sql, "(")
	closeCount := strings.Count(sql, ")")
	if openCount != closeCount {
		return false
	}

	return true
}

// GetSchemaContext returns the schema description for LLM context
func (e *QueryEngine) GetSchemaContext() string {
	if e.schema == nil {
		return ""
	}
	return generateSchemaDescription(e.schema)
}

func generateSchemaDescription(cache *config.SchemaCache) string {
	var desc strings.Builder

	desc.WriteString("DATABASE SCHEMA:\n")
	desc.WriteString("================\n\n")

	if cache.HasPostGIS {
		desc.WriteString("PostGIS is installed - spatial queries are supported.\n\n")
	}

	desc.WriteString("TABLES:\n")
	for _, t := range cache.Tables {
		desc.WriteString(fmt.Sprintf("- %s.%s", t.Schema, t.Name))
		if t.Comment != "" {
			desc.WriteString(fmt.Sprintf(" (%s)", t.Comment))
		}
		desc.WriteString("\n")
		for _, c := range t.Columns {
			desc.WriteString(fmt.Sprintf("    - %s (%s)", c.Name, c.DataType))
			if c.IsPrimaryKey {
				desc.WriteString(" [PK]")
			}
			if c.IsForeignKey {
				desc.WriteString(fmt.Sprintf(" [FK -> %s.%s]", c.FKTable, c.FKColumn))
			}
			if c.IsGeometry {
				desc.WriteString(fmt.Sprintf(" [GEOMETRY: %s]", c.GeomType))
			}
			desc.WriteString("\n")
		}
		desc.WriteString("\n")
	}

	return desc.String()
}

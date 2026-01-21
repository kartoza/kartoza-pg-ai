package nn

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"

	"github.com/kartoza/kartoza-pg-ai/internal/config"
)

// QueryTrainer manages training the neural network from query history
type QueryTrainer struct {
	model     *Seq2SeqModel
	tokenizer *Tokenizer
	modelPath string
	tokPath   string
	mu        sync.RWMutex
	minPairs  int // Minimum query-SQL pairs required for training
}

// NewQueryTrainer creates a new query trainer
func NewQueryTrainer() (*QueryTrainer, error) {
	configDir, err := config.ConfigDir()
	if err != nil {
		return nil, err
	}

	modelDir := filepath.Join(configDir, "nn_model")
	if err := os.MkdirAll(modelDir, 0755); err != nil {
		return nil, err
	}

	modelPath := filepath.Join(modelDir, "model.gob")
	tokPath := filepath.Join(modelDir, "tokenizer.json")

	cfg := DefaultModelConfig()
	model := NewSeq2SeqModel(cfg)
	tokenizer := NewTokenizer(cfg.MaxSeqLen)

	trainer := &QueryTrainer{
		model:     model,
		tokenizer: tokenizer,
		modelPath: modelPath,
		tokPath:   tokPath,
		minPairs:  10, // Require at least 10 pairs before training
	}

	// Try to load existing model and tokenizer
	trainer.loadModelIfExists()

	return trainer, nil
}

// loadModelIfExists attempts to load an existing model
func (t *QueryTrainer) loadModelIfExists() {
	// Try to load tokenizer
	if tokData, err := os.ReadFile(t.tokPath); err == nil {
		var tokExport map[string]interface{}
		if err := json.Unmarshal(tokData, &tokExport); err == nil {
			t.tokenizer.Import(tokExport)
		}
	}

	// Try to load model
	if _, err := os.Stat(t.modelPath); err == nil {
		t.model.Load(t.modelPath)
	}
}

// ExtractTrainingData extracts training pairs from query history
func (t *QueryTrainer) ExtractTrainingData(history []config.QueryHistoryEntry) ([]TrainingPair, error) {
	var pairs []TrainingPair

	for _, entry := range history {
		// Only use successful queries with both NL and SQL
		if !entry.Success || entry.NaturalQuery == "" || entry.GeneratedSQL == "" {
			continue
		}

		// Skip very short queries (likely not useful)
		if len(entry.NaturalQuery) < 10 || len(entry.GeneratedSQL) < 10 {
			continue
		}

		pairs = append(pairs, TrainingPair{
			NaturalLanguage: entry.NaturalQuery,
			SQL:             entry.GeneratedSQL,
		})
	}

	return pairs, nil
}

// TrainingPair represents a NL-SQL pair for training
type TrainingPair struct {
	NaturalLanguage string
	SQL             string
}

// Train trains the model on query history
func (t *QueryTrainer) Train(history []config.QueryHistoryEntry, epochs int) error {
	t.mu.Lock()
	defer t.mu.Unlock()

	// Extract training data
	pairs, err := t.ExtractTrainingData(history)
	if err != nil {
		return fmt.Errorf("failed to extract training data: %w", err)
	}

	if len(pairs) < t.minPairs {
		return fmt.Errorf("insufficient training data: have %d pairs, need at least %d", len(pairs), t.minPairs)
	}

	// Build vocabulary from all texts
	var allTexts []string
	for _, pair := range pairs {
		allTexts = append(allTexts, pair.NaturalLanguage)
		allTexts = append(allTexts, pair.SQL)
	}
	t.tokenizer.BuildVocabulary(allTexts)

	// Update model vocab size
	t.model.vocabSize = t.tokenizer.VocabSize()

	// Encode training data
	var inputSeqs, targetSeqs [][]int
	for _, pair := range pairs {
		inputSeq := t.tokenizer.Encode(pair.NaturalLanguage)
		targetSeq := t.tokenizer.Encode(pair.SQL)
		inputSeqs = append(inputSeqs, inputSeq)
		targetSeqs = append(targetSeqs, targetSeq)
	}

	// Train the model
	fmt.Printf("Training on %d query-SQL pairs for %d epochs...\n", len(pairs), epochs)
	if err := t.model.Train(inputSeqs, targetSeqs, epochs); err != nil {
		return fmt.Errorf("training failed: %w", err)
	}

	// Save model and tokenizer
	if err := t.Save(); err != nil {
		return fmt.Errorf("failed to save model: %w", err)
	}

	return nil
}

// TrainAsync trains the model asynchronously
func (t *QueryTrainer) TrainAsync(history []config.QueryHistoryEntry, epochs int, callback func(error)) {
	go func() {
		err := t.Train(history, epochs)
		if callback != nil {
			callback(err)
		}
	}()
}

// Predict generates SQL from natural language using the trained model
func (t *QueryTrainer) Predict(query string) (string, float64, error) {
	t.mu.RLock()
	defer t.mu.RUnlock()

	if !t.model.IsTrained() {
		return "", 0, fmt.Errorf("model not trained")
	}

	// Encode input
	inputSeq := t.tokenizer.Encode(query)

	// Get prediction
	outputSeq, err := t.model.Predict(inputSeq)
	if err != nil {
		return "", 0, err
	}

	// Decode output
	sql := t.tokenizer.Decode(outputSeq)

	// Return with a confidence score (simplified - just return 0.5 for now)
	confidence := 0.5
	if len(outputSeq) > 0 {
		confidence = 0.7
	}

	return sql, confidence, nil
}

// IsTrained returns whether the model is trained
func (t *QueryTrainer) IsTrained() bool {
	t.mu.RLock()
	defer t.mu.RUnlock()
	return t.model.IsTrained()
}

// GetTrainingStats returns statistics about the model
func (t *QueryTrainer) GetTrainingStats() map[string]interface{} {
	t.mu.RLock()
	defer t.mu.RUnlock()

	return map[string]interface{}{
		"trained":        t.model.IsTrained(),
		"vocab_size":     t.tokenizer.VocabSize(),
		"max_seq_len":    t.tokenizer.MaxSeqLen(),
		"embedding_dim":  t.model.embeddingDim,
		"hidden_dim":     t.model.hiddenDim,
		"min_pairs":      t.minPairs,
	}
}

// Save saves the model and tokenizer to disk
func (t *QueryTrainer) Save() error {
	// Save tokenizer
	tokExport := t.tokenizer.Export()
	tokData, err := json.MarshalIndent(tokExport, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal tokenizer: %w", err)
	}
	if err := os.WriteFile(t.tokPath, tokData, 0644); err != nil {
		return fmt.Errorf("failed to write tokenizer: %w", err)
	}

	// Save model
	if err := t.model.Save(t.modelPath); err != nil {
		return fmt.Errorf("failed to save model: %w", err)
	}

	return nil
}

// SetMinPairs sets the minimum number of pairs required for training
func (t *QueryTrainer) SetMinPairs(min int) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.minPairs = min
}

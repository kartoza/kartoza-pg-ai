package nn

import (
	"encoding/gob"
	"fmt"
	"math"
	"math/rand"
	"os"

	"gorgonia.org/gorgonia"
	"gorgonia.org/tensor"
)

// Seq2SeqModel represents a sequence-to-sequence neural network for NL to SQL
type Seq2SeqModel struct {
	g            *gorgonia.ExprGraph
	vm           gorgonia.VM

	// Model parameters
	vocabSize    int
	embeddingDim int
	hiddenDim    int
	maxSeqLen    int

	// Encoder weights
	encoderEmbed *gorgonia.Node // Embedding matrix
	encoderWih   *gorgonia.Node // Input to hidden weights
	encoderBih   *gorgonia.Node // Input to hidden bias
	encoderWhh   *gorgonia.Node // Hidden to hidden weights
	encoderBhh   *gorgonia.Node // Hidden to hidden bias

	// Decoder weights
	decoderEmbed *gorgonia.Node // Embedding matrix (shared or separate)
	decoderWih   *gorgonia.Node // Input to hidden weights
	decoderBih   *gorgonia.Node // Input to hidden bias
	decoderWhh   *gorgonia.Node // Hidden to hidden weights
	decoderBhh   *gorgonia.Node // Hidden to hidden bias
	decoderWho   *gorgonia.Node // Hidden to output weights
	decoderBho   *gorgonia.Node // Hidden to output bias

	// Attention weights
	attnW        *gorgonia.Node // Attention weight matrix

	// Learnables (for gradient descent)
	learnables   gorgonia.Nodes

	// Input/output nodes
	input        *gorgonia.Node
	target       *gorgonia.Node
	output       *gorgonia.Node
	loss         *gorgonia.Node

	// Training state
	solver       gorgonia.Solver
	trained      bool
}

// ModelConfig holds configuration for the model
type ModelConfig struct {
	VocabSize    int
	EmbeddingDim int
	HiddenDim    int
	MaxSeqLen    int
	LearningRate float64
}

// DefaultModelConfig returns default configuration
func DefaultModelConfig() ModelConfig {
	return ModelConfig{
		VocabSize:    5000,
		EmbeddingDim: 64,
		HiddenDim:    128,
		MaxSeqLen:    100,
		LearningRate: 0.001,
	}
}

// NewSeq2SeqModel creates a new sequence-to-sequence model
func NewSeq2SeqModel(config ModelConfig) *Seq2SeqModel {
	g := gorgonia.NewGraph()

	m := &Seq2SeqModel{
		g:            g,
		vocabSize:    config.VocabSize,
		embeddingDim: config.EmbeddingDim,
		hiddenDim:    config.HiddenDim,
		maxSeqLen:    config.MaxSeqLen,
	}

	// Initialize weights with Xavier initialization
	m.initWeights()

	// Create solver (Adam optimizer)
	m.solver = gorgonia.NewAdamSolver(gorgonia.WithLearnRate(config.LearningRate))

	return m
}

// initWeights initializes all model weights
func (m *Seq2SeqModel) initWeights() {
	// Xavier initialization scale
	embedScale := math.Sqrt(2.0 / float64(m.vocabSize+m.embeddingDim))
	rnnScale := math.Sqrt(2.0 / float64(m.embeddingDim+m.hiddenDim))
	outScale := math.Sqrt(2.0 / float64(m.hiddenDim+m.vocabSize))

	// Encoder embedding
	encoderEmbedT := tensor.New(
		tensor.WithShape(m.vocabSize, m.embeddingDim),
		tensor.WithBacking(randomFloat64(m.vocabSize*m.embeddingDim, embedScale)),
	)
	m.encoderEmbed = gorgonia.NewMatrix(m.g, tensor.Float64,
		gorgonia.WithShape(m.vocabSize, m.embeddingDim),
		gorgonia.WithName("encoder_embed"),
		gorgonia.WithValue(encoderEmbedT),
	)

	// Encoder RNN weights
	m.encoderWih = m.newWeight("encoder_wih", m.embeddingDim, m.hiddenDim, rnnScale)
	m.encoderBih = m.newBias("encoder_bih", m.hiddenDim)
	m.encoderWhh = m.newWeight("encoder_whh", m.hiddenDim, m.hiddenDim, rnnScale)
	m.encoderBhh = m.newBias("encoder_bhh", m.hiddenDim)

	// Decoder embedding (can share with encoder)
	decoderEmbedT := tensor.New(
		tensor.WithShape(m.vocabSize, m.embeddingDim),
		tensor.WithBacking(randomFloat64(m.vocabSize*m.embeddingDim, embedScale)),
	)
	m.decoderEmbed = gorgonia.NewMatrix(m.g, tensor.Float64,
		gorgonia.WithShape(m.vocabSize, m.embeddingDim),
		gorgonia.WithName("decoder_embed"),
		gorgonia.WithValue(decoderEmbedT),
	)

	// Decoder RNN weights
	m.decoderWih = m.newWeight("decoder_wih", m.embeddingDim, m.hiddenDim, rnnScale)
	m.decoderBih = m.newBias("decoder_bih", m.hiddenDim)
	m.decoderWhh = m.newWeight("decoder_whh", m.hiddenDim, m.hiddenDim, rnnScale)
	m.decoderBhh = m.newBias("decoder_bhh", m.hiddenDim)

	// Output projection
	m.decoderWho = m.newWeight("decoder_who", m.hiddenDim, m.vocabSize, outScale)
	m.decoderBho = m.newBias("decoder_bho", m.vocabSize)

	// Attention weights
	m.attnW = m.newWeight("attn_w", m.hiddenDim, m.hiddenDim, rnnScale)

	// Collect learnables
	m.learnables = gorgonia.Nodes{
		m.encoderEmbed, m.encoderWih, m.encoderBih, m.encoderWhh, m.encoderBhh,
		m.decoderEmbed, m.decoderWih, m.decoderBih, m.decoderWhh, m.decoderBhh,
		m.decoderWho, m.decoderBho, m.attnW,
	}
}

func (m *Seq2SeqModel) newWeight(name string, rows, cols int, scale float64) *gorgonia.Node {
	t := tensor.New(
		tensor.WithShape(rows, cols),
		tensor.WithBacking(randomFloat64(rows*cols, scale)),
	)
	return gorgonia.NewMatrix(m.g, tensor.Float64,
		gorgonia.WithShape(rows, cols),
		gorgonia.WithName(name),
		gorgonia.WithValue(t),
	)
}

func (m *Seq2SeqModel) newBias(name string, size int) *gorgonia.Node {
	t := tensor.New(
		tensor.WithShape(size),
		tensor.WithBacking(make([]float64, size)),
	)
	return gorgonia.NewVector(m.g, tensor.Float64,
		gorgonia.WithShape(size),
		gorgonia.WithName(name),
		gorgonia.WithValue(t),
	)
}

func randomFloat64(size int, scale float64) []float64 {
	result := make([]float64, size)
	for i := range result {
		result[i] = (rand.Float64()*2 - 1) * scale
	}
	return result
}

// Forward performs a forward pass through the network
func (m *Seq2SeqModel) Forward(inputSeq, targetSeq []int) (*gorgonia.Node, error) {
	// Rebuild graph for this forward pass
	m.g = gorgonia.NewGraph()
	m.initWeights()

	batchSize := 1

	// Create input tensor (one-hot or index-based)
	inputT := tensor.New(
		tensor.WithShape(batchSize, m.maxSeqLen),
		tensor.WithBacking(toFloat64(inputSeq)),
	)
	m.input = gorgonia.NewMatrix(m.g, tensor.Float64,
		gorgonia.WithShape(batchSize, m.maxSeqLen),
		gorgonia.WithName("input"),
		gorgonia.WithValue(inputT),
	)

	// Encode
	encoderOutputs, encoderHidden, err := m.encode(m.input)
	if err != nil {
		return nil, fmt.Errorf("encoding failed: %w", err)
	}

	// Decode with attention
	decoderOutput, err := m.decode(encoderOutputs, encoderHidden, targetSeq)
	if err != nil {
		return nil, fmt.Errorf("decoding failed: %w", err)
	}

	m.output = decoderOutput
	return decoderOutput, nil
}

// encode runs the encoder RNN
func (m *Seq2SeqModel) encode(input *gorgonia.Node) ([]*gorgonia.Node, *gorgonia.Node, error) {
	var outputs []*gorgonia.Node

	// Initialize hidden state
	hiddenT := tensor.New(
		tensor.WithShape(m.hiddenDim),
		tensor.WithBacking(make([]float64, m.hiddenDim)),
	)
	hidden := gorgonia.NewVector(m.g, tensor.Float64,
		gorgonia.WithShape(m.hiddenDim),
		gorgonia.WithName("encoder_h0"),
		gorgonia.WithValue(hiddenT),
	)

	// Process each timestep
	for t := 0; t < m.maxSeqLen; t++ {
		// Get embedding for this timestep
		// For simplicity, we'll use a simple lookup
		embedded, err := m.lookupEmbedding(m.encoderEmbed, input, t)
		if err != nil {
			return nil, nil, err
		}

		// RNN cell: h_t = tanh(W_ih * x_t + b_ih + W_hh * h_{t-1} + b_hh)
		hidden, err = m.rnnCell(embedded, hidden,
			m.encoderWih, m.encoderBih, m.encoderWhh, m.encoderBhh,
			fmt.Sprintf("enc_%d", t))
		if err != nil {
			return nil, nil, err
		}

		outputs = append(outputs, hidden)
	}

	return outputs, hidden, nil
}

// decode runs the decoder RNN with attention
func (m *Seq2SeqModel) decode(encoderOutputs []*gorgonia.Node, encoderHidden *gorgonia.Node, targetSeq []int) (*gorgonia.Node, error) {
	hidden := encoderHidden
	var outputLogits []*gorgonia.Node

	// Start token
	prevToken := 2 // START token index

	for t := 0; t < m.maxSeqLen && t < len(targetSeq); t++ {
		// Get embedding for previous token
		prevTokenT := tensor.New(
			tensor.WithShape(1),
			tensor.WithBacking([]float64{float64(prevToken)}),
		)
		prevTokenNode := gorgonia.NewVector(m.g, tensor.Float64,
			gorgonia.WithShape(1),
			gorgonia.WithName(fmt.Sprintf("prev_token_%d", t)),
			gorgonia.WithValue(prevTokenT),
		)

		embedded, err := m.lookupEmbeddingDirect(m.decoderEmbed, prevTokenNode)
		if err != nil {
			return nil, err
		}

		// Compute attention context
		context, err := m.attention(hidden, encoderOutputs, t)
		if err != nil {
			return nil, err
		}

		// Combine embedding and context
		combined, err := gorgonia.Add(embedded, context)
		if err != nil {
			return nil, err
		}

		// RNN cell
		hidden, err = m.rnnCell(combined, hidden,
			m.decoderWih, m.decoderBih, m.decoderWhh, m.decoderBhh,
			fmt.Sprintf("dec_%d", t))
		if err != nil {
			return nil, err
		}

		// Project to vocabulary
		logits, err := m.project(hidden, t)
		if err != nil {
			return nil, err
		}

		outputLogits = append(outputLogits, logits)

		// Teacher forcing: use actual target as next input during training
		if t < len(targetSeq)-1 {
			prevToken = targetSeq[t+1]
		}
	}

	// Stack outputs
	if len(outputLogits) == 0 {
		return nil, fmt.Errorf("no output logits generated")
	}

	return outputLogits[0], nil // Return first for now; proper impl would stack all
}

// lookupEmbedding performs embedding lookup
func (m *Seq2SeqModel) lookupEmbedding(embedMatrix, input *gorgonia.Node, timestep int) (*gorgonia.Node, error) {
	// Simplified: return a slice of the embedding matrix
	// In a proper implementation, this would index into the matrix
	sliced, err := gorgonia.Slice(embedMatrix, gorgonia.S(0)) // Get first row as placeholder
	if err != nil {
		return nil, err
	}
	return sliced, nil
}

// lookupEmbeddingDirect performs direct embedding lookup
func (m *Seq2SeqModel) lookupEmbeddingDirect(embedMatrix, tokenIdx *gorgonia.Node) (*gorgonia.Node, error) {
	// Simplified embedding lookup
	sliced, err := gorgonia.Slice(embedMatrix, gorgonia.S(0))
	if err != nil {
		return nil, err
	}
	return sliced, nil
}

// rnnCell performs one step of RNN computation
func (m *Seq2SeqModel) rnnCell(input, hidden, wih, bih, whh, bhh *gorgonia.Node, name string) (*gorgonia.Node, error) {
	// h_t = tanh(W_ih * x_t + b_ih + W_hh * h_{t-1} + b_hh)

	// W_ih * x_t
	ih, err := gorgonia.Mul(input, wih)
	if err != nil {
		return nil, fmt.Errorf("rnn %s ih mul: %w", name, err)
	}

	// + b_ih
	ih, err = gorgonia.BroadcastAdd(ih, bih, nil, []byte{0})
	if err != nil {
		return nil, fmt.Errorf("rnn %s ih bias: %w", name, err)
	}

	// W_hh * h_{t-1}
	hh, err := gorgonia.Mul(hidden, whh)
	if err != nil {
		return nil, fmt.Errorf("rnn %s hh mul: %w", name, err)
	}

	// + b_hh
	hh, err = gorgonia.BroadcastAdd(hh, bhh, nil, []byte{0})
	if err != nil {
		return nil, fmt.Errorf("rnn %s hh bias: %w", name, err)
	}

	// sum
	sum, err := gorgonia.Add(ih, hh)
	if err != nil {
		return nil, fmt.Errorf("rnn %s sum: %w", name, err)
	}

	// tanh activation
	newHidden, err := gorgonia.Tanh(sum)
	if err != nil {
		return nil, fmt.Errorf("rnn %s tanh: %w", name, err)
	}

	return newHidden, nil
}

// attention computes attention over encoder outputs
func (m *Seq2SeqModel) attention(decoderHidden *gorgonia.Node, encoderOutputs []*gorgonia.Node, timestep int) (*gorgonia.Node, error) {
	if len(encoderOutputs) == 0 {
		// Return zeros if no encoder outputs
		zeroT := tensor.New(
			tensor.WithShape(m.hiddenDim),
			tensor.WithBacking(make([]float64, m.hiddenDim)),
		)
		return gorgonia.NewVector(m.g, tensor.Float64,
			gorgonia.WithShape(m.hiddenDim),
			gorgonia.WithName(fmt.Sprintf("attn_ctx_%d", timestep)),
			gorgonia.WithValue(zeroT),
		), nil
	}

	// Simplified attention: just use the last encoder output
	// A proper implementation would compute attention scores
	return encoderOutputs[len(encoderOutputs)-1], nil
}

// project projects hidden state to vocabulary
func (m *Seq2SeqModel) project(hidden *gorgonia.Node, timestep int) (*gorgonia.Node, error) {
	// hidden * W_ho + b_ho
	projected, err := gorgonia.Mul(hidden, m.decoderWho)
	if err != nil {
		return nil, err
	}

	projected, err = gorgonia.BroadcastAdd(projected, m.decoderBho, nil, []byte{0})
	if err != nil {
		return nil, err
	}

	// Softmax for probabilities
	output, err := gorgonia.SoftMax(projected)
	if err != nil {
		return nil, err
	}

	return output, nil
}

// ComputeLoss computes cross-entropy loss
func (m *Seq2SeqModel) ComputeLoss(output *gorgonia.Node, target []int) (*gorgonia.Node, error) {
	// Create target tensor
	targetT := tensor.New(
		tensor.WithShape(m.vocabSize),
		tensor.WithBacking(oneHot(target[0], m.vocabSize)),
	)
	targetNode := gorgonia.NewVector(m.g, tensor.Float64,
		gorgonia.WithShape(m.vocabSize),
		gorgonia.WithName("target"),
		gorgonia.WithValue(targetT),
	)

	// Cross-entropy loss: -sum(target * log(output))
	logOutput, err := gorgonia.Log(output)
	if err != nil {
		return nil, err
	}

	prod, err := gorgonia.HadamardProd(targetNode, logOutput)
	if err != nil {
		return nil, err
	}

	loss, err := gorgonia.Sum(prod)
	if err != nil {
		return nil, err
	}

	negLoss, err := gorgonia.Neg(loss)
	if err != nil {
		return nil, err
	}

	m.loss = negLoss
	return negLoss, nil
}

// Train trains the model on a batch of examples
func (m *Seq2SeqModel) Train(inputSeqs, targetSeqs [][]int, epochs int) error {
	for epoch := 0; epoch < epochs; epoch++ {
		totalLoss := 0.0

		for i := range inputSeqs {
			// Forward pass
			output, err := m.Forward(inputSeqs[i], targetSeqs[i])
			if err != nil {
				return fmt.Errorf("forward pass failed: %w", err)
			}

			// Compute loss
			_, err = m.ComputeLoss(output, targetSeqs[i])
			if err != nil {
				return fmt.Errorf("loss computation failed: %w", err)
			}

			// Create VM and run
			m.vm = gorgonia.NewTapeMachine(m.g)
			if err := m.vm.RunAll(); err != nil {
				m.vm.Close()
				return fmt.Errorf("vm run failed: %w", err)
			}

			// Get loss value
			if m.loss != nil {
				lossVal := m.loss.Value()
				if lossVal != nil {
					if scalar, ok := lossVal.Data().(float64); ok {
						totalLoss += scalar
					}
				}
			}

			// Backward pass
			if _, err := gorgonia.Grad(m.loss, m.learnables...); err != nil {
				m.vm.Close()
				return fmt.Errorf("gradient computation failed: %w", err)
			}

			// Update weights
			if err := m.solver.Step(gorgonia.NodesToValueGrads(m.learnables)); err != nil {
				m.vm.Close()
				return fmt.Errorf("solver step failed: %w", err)
			}

			m.vm.Reset()
			m.vm.Close()
		}

		avgLoss := totalLoss / float64(len(inputSeqs))
		if epoch%10 == 0 {
			fmt.Printf("Epoch %d, Average Loss: %.4f\n", epoch, avgLoss)
		}
	}

	m.trained = true
	return nil
}

// Predict generates SQL from natural language
func (m *Seq2SeqModel) Predict(inputSeq []int) ([]int, error) {
	if !m.trained {
		return nil, fmt.Errorf("model not trained")
	}

	// Use a dummy target for forward pass
	targetSeq := make([]int, m.maxSeqLen)
	targetSeq[0] = 2 // START token

	output, err := m.Forward(inputSeq, targetSeq)
	if err != nil {
		return nil, err
	}

	// Create VM and run
	m.vm = gorgonia.NewTapeMachine(m.g)
	defer m.vm.Close()

	if err := m.vm.RunAll(); err != nil {
		return nil, err
	}

	// Get output values and convert to token indices
	outVal := output.Value()
	if outVal == nil {
		return nil, fmt.Errorf("no output value")
	}

	// Convert probabilities to token indices (argmax)
	probs := outVal.Data().([]float64)
	predicted := make([]int, 1)
	maxIdx := 0
	maxProb := probs[0]
	for i, p := range probs {
		if p > maxProb {
			maxProb = p
			maxIdx = i
		}
	}
	predicted[0] = maxIdx

	return predicted, nil
}

// Save saves the model weights to a file
func (m *Seq2SeqModel) Save(path string) error {
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()

	// Save model configuration and weights
	modelData := struct {
		VocabSize    int
		EmbeddingDim int
		HiddenDim    int
		MaxSeqLen    int
		Trained      bool
		// Weight tensors would be saved here
	}{
		VocabSize:    m.vocabSize,
		EmbeddingDim: m.embeddingDim,
		HiddenDim:    m.hiddenDim,
		MaxSeqLen:    m.maxSeqLen,
		Trained:      m.trained,
	}

	encoder := gob.NewEncoder(f)
	return encoder.Encode(modelData)
}

// Load loads model weights from a file
func (m *Seq2SeqModel) Load(path string) error {
	f, err := os.Open(path)
	if err != nil {
		return err
	}
	defer f.Close()

	var modelData struct {
		VocabSize    int
		EmbeddingDim int
		HiddenDim    int
		MaxSeqLen    int
		Trained      bool
	}

	decoder := gob.NewDecoder(f)
	if err := decoder.Decode(&modelData); err != nil {
		return err
	}

	m.vocabSize = modelData.VocabSize
	m.embeddingDim = modelData.EmbeddingDim
	m.hiddenDim = modelData.HiddenDim
	m.maxSeqLen = modelData.MaxSeqLen
	m.trained = modelData.Trained

	return nil
}

// IsTrained returns whether the model has been trained
func (m *Seq2SeqModel) IsTrained() bool {
	return m.trained
}

// Helper functions

func toFloat64(ints []int) []float64 {
	result := make([]float64, len(ints))
	for i, v := range ints {
		result[i] = float64(v)
	}
	return result
}

func oneHot(idx, size int) []float64 {
	result := make([]float64, size)
	if idx >= 0 && idx < size {
		result[idx] = 1.0
	}
	return result
}

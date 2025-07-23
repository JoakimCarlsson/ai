package llm

import "time"

type parameterBuilder struct {
	temperature   *float64
	topP          *float64
	topK          *float64
	maxTokens     int64
	stopSequences []string
	timeout       *time.Duration
}

func newParameterBuilder(opts llmClientOptions) *parameterBuilder {
	var topK *float64
	if opts.topK != nil {
		f := float64(*opts.topK)
		topK = &f
	}
	return &parameterBuilder{
		temperature:   opts.temperature,
		topP:          opts.topP,
		topK:          topK,
		maxTokens:     opts.maxTokens,
		stopSequences: opts.stopSequences,
		timeout:       opts.timeout,
	}
}

func (p *parameterBuilder) applyFloat32Temperature(setter func(*float32)) {
	if p.temperature != nil {
		temp := float32(*p.temperature)
		setter(&temp)
	}
}

func (p *parameterBuilder) applyFloat32TopP(setter func(*float32)) {
	if p.topP != nil {
		topP := float32(*p.topP)
		setter(&topP)
	}
}

func (p *parameterBuilder) applyFloat32TopK(setter func(*float32)) {
	if p.topK != nil {
		topK := float32(*p.topK)
		setter(&topK)
	}
}

func (p *parameterBuilder) applyInt32Seed(seed *int64, setter func(*int32)) {
	if seed != nil {
		s := int32(*seed)
		setter(&s)
	}
}

func (p *parameterBuilder) applyFloat32FrequencyPenalty(penalty *float64, setter func(*float32)) {
	if penalty != nil {
		fp := float32(*penalty)
		setter(&fp)
	}
}

func (p *parameterBuilder) applyFloat32PresencePenalty(penalty *float64, setter func(*float32)) {
	if penalty != nil {
		pp := float32(*penalty)
		setter(&pp)
	}
}

func (p *parameterBuilder) applyFloat64Temperature(setter func(*float64)) {
	if p.temperature != nil {
		setter(p.temperature)
	}
}

func (p *parameterBuilder) applyFloat64TopP(setter func(*float64)) {
	if p.topP != nil {
		setter(p.topP)
	}
}

func (p *parameterBuilder) applyInt64TopK(setter func(*int64)) {
	if p.topK != nil {
		topK := int64(*p.topK)
		setter(&topK)
	}
}

func (p *parameterBuilder) applyInt64Seed(seed *int64, setter func(*int64)) {
	if seed != nil {
		setter(seed)
	}
}

func (p *parameterBuilder) applyFloat64FrequencyPenalty(penalty *float64, setter func(*float64)) {
	if penalty != nil {
		setter(penalty)
	}
}

func (p *parameterBuilder) applyFloat64PresencePenalty(penalty *float64, setter func(*float64)) {
	if penalty != nil {
		setter(penalty)
	}
}

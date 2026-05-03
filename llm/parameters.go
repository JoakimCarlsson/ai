package llm

// ParameterBuilder helps vendor implementations apply optional sampling parameters
// to provider-specific request types. Construct it once per request via
// [NewParameterBuilder] with raw values; the Apply* methods only call setters
// when the corresponding pointer is non-nil.
type ParameterBuilder struct {
	Temperature *float64
	TopP        *float64
	TopK        *int64
}

// NewParameterBuilder constructs a ParameterBuilder from raw optional values.
// Vendor packages typically pass their Options' temperature/topP/topK fields.
func NewParameterBuilder(temperature, topP *float64, topK *int64) *ParameterBuilder {
	return &ParameterBuilder{
		Temperature: temperature,
		TopP:        topP,
		TopK:        topK,
	}
}

// ApplyFloat32Temperature calls setter with a float32 view of Temperature when non-nil.
func (p *ParameterBuilder) ApplyFloat32Temperature(setter func(*float32)) {
	if p.Temperature != nil {
		temp := float32(*p.Temperature)
		setter(&temp)
	}
}

// ApplyFloat32TopP calls setter with a float32 view of TopP when non-nil.
func (p *ParameterBuilder) ApplyFloat32TopP(setter func(*float32)) {
	if p.TopP != nil {
		topP := float32(*p.TopP)
		setter(&topP)
	}
}

// ApplyFloat32TopK calls setter with a float32 view of TopK when non-nil.
func (p *ParameterBuilder) ApplyFloat32TopK(setter func(*float32)) {
	if p.TopK != nil {
		topK := float32(*p.TopK)
		setter(&topK)
	}
}

// ApplyInt32Seed calls setter with an int32 view of seed when non-nil.
func (p *ParameterBuilder) ApplyInt32Seed(seed *int64, setter func(*int32)) {
	if seed != nil {
		s := int32(*seed)
		setter(&s)
	}
}

// ApplyFloat32FrequencyPenalty calls setter with a float32 view of penalty when non-nil.
func (p *ParameterBuilder) ApplyFloat32FrequencyPenalty(penalty *float64, setter func(*float32)) {
	if penalty != nil {
		fp := float32(*penalty)
		setter(&fp)
	}
}

// ApplyFloat32PresencePenalty calls setter with a float32 view of penalty when non-nil.
func (p *ParameterBuilder) ApplyFloat32PresencePenalty(penalty *float64, setter func(*float32)) {
	if penalty != nil {
		pp := float32(*penalty)
		setter(&pp)
	}
}

// ApplyFloat64Temperature calls setter with the raw float64 Temperature when non-nil.
func (p *ParameterBuilder) ApplyFloat64Temperature(setter func(*float64)) {
	if p.Temperature != nil {
		setter(p.Temperature)
	}
}

// ApplyFloat64TopP calls setter with the raw float64 TopP when non-nil.
func (p *ParameterBuilder) ApplyFloat64TopP(setter func(*float64)) {
	if p.TopP != nil {
		setter(p.TopP)
	}
}

// ApplyInt64TopK calls setter with the int64 TopK when non-nil.
func (p *ParameterBuilder) ApplyInt64TopK(setter func(*int64)) {
	if p.TopK != nil {
		setter(p.TopK)
	}
}

// ApplyInt64Seed calls setter with the raw int64 seed when non-nil.
func (p *ParameterBuilder) ApplyInt64Seed(seed *int64, setter func(*int64)) {
	if seed != nil {
		setter(seed)
	}
}

// ApplyFloat64FrequencyPenalty calls setter with the raw float64 penalty when non-nil.
func (p *ParameterBuilder) ApplyFloat64FrequencyPenalty(penalty *float64, setter func(*float64)) {
	if penalty != nil {
		setter(penalty)
	}
}

// ApplyFloat64PresencePenalty calls setter with the raw float64 penalty when non-nil.
func (p *ParameterBuilder) ApplyFloat64PresencePenalty(penalty *float64, setter func(*float64)) {
	if penalty != nil {
		setter(penalty)
	}
}

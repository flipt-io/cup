package ext

type Document struct {
	Version   string     `yaml:"version,omitempty" json:"version"`
	Namespace string     `yaml:"namespace,omitempty" json:"namespace"`
	Flags     []*Flag    `yaml:"flags,omitempty" json:"flags"`
	Segments  []*Segment `yaml:"segments,omitempty" json:"segments"`
}

type Flag struct {
	Key         string     `yaml:"key,omitempty" json:"key"`
	Name        string     `yaml:"name,omitempty" json:"name"`
	Type        string     `yaml:"type,omitempty" json:"type"`
	Description string     `yaml:"description,omitempty" json:"description"`
	Enabled     bool       `yaml:"enabled" json:"enabled"`
	Variants    []*Variant `yaml:"variants,omitempty" json:"variants"`
	Rules       []*Rule    `yaml:"rules,omitempty" json:"rules"`
	Rollouts    []*Rollout `yaml:"rollouts,omitempty" json:"rollouts"`
}

type Variant struct {
	Key         string `yaml:"key,omitempty" json:"key"`
	Name        string `yaml:"name,omitempty" json:"name"`
	Description string `yaml:"description,omitempty" json:"description"`
	Attachment  any    `yaml:"attachment,omitempty" json:"attachment"`
}

type Rule struct {
	SegmentKey    string          `yaml:"segment,omitempty" json:"segment_key"`
	Rank          uint            `yaml:"rank,omitempty" json:"rank"`
	Distributions []*Distribution `yaml:"distributions,omitempty" json:"distributions"`
}

type Distribution struct {
	VariantKey string  `yaml:"variant,omitempty" json:"variant_key"`
	Rollout    float32 `yaml:"rollout,omitempty" json:"rollout"`
}

type Rollout struct {
	Description string         `yaml:"description,omitempty" json:"description"`
	Segment     *SegmentRule   `yaml:"segment,omitempty" json:"segment"`
	Threshold   *ThresholdRule `yaml:"threshold,omitempty" json:"threshold"`
}

type SegmentRule struct {
	Key   string `yaml:"key,omitempty" json:"key"`
	Value bool   `yaml:"value,omitempty" json:"value"`
}

type ThresholdRule struct {
	Percentage float32 `yaml:"percentage,omitempty" json:"percentage"`
	Value      bool    `yaml:"value,omitempty" json:"value"`
}

type Segment struct {
	Key         string        `yaml:"key,omitempty" json:"key"`
	Name        string        `yaml:"name,omitempty" json:"name"`
	Description string        `yaml:"description,omitempty" json:"description"`
	Constraints []*Constraint `yaml:"constraints,omitempty" json:"constraints"`
	MatchType   string        `yaml:"match_type,omitempty" json:"match_type"`
}

type Constraint struct {
	Type        string `yaml:"type,omitempty" json:"type"`
	Property    string `yaml:"property,omitempty" json:"property"`
	Operator    string `yaml:"operator,omitempty" json:"operator"`
	Value       string `yaml:"value,omitempty" json:"value"`
	Description string `yaml:"description,omitempty" json:"description"`
}

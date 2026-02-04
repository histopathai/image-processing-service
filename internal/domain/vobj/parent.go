package vobj

type ParentType string

func (p ParentType) IsValid() bool {
	switch p {
	case ParentTypeWorkspace, ParentTypePatient, ParentTypeImage, ParentTypeAnnotationType, ParentTypeNone:
		return true
	default:
		return false
	}
}

func (p ParentType) String() string {
	return string(p)
}

type ParentRef struct {
	ID   string     `json:"id"`
	Type ParentType `json:"type"`
}

func (p *ParentRef) IsValid() bool {
	return p != nil && p.ID != "" && p.Type != ""
}

func (p *ParentRef) Equals(other *ParentRef) bool {
	if p == nil && other == nil {
		return true
	}
	if p == nil || other == nil {
		return false
	}
	return p.ID == other.ID && p.Type == other.Type
}

func (p *ParentRef) Copy() *ParentRef {
	if p == nil {
		return nil
	}
	return &ParentRef{
		ID:   p.ID,
		Type: p.Type,
	}
}

func (p *ParentRef) String() string {
	if p == nil {
		return "<nil>"
	}
	return string(p.Type) + ":" + p.ID
}

func (p *ParentRef) IsEmpty() bool {
	return p == nil || p.ID == "" || p.Type == ""
}

func (p *ParentRef) GetID() string {
	if p == nil {
		return ""
	}
	return p.ID
}

func (p *ParentRef) GetType() ParentType {
	if p == nil {
		return ""
	}
	return p.Type
}

func (p *ParentRef) GetMap() map[string]interface{} {
	return map[string]interface{}{
		"ID":   p.ID,
		"Type": p.Type.String(),
	}
}

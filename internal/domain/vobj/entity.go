package vobj

import (
	"time"
)

type EntityType string

type Entity struct {
	ID         string
	EntityType EntityType
	Name       string
	CreatorID  string
	Parent     ParentRef
	Deleted    bool
	CreatedAt  time.Time
	UpdatedAt  time.Time
}

func (e EntityType) IsValid() bool {
	switch e {
	case EntityTypeImage, EntityTypeAnnotation, EntityTypePatient, EntityTypeWorkspace, EntityTypeAnnotationType:
		return true
	default:
		return false
	}
}

func (e EntityType) String() string {
	return string(e)
}

// Getter metodları - value receiver (embedded struct'lar için çalışır)
func (e Entity) GetID() string {
	return e.ID
}

func (e Entity) GetCreatorID() string {
	return e.CreatorID
}

func (e Entity) GetName() string {
	return e.Name
}

func (e Entity) GetEntityType() EntityType {
	return e.EntityType
}

func (e Entity) GetCreatedAt() time.Time {
	return e.CreatedAt
}

func (e Entity) GetUpdatedAt() time.Time {
	return e.UpdatedAt
}

func (e Entity) GetParent() *ParentRef {
	return &e.Parent
}

func (e Entity) HasParent() bool {
	return e.Parent.Type != ParentTypeNone && e.Parent.ID != ""
}

func (e *Entity) SetID(id string) {
	e.ID = id
}

func (e *Entity) SetCreatorID(creatorID string) {
	e.CreatorID = creatorID
}

func (e *Entity) SetName(name string) {
	e.Name = name
}

func (e *Entity) SetEntityType(entityType EntityType) {
	e.EntityType = entityType
}

func (e *Entity) SetCreatedAt(t time.Time) {
	e.CreatedAt = t
}

func (e *Entity) SetUpdatedAt(t time.Time) {
	e.UpdatedAt = t
}

func (e *Entity) SetParent(parent *ParentRef) {
	e.Parent = *parent
}

func (e *Entity) IsDeleted() bool {
	return e.Deleted
}

func (e *Entity) SetDeleted(deleted bool) {
	e.Deleted = deleted
}

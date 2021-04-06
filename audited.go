package audited

import (
	"fmt"
)

// AuditedModel make Model Auditable, embed `audited.AuditedModel` into your model as anonymous field to make the model auditable
//    type User struct {
//      gorm.Model
//      audited.AuditedModel
//    }
type AuditedModel struct {
	CreatedBy *string `gorm:"not null"`
	UpdatedBy *string
}

// SetCreatedBy set created by
func (model *AuditedModel) SetCreatedBy(createdBy interface{}) {
	s := fmt.Sprintf("%v", createdBy)
	model.CreatedBy = &s
}

// GetCreatedBy get created by
func (model AuditedModel) GetCreatedBy() *string {
	return model.CreatedBy
}

// SetUpdatedBy set updated by
func (model *AuditedModel) SetUpdatedBy(updatedBy interface{}) {
	s := fmt.Sprintf("%v", updatedBy)
	model.UpdatedBy = &s
}

// GetUpdatedBy get updated by
func (model AuditedModel) GetUpdatedBy() *string {
	return model.UpdatedBy
}

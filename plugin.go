package audited

import (
	"reflect"

	"gorm.io/gorm"
)

type auditableInterface interface {
	SetCreatedBy(createdBy interface{})
	GetCreatedBy() *string
	SetUpdatedBy(updatedBy interface{})
	GetUpdatedBy() *string
}

func isAuditable(db *gorm.DB) (isAuditable bool) {
	if db.Statement.Schema == nil {
		return false
	}
	if db.Statement.Schema.ModelType == nil {
		return false
	}

	_, isAuditable = reflect.New(db.Statement.Schema.ModelType).Interface().(auditableInterface)
	return
}

func assignCreatedBy(db *gorm.DB) {
	if isAuditable(db) {
		if user, ok := db.Get("audited:current_user"); ok {
			db.Statement.SetColumn("created_by", user, true)
		}
	}
}

func assignUpdatedBy(db *gorm.DB) {
	if isAuditable(db) {
		if user, ok := db.Get("audited:current_user"); ok {
			db.Statement.SetColumn("updated_by", user, true)
		}
	}
}

type Plugin struct{}

func (p *Plugin) Name() string {
	return "audited"
}

func (p *Plugin) Initialize(db *gorm.DB) error {
	var err error
	callback := db.Callback()
	if callback.Create().Get("audited:assign_created_by") == nil {
		err = callback.Create().Before("gorm:before_create").Register("audited:assign_created_by", assignCreatedBy)
		if err != nil {
			return err
		}
	}
	if callback.Update().Get("audited:assign_updated_by") == nil {
		err = callback.Update().Before("gorm:before_update").Register("audited:assign_updated_by", assignUpdatedBy)
		if err != nil {
			return err
		}
	}
	return err
}

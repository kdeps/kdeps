package resolver

import (
	"encoding/json"
	"fmt"
	"reflect"
	"time"

	"github.com/apple/pkl-go/pkl"
	pklResource "github.com/kdeps/schema/gen/resource"
)

// ResourceHandler provides a generic interface for handling different resource types
type ResourceHandler[T any] struct {
	dr           *DependencyResolver
	resourceType string
}

// NewResourceHandler creates a new generic resource handler
func NewResourceHandler[T any](dr *DependencyResolver, resourceType string) *ResourceHandler[T] {
	return &ResourceHandler[T]{
		dr:           dr,
		resourceType: resourceType,
	}
}

// HandleResource provides generic resource handling with PKL reloading and comprehensive storage
func (rh *ResourceHandler[T]) HandleResource(actionID string, resource *T, processor func(*T) error) error {
	rh.dr.Logger.Info("HandleResource: ENTRY", "actionID", actionID, "resourceType", rh.resourceType, "resource_nil", resource == nil)

	// Canonicalize the actionID
	canonicalActionID := actionID
	if rh.dr.PklresHelper != nil {
		canonicalActionID = rh.dr.PklresHelper.resolveActionID(actionID)
		if canonicalActionID != actionID {
			rh.dr.Logger.Debug("canonicalized actionID", "original", actionID, "canonical", canonicalActionID)
		}
	}

	// Reload resource with fresh PKL template evaluation
	if err := rh.reloadResourceWithDependencies(canonicalActionID, resource); err != nil {
		rh.dr.Logger.Warn("failed to reload resource, continuing with original", "actionID", canonicalActionID, "resourceType", rh.resourceType, "error", err)
	}

	// Process the resource
	if err := processor(resource); err != nil {
		rh.dr.Logger.Error("failed to process resource", "actionID", canonicalActionID, "resourceType", rh.resourceType, "error", err)
		return err
	}

	// Store comprehensive attributes in pklres
	if err := rh.storeResourceAttributes(canonicalActionID, resource); err != nil {
		rh.dr.Logger.Warn("failed to store comprehensive resource attributes", "actionID", canonicalActionID, "resourceType", rh.resourceType, "error", err)
	}

	return nil
}

// reloadResourceWithDependencies generically reloads any resource type
func (rh *ResourceHandler[T]) reloadResourceWithDependencies(actionID string, resource *T) error {
	rh.dr.Logger.Debug("reloadResourceWithDependencies: reloading resource for fresh template evaluation", "actionID", actionID, "resourceType", rh.resourceType)

	// Find the resource file path for this actionID
	resourceFile := ""
	for _, resInterface := range rh.dr.Resources {
		if res, ok := resInterface.(ResourceNodeEntry); ok {
			if res.ActionID == actionID {
				resourceFile = res.File
				break
			}
		}
	}

	if resourceFile == "" {
		return fmt.Errorf("could not find resource file for actionID: %s", actionID)
	}

	// Reload the resource with fresh PKL template evaluation
	var reloadedResource interface{}
	var err error
	if rh.dr.APIServerMode {
		reloadedResource, err = rh.dr.LoadResourceWithRequestContextFn(rh.dr.Context, resourceFile, Resource)
	} else {
		reloadedResource, err = rh.dr.LoadResourceFn(rh.dr.Context, resourceFile, Resource)
	}

	if err != nil {
		return fmt.Errorf("failed to reload resource: %w", err)
	}

	// Cast to generic Resource and extract the specific resource block
	reloadedGenericResource, ok := reloadedResource.(*pklResource.Resource)
	if !ok {
		return fmt.Errorf("failed to cast reloaded resource to generic Resource")
	}

	// Update resource with reloaded values (implementation varies by resource type)
	rh.updateResourceFromReloaded(resource, reloadedGenericResource)

	rh.dr.Logger.Info("reloadResourceWithDependencies: successfully reloaded resource with fresh template evaluation", "actionID", actionID, "resourceType", rh.resourceType)
	return nil
}

// updateResourceFromReloaded updates the resource with reloaded values (to be implemented by specific handlers)
func (rh *ResourceHandler[T]) updateResourceFromReloaded(resource *T, reloadedGenericResource *pklResource.Resource) {
	// This will be overridden by specific resource handlers
	rh.dr.Logger.Debug("updateResourceFromReloaded: generic implementation, no updates performed", "resourceType", rh.resourceType)
}

// storeResourceAttributes generically stores all resource attributes in pklres
func (rh *ResourceHandler[T]) storeResourceAttributes(actionID string, resource *T) error {
	if rh.dr.PklresHelper == nil {
		return nil
	}

	// Use reflection to store all struct fields
	v := reflect.ValueOf(resource).Elem()
	t := v.Type()

	for i := 0; i < v.NumField(); i++ {
		field := v.Field(i)
		fieldType := t.Field(i)
		fieldName := rh.convertFieldNameToKey(fieldType.Name)

		if !field.IsValid() || !field.CanInterface() {
			continue
		}

		// Handle different field types
		switch field.Kind() {
		case reflect.String:
			if str := field.String(); str != "" {
				if err := rh.dr.PklresHelper.Set(actionID, fieldName, str); err != nil {
					rh.dr.Logger.Error("failed to store string field", "actionID", actionID, "field", fieldName, "error", err)
				}
			}

		case reflect.Ptr:
			if !field.IsNil() {
				elem := field.Elem()
				switch elem.Kind() {
				case reflect.String:
					if str := elem.String(); str != "" {
						if err := rh.dr.PklresHelper.Set(actionID, fieldName, str); err != nil {
							rh.dr.Logger.Error("failed to store string pointer field", "actionID", actionID, "field", fieldName, "error", err)
						}
					}
				case reflect.Int, reflect.Int32, reflect.Int64:
					intVal := elem.Int()
					if err := rh.dr.PklresHelper.Set(actionID, fieldName, fmt.Sprintf("%d", intVal)); err != nil {
						rh.dr.Logger.Error("failed to store int field", "actionID", actionID, "field", fieldName, "error", err)
					}
				case reflect.Bool:
					boolVal := elem.Bool()
					boolStr := "false"
					if boolVal {
						boolStr = "true"
					}
					if err := rh.dr.PklresHelper.Set(actionID, fieldName, boolStr); err != nil {
						rh.dr.Logger.Error("failed to store bool field", "actionID", actionID, "field", fieldName, "error", err)
					}
				case reflect.Slice, reflect.Map, reflect.Struct:
					// JSON encode complex structures
					if jsonData, err := json.Marshal(elem.Interface()); err == nil {
						if err := rh.dr.PklresHelper.Set(actionID, fieldName, string(jsonData)); err != nil {
							rh.dr.Logger.Error("failed to store complex field", "actionID", actionID, "field", fieldName, "error", err)
						}
					} else {
						rh.dr.Logger.Error("failed to marshal complex field", "actionID", actionID, "field", fieldName, "error", err)
					}
				case reflect.Float64:
					// Handle pkl.Duration and other float fields
					floatVal := elem.Float()
					if err := rh.dr.PklresHelper.Set(actionID, fieldName, fmt.Sprintf("%g", floatVal)); err != nil {
						rh.dr.Logger.Error("failed to store float field", "actionID", actionID, "field", fieldName, "error", err)
					}
				}
			}

		case reflect.Slice, reflect.Map:
			// JSON encode complex structures
			if field.Len() > 0 {
				if jsonData, err := json.Marshal(field.Interface()); err == nil {
					if err := rh.dr.PklresHelper.Set(actionID, fieldName, string(jsonData)); err != nil {
						rh.dr.Logger.Error("failed to store complex field", "actionID", actionID, "field", fieldName, "error", err)
					}
				} else {
					rh.dr.Logger.Error("failed to marshal complex field", "actionID", actionID, "field", fieldName, "error", err)
				}
			}
		}
	}

	rh.dr.Logger.Info("stored comprehensive resource attributes in pklres", "actionID", actionID, "resourceType", rh.resourceType)
	return nil
}

// convertFieldNameToKey converts Go struct field names to pklres keys (camelCase)
func (rh *ResourceHandler[T]) convertFieldNameToKey(fieldName string) string {
	if len(fieldName) == 0 {
		return fieldName
	}

	// Convert PascalCase to camelCase
	runes := []rune(fieldName)
	runes[0] = runes[0] + 32 // Convert first character to lowercase
	return string(runes)
}

// SetTimestamp sets a timestamp on the resource using reflection
func (rh *ResourceHandler[T]) SetTimestamp(resource *T) {
	v := reflect.ValueOf(resource).Elem()
	timestampField := v.FieldByName("Timestamp")

	if timestampField.IsValid() && timestampField.CanSet() && timestampField.Kind() == reflect.Ptr {
		ts := pkl.Duration{
			Value: float64(time.Now().UnixNano()),
			Unit:  pkl.Nanosecond,
		}
		timestampField.Set(reflect.ValueOf(&ts))
	}
}

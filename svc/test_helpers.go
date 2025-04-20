package svc

import (
	"log"
	"reflect"
)

// SetTestFields sets private fields on an OptimizedDialecticService instance for testing purposes.
// This allows tests to inject mock instances of dependencies.
func SetTestFields(
	service *OptimizedDialecticService,
	kvStore interface{},
	aiHelper AIHelperInterface,
	dialecticEpiSvc interface{},
	enablePredictiveProcessing bool,
) {
	if service == nil {
		log.Printf("ERROR: service is nil in SetTestFields")
		return
	}

	// Use reflection to set the private fields
	serviceValue := reflect.ValueOf(service).Elem()

	// Set the kvStore field
	kvStoreField := serviceValue.FieldByName("kvStore")
	if kvStoreField.IsValid() && kvStoreField.CanSet() {
		kvStoreField.Set(reflect.ValueOf(kvStore))
		log.Printf("Successfully set kvStore field")
	} else {
		log.Printf("ERROR: Failed to set kvStore field. Valid: %v, CanSet: %v",
			kvStoreField.IsValid(), kvStoreField.CanSet())
	}

	// Set the aiHelper field
	aiHelperField := serviceValue.FieldByName("aiHelper")
	if aiHelperField.IsValid() && aiHelperField.CanSet() {
		aiHelperField.Set(reflect.ValueOf(aiHelper))
		log.Printf("Successfully set aiHelper field")
	} else {
		log.Printf("ERROR: Failed to set aiHelper field. Valid: %v, CanSet: %v",
			aiHelperField.IsValid(), aiHelperField.CanSet())
	}

	// Set the dialecticEpiSvc field
	dialecticEpiSvcField := serviceValue.FieldByName("dialecticEpiSvc")
	if dialecticEpiSvcField.IsValid() && dialecticEpiSvcField.CanSet() {
		dialecticEpiSvcField.Set(reflect.ValueOf(dialecticEpiSvc))
		log.Printf("Successfully set dialecticEpiSvc field")
	} else {
		log.Printf("ERROR: Failed to set dialecticEpiSvc field. Valid: %v, CanSet: %v",
			dialecticEpiSvcField.IsValid(), dialecticEpiSvcField.CanSet())
	}

	// Set the enablePredictiveProcessing field
	enablePPField := serviceValue.FieldByName("enablePredictiveProcessing")
	if enablePPField.IsValid() && enablePPField.CanSet() {
		enablePPField.SetBool(enablePredictiveProcessing)
		log.Printf("Successfully set enablePredictiveProcessing field")
	} else {
		log.Printf("ERROR: Failed to set enablePredictiveProcessing field. Valid: %v, CanSet: %v",
			enablePPField.IsValid(), enablePPField.CanSet())
	}

	// Print a summary of the service fields after setting
	log.Printf("OptimizedDialecticService fields after SetTestFields:")
	log.Printf("kvStore: %v", service.kvStore)
	log.Printf("aiHelper: %v", service.aiHelper)
	log.Printf("dialecticEpiSvc: %v", service.dialecticEpiSvc)
	log.Printf("enablePredictiveProcessing: %v", service.enablePredictiveProcessing)
}

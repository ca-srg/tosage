package repository

// ModelAvailability defines which models are available in which regions
// Based on actual testing and Google Cloud documentation
var ModelAvailability = map[string][]string{
	// Asia regions
	"asia-northeast1": {
		"gemini-2.5-flash",
		"gemini-1.5-pro", 
		"gemini-1.5-flash",
		"gemini-pro",
	},
	"asia-northeast2": {
		// Limited availability - no models currently available
	},
	"asia-northeast3": {
		"gemini-1.5-pro",
		"gemini-1.5-flash", 
		"gemini-pro",
	},
	"asia-southeast1": {
		"gemini-2.5-flash",
		"gemini-1.5-pro",
		"gemini-1.5-flash",
		"gemini-pro",
	},
	
	// US regions  
	"us-central1": {
		"gemini-2.5-pro",  // May be available in US regions
		"gemini-2.5-flash",
		"gemini-2.0-flash",
		"gemini-2.0-flash-lite",
		"gemini-1.5-pro",
		"gemini-1.5-flash",
		"gemini-pro",
	},
	"us-east1": {
		"gemini-2.5-flash",
		"gemini-1.5-pro", 
		"gemini-1.5-flash",
		"gemini-pro",
	},
	"us-west1": {
		"gemini-2.5-flash",
		"gemini-1.5-pro",
		"gemini-1.5-flash", 
		"gemini-pro",
	},
	
	// Europe regions
	"europe-west1": {
		"gemini-2.5-flash",
		"gemini-1.5-pro",
		"gemini-1.5-flash",
		"gemini-pro",
	},
	"europe-west4": {
		"gemini-2.5-flash",
		"gemini-1.5-pro", 
		"gemini-1.5-flash",
		"gemini-pro",
	},
}

// GetAvailableModelsForLocation returns models available in a specific location
func GetAvailableModelsForLocation(location string) []string {
	if models, ok := ModelAvailability[location]; ok {
		return models
	}
	
	// Default fallback models if location not in map
	return []string{
		"gemini-1.5-pro",
		"gemini-1.5-flash", 
		"gemini-pro",
	}
}
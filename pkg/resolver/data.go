package resolver

// AppendDataEntry has been removed as it's no longer needed.
// We now use real-time pklres access through getResourceOutput() instead of storing PKL content.
// Data resources are now handled differently - they don't store PKL content but rather
// manage file references directly through the agent system.

// Exported for testing
var (
	FormatValue     = formatValue
	FormatErrors    = formatErrors
	FormatDataValue = formatDataValue
)

package codes

// ErrorCodes maps Crestron SIMPL+ compiler exit codes to their descriptions
var ErrorCodes = map[int]string{
	0:   "Success",
	100: "General failure",
	101: "Cannot open module",
	102: "Cannot create new makefile",
	103: "Cannot create new globals.h",
	104: "Cannot open master makefile",
	105: "Cannot write new makefile",
	106: "Compile errors",
	107: "Link errors",
	108: "Cannot copy output file to LinkMakeFileDir",
	109: "Cannot copy gnu files to LinkMakeFileDir",
	110: "Cannot launch gnu compiler",
	111: "Cannot retrieve total number of NVRam in module",
	112: "GNU not installed",
	113: "System.CodeDom.Compiler wrapper can not be instantiated",
	114: "The CompilerObj instance is invalid",
	115: "Invalid compiler results object",
	116: "The system.CodeDom.Compiler finished successfully, but with errors",
	117: "CompilerResult class is NULL or there is a problem with it",
	118: "Error extracting reference files from Include.dat",
	119: ".cs file doesn't exist",
	120: "Error launching NVRAM utility",
	121: "NVRAM Utility ran - no output generated",
	122: "Error saving temporary certificate file",
	123: "Invalid writer object",
	124: "Unable to create temporary file",
	125: "Unable to translate the certificate hex string to byte array",
	126: "Signing process failed",
	127: "Cleanup of the temp file failed",
	128: "The registry key for signing assemblies does not exist.",
	129: "CAPICOM is either not installed properly or not registered!",
	130: "Error found while signing. Unable to cleanup unsigned assembly.",
}

// IsSuccess returns true if the exit code indicates successful compilation
func IsSuccess(code int) bool {
	return code == 0 || code == 116
}

// GetErrorMessage returns the error message for a given exit code, or a generic message if unknown
func GetErrorMessage(code int) string {
	if msg, ok := ErrorCodes[code]; ok {
		return msg
	}

	return "Unknown error"
}

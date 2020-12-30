package config

// consts for each config field.
// These are used in tests to verify error messages match json/toml field names
// to ensure error messages are not misleading
const (
	nameField                   = "name"
	controlTypeField            = "controlType"
	containerIdField            = "containerId"
	stopCommandField            = "stopCommand"
	startCommandField           = "startCommand"
	upcheckConfigField          = "upcheckConfig"
	typeField                   = "type"
	proxyAddressField           = "proxyAddress"
	upstreamAddressField        = "upstreamAddress"
	proxyPathsField             = "proxyPaths"
	ignorePathsForActivityField = "ignorePathsForActivity"
	readTimeoutField            = "readTimeout"
	writeTimeoutField           = "writeTimeout"
	proxyTlsConfigField         = "proxyTlsConfig"
	clientTlsConfigField        = "clientTlsConfig"
	rpcAddressField             = "rpcAddress"
	rpcCorsListField            = "rpcCorsList"
	rpcvHostsField              = "rpcvHosts"
	tlsConfigField              = "tlsConfig"
	keyFileField                = "keyFile"
	certificateFileField        = "certificateFile"
	clientCaCertificateField    = "clientCaCertificateFile"
	caCertificateFileField      = "caCertificateFile"
	insecureSkipVerifyField     = "insecureSkipVerify"
	urlField                    = "url"
	returnTypeField             = "returnType"
	methodField                 = "method"
	bodyField                   = "body"
	expectedField               = "expected"
)

package config

// consts for each config field.
// These are used in tests to verify error messages match json/toml field names
// to ensure error messages are not misleading
const (
	nameField                   = "name"
	disableStrictModeField      = "disableStrictMode"
	upcheckPollingIntervalField = "upcheckPollingInterval"
	peersConfigFileField        = "peersConfigFile"
	inactivityTimeField         = "inactivityTime"
	resyncTimeField             = "resyncTime"
	blockchainClientField       = "blockchainClient"
	privacyManagerField         = "privacyManager"
	serverField                 = "server"
	proxiesField                = "proxies"
	typeField                   = "type"
	consensusField              = "consensus"
	rpcUrlField                 = "rpcUrl"
	tlsConfigField              = "tlsConfig"
	processField                = "process"
	publicKeyField              = "publicKey"
	controlTypeField            = "controlType"
	containerIdField            = "containerId"
	stopCommandField            = "stopCommand"
	startCommandField           = "startCommand"
	upcheckConfigField          = "upcheckConfig"
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

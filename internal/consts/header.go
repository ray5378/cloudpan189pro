package consts

const (
	HeaderKeyTransferType            = "X-Transfer-Type"
	HeaderKeyTransferChunkSize       = "X-Transfer-Chunk-Size"
	HeaderKeyTransferChunkSizeFormat = "X-Transfer-Chunk-Size-Format"
	HeaderKeyTransferThreadCount     = "X-Transfer-Thread-Count"

	HeaderValueTransferTypeRedirect    = "redirect"
	HeaderValueTransferTypeMultiStream = "multi_stream"
	HeaderValueTransferTypeLocalProxy  = "local_proxy"
)

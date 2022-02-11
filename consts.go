package main

const (
	HeaderBytesQuantity         = 2
	CRCBytesQuantity            = 4
	CRCPaddingBytesQuantity     = 16
	MetadataLengthBytesQuantity = 2
	FileMetaBytesQuantity       = HeaderBytesQuantity + CRCBytesQuantity + CRCPaddingBytesQuantity + MetadataLengthBytesQuantity
	MinRequiredFileLength       = FileMetaBytesQuantity + 1

	MetadataStart = HeaderBytesQuantity + CRCBytesQuantity + CRCPaddingBytesQuantity
)

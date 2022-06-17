package main

const (
	HeaderBytesQuantity         = 2
	CRCBytesQuantity            = 4
	CRCPaddingBytesQuantity     = 16
	MetadataLengthBytesQuantity = 2
	MetadataHeader              = 4
	FileMetaBytesQuantity       = HeaderBytesQuantity + CRCBytesQuantity + CRCPaddingBytesQuantity + MetadataLengthBytesQuantity + MetadataHeader
	MinRequiredFileLength       = FileMetaBytesQuantity + 1

	MetadataStart = HeaderBytesQuantity + CRCBytesQuantity + CRCPaddingBytesQuantity
)

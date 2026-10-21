package providers

const Sybill Provider = "sybill"

func init() {
	SetInfo(Sybill, ProviderInfo{
		DisplayName: "Sybill",
		AuthType:    ApiKey,
		// API paths are versioned under /v1, e.g. https://api.sybill.ai/v1/conversations.
		BaseURL: "https://api.sybill.ai",
		ApiKeyOpts: &ApiKeyOpts{
			AttachmentType: Header,
			Header: &ApiKeyOptsHeader{
				Name:        "Authorization",
				ValuePrefix: "Bearer ",
			},
			DocsURL: "https://api.sybill.ai/docs/authentication.html",
		},
		Support: Support{
			BulkWrite: BulkWriteSupport{
				Insert: false,
				Update: false,
				Upsert: false,
				Delete: false,
			},
			Proxy:     false,
			Read:      false,
			Subscribe: false,
			Write:     false,
		},
	})
}

package providers

const Circleback Provider = "circleback"

func init() {
	SetInfo(Circleback, ProviderInfo{
		DisplayName: "Circleback",
		AuthType:    ApiKey,
		// The docs' base URL and the OpenAPI spec's server, e.g. https://circleback.ai/api/meetings.
		// https://circleback.ai/docs/api#base-url
		BaseURL: "https://circleback.ai/api",
		ApiKeyOpts: &ApiKeyOpts{
			AttachmentType: Header,
			Header: &ApiKeyOptsHeader{
				Name:        "Authorization",
				ValuePrefix: "Bearer ",
			},
			DocsURL: "https://circleback.ai/docs/api#get-an-api-key",
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

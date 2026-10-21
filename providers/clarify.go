package providers

const Clarify Provider = "clarify"

func init() {
	SetInfo(Clarify, ProviderInfo{
		DisplayName: "Clarify",
		AuthType:    ApiKey,
		// API paths are versioned and workspace-scoped, e.g.
		// https://api.clarify.ai/v1/workspaces/{workspace}/objects/person/resources.
		// The workspace slug sits after the version segment, so the deep connector adds it in code
		// and proxy callers put it in their own path.
		BaseURL: "https://api.clarify.ai",
		ApiKeyOpts: &ApiKeyOpts{
			AttachmentType: Header,
			Header: &ApiKeyOptsHeader{
				Name:        "Authorization",
				ValuePrefix: "api-key ",
			},
			DocsURL: "https://developer.clarify.ai/docs/getting-started/authentication#api-keys",
		},
		Metadata: &ProviderMetadata{
			Input: []MetadataItemInput{
				{
					Name:        "workspace",
					DisplayName: "Workspace Slug",
					Prompt:      "The slug in your Clarify login URL, also shown under the user avatar menu.",
					DocsURL:     "https://developer.clarify.ai/docs/getting-started/key-concepts#workspace",
				},
			},
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

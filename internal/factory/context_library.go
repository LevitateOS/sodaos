package factory

// CurrentLibraryContext names the admission's collection/issue grant and
// complete coherent inventory pages. Each page preserves the canonical
// inventory's revision and relationship analysis; removed records stay visible.
type CurrentLibraryContext struct {
	Scope      string             `json:"scope"`
	Operations []MCPTool          `json:"operations"`
	Inventory  []LibraryInventory `json:"inventory"`
	Complete   bool               `json:"complete"`
}

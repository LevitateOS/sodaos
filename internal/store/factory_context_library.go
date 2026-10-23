package store

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"

	"github.com/levitateos/sodaos/internal/factory"
)

func currentLibraryTx(ctx context.Context, tx *sql.Tx, a factory.Assignment, out *factory.CurrentContext) error {
	// Reuse the actual packet's explicit issue/collection admission owner.
	// A repository association alone never becomes a collection grant.
	if err := validateAttemptPacketTx(ctx, tx, a, false); err != nil {
		return err
	}
	current := &factory.CurrentLibraryContext{Scope: a.Subject.SubjectKey(a.Repository), Operations: []factory.MCPTool{}, Inventory: []factory.LibraryInventory{}}
	for _, tool := range factory.MCPDiscovery(factory.MCPProfileLibraryAdmin) {
		if factory.MCPIsLibraryMutation(tool) {
			current.Operations = append(current.Operations, tool)
		}
	}
	cursor, count := "", 0
	for {
		page, err := libraryInventory(ctx, tx, a.Repository, "", cursor, true)
		if err != nil {
			return err
		}
		count += len(page.Items)
		current.Inventory = append(current.Inventory, page)
		raw, err := json.Marshal(current)
		if err != nil {
			return err
		}
		if page.Total > 512 || count > 512 || len(current.Inventory) > 64 || len(raw) > 1024*1024 {
			out.Evidence.MissingReasons = append(out.Evidence.MissingReasons, "required_current_collection_bound")
			return nil
		}
		if page.Complete {
			current.Complete = true
			out.Role.Library = current
			return nil
		}
		if page.NextCursor == "" || page.NextCursor <= cursor {
			return errors.New("current library inventory is incomplete")
		}
		cursor = page.NextCursor
	}
}

package store

import (
	"context"

	"github.com/levitateos/sodaos/internal/project"
)

// ProjectMaintenanceWork returns bounded global counts of active or uncertain
// managed work. The sentinel 129 means at least 129, never a complete larger
// count. A preparation only ceases to block after confirmed readiness or stop;
// missing observations and interrupted/failed setup remain explicit blockers.
func (s *Store) ProjectMaintenanceWork(ctx context.Context) (factory, preparation int, err error) {
	err = s.db.QueryRowContext(ctx, `SELECT
		(SELECT count(*) FROM (SELECT 1 FROM factory_runs WHERE active OR NOT settled LIMIT 129) work),
		(SELECT count(*) FROM (SELECT 1 FROM project_preparations WHERE (
			data#>>'{state,id}'=id AND data#>>'{state,project}'=project_id AND data#>>'{state,role}'=role AND (
			(data#>>'{state,phase}'=$1 AND data#>>'{state,ready}'='true' AND data#>>'{state,stopped}'='false')
			OR (data#>>'{state,phase}'=$2 AND data#>>'{state,stopped}'='true' AND data#>>'{state,ready}'='false' AND data#>>'{state,retirement}'='confirmed'))
		) IS NOT TRUE LIMIT 129) work)`, project.PrepareReady, project.PrepareStopped).Scan(&factory, &preparation)
	return factory, preparation, err
}

package store

import "context"

type Member struct {
	UserID int64
	Login  string
}

func (s *Store) Members(ctx context.Context, project string) ([]Member, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT user_id,login FROM memberships WHERE project_id=? ORDER BY login`, project)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	members := []Member{}
	for rows.Next() {
		var member Member
		if err = rows.Scan(&member.UserID, &member.Login); err != nil {
			return nil, err
		}
		members = append(members, member)
	}
	return members, rows.Err()
}

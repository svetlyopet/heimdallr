package fleetcompliance

import "strings"

// ServerCompliantCondition returns SQL that evaluates to true when the server
// identified by serverIDColumn has every required agent installed.
func ServerCompliantCondition(serverIDColumn string) string {
	return `
		NOT EXISTS (
			SELECT 1
			FROM required_agents ra
			WHERE ra.deleted_at IS NULL
				AND NOT EXISTS (
					SELECT 1
					FROM server_agents sa
					INNER JOIN agents a ON a.id = sa.agent_id AND a.deleted_at IS NULL
					WHERE sa.server_id = ` + serverIDColumn + `
						AND a.name = ra.agent_name
				)
		)
	`
}

// ServerCompliantSelectExpr returns a SQL expression that evaluates compliance
// for the server identified by serverIDColumn, treating all servers as compliant
// when no required agents are configured.
func ServerCompliantSelectExpr(serverIDColumn string) string {
	return `CASE
		WHEN (SELECT COUNT(*) FROM required_agents WHERE deleted_at IS NULL) = 0 THEN TRUE
		ELSE (` + strings.TrimSpace(ServerCompliantCondition(serverIDColumn)) + `)
	END`
}

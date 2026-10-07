content = open("backend/internal/infrastructure/meta/graph_connector.go").read()

patch = """g.sessions[pending.AccountID] = &graphSession{UserToken: token.AccessToken, TokenExpiresAt: debug.Data.ExpiresAt, DataAccessExpiresAt: debug.Data.DataAccessExpiresAt, GrantedPermissions: append([]string(nil), debug.Data.Scopes...), Pages: pages}
		if g.db != nil {
			g.db.Save(&database.MetaConnectionModel{AccountID: pending.AccountID, UserToken: token.AccessToken})
		}"""

content = content.replace("g.sessions[pending.AccountID] = &graphSession{UserToken: token.AccessToken, TokenExpiresAt: debug.Data.ExpiresAt, DataAccessExpiresAt: debug.Data.DataAccessExpiresAt, GrantedPermissions: append([]string(nil), debug.Data.Scopes...), Pages: pages}", patch)

open("backend/internal/infrastructure/meta/graph_connector.go", "w").write(content)
